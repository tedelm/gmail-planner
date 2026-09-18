package digest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tedelm/gmail-planner/internal/calendar"
	"github.com/tedelm/gmail-planner/internal/config"
	"github.com/tedelm/gmail-planner/internal/gmail"
	"github.com/tedelm/gmail-planner/internal/logging"
)

const openAIChatURL = "https://api.openai.com/v1/chat/completions"

// Summarizer calls the OpenAI Chat Completions API.
type Summarizer struct {
	apiKey string
	model  string
	hc     *http.Client
}

// DigestOutput is the structured digest returned from OpenAI.
type DigestOutput struct {
	Text string `json:"text"`
	HTML string `json:"html"`
}

// NewSummarizer returns a summarizer using cfg.OpenAIAPIKey and cfg.OpenAIModel.
func NewSummarizer(cfg *config.DigestConfig, hc *http.Client) *Summarizer {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &Summarizer{apiKey: cfg.OpenAIAPIKey, model: cfg.OpenAIModel, hc: hc}
}

// SummarizeFamilyWeeks produces a plain-text family digest from the given messages.
func (s *Summarizer) SummarizeFamilyWeeks(ctx context.Context, msgs []gmail.InboxMessage, cfg *config.DigestConfig, logger *logging.Logger) (string, error) {
	out, err := s.SummarizeFamilyWeeksHTML(ctx, msgs, nil, cfg, logger)
	if err != nil {
		return "", err
	}
	return out.Text, nil
}

// SummarizeFamilyWeeksHTML produces both plain text and HTML for the family digest.
// The HTML is intended to contain headings plus bullet points (<h2>/<h3>, <ul>/<li>).
// events may be nil or empty when no calendar is configured.
func (s *Summarizer) SummarizeFamilyWeeksHTML(ctx context.Context, msgs []gmail.InboxMessage, events []calendar.Event, cfg *config.DigestConfig, logger *logging.Logger) (DigestOutput, error) {
	weeks := cfg.DigestWeeks
	if weeks < 1 {
		weeks = 1
	}
	today := time.Now().Format("2006-01-02")
	userPrompt, cited, omittedMsgs, omittedEvents := BuildDigestPrompt(msgs, events, cfg.DigestMaxBodyChars, cfg.PromptBudgetRunes)
	if omittedMsgs > 0 && logger != nil {
		logger.Infof("%d message(s) omitted from the model prompt due to size budget", omittedMsgs)
	}
	if omittedEvents > 0 && logger != nil {
		logger.Infof("%d calendar event(s) omitted from the model prompt due to size budget", omittedEvents)
	}
	lang := cfg.DigestLanguage
	if lang == "" {
		lang = config.DefaultDigestLanguage
	}
	system := buildFamilyDigestSystemPrompt(weeks, lang, today, cfg.Children)
	reqBody := map[string]any{
		"model":       s.model,
		"temperature": 0.35,
		"max_tokens":  4096,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": userPrompt},
		},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return DigestOutput{}, fmt.Errorf("marshal openai request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIChatURL, bytes.NewReader(payload))
	if err != nil {
		return DigestOutput{}, fmt.Errorf("openai request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	resp, err := s.hc.Do(req)
	if err != nil {
		return DigestOutput{}, fmt.Errorf("openai http: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return DigestOutput{}, fmt.Errorf("openai read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return DigestOutput{}, fmt.Errorf("openai status %s: %s", resp.Status, truncateForErr(string(body), 500))
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return DigestOutput{}, fmt.Errorf("openai decode: %w", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return DigestOutput{}, fmt.Errorf("openai api error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 || parsed.Choices[0].Message.Content == "" {
		return DigestOutput{}, fmt.Errorf("openai: empty choices or content")
	}
	out, err := parseDigestOutput(parsed.Choices[0].Message.Content)
	if err != nil {
		out = DigestOutput{
			Text: strings.TrimSpace(parsed.Choices[0].Message.Content),
			HTML: gmail.TextToSimpleHTML(parsed.Choices[0].Message.Content),
		}
	} else {
		if strings.TrimSpace(out.Text) == "" {
			out.Text = stripHTMLToText(out.HTML)
		}
		if strings.TrimSpace(out.HTML) == "" {
			out.HTML = gmail.TextToSimpleHTML(out.Text)
		}
	}
	return EnsureDigestSources(out, cited), nil
}

func parseDigestOutput(content string) (DigestOutput, error) {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		return DigestOutput{}, fmt.Errorf("unexpected markdown fence")
	}
	var out DigestOutput
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return DigestOutput{}, err
	}
	out.Text = strings.TrimSpace(out.Text)
	out.HTML = strings.TrimSpace(out.HTML)
	return out, nil
}

func stripHTMLToText(html string) string {
	s := strings.ReplaceAll(html, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
	return strings.TrimSpace(s)
}

func truncateForErr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
