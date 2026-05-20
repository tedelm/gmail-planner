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
	out, err := s.SummarizeFamilyWeeksHTML(ctx, msgs, cfg, logger)
	if err != nil {
		return "", err
	}
	return out.Text, nil
}

// SummarizeFamilyWeeksHTML produces both plain text and HTML for the family digest.
// The HTML is intended to contain headings plus bullet points (<h2>/<h3>, <ul>/<li>).
func (s *Summarizer) SummarizeFamilyWeeksHTML(ctx context.Context, msgs []gmail.InboxMessage, cfg *config.DigestConfig, logger *logging.Logger) (DigestOutput, error) {
	weeks := cfg.DigestWeeks
	if weeks < 1 {
		weeks = 1
	}
	today := time.Now().Format("2006-01-02")
	userPrompt, omitted := BuildMailDigestPrompt(msgs, cfg.DigestMaxBodyChars, cfg.PromptBudgetRunes)
	cited := messagesCitedInPrompt(msgs, omitted)
	if omitted > 0 && logger != nil {
		logger.Infof("%d message(s) omitted from the model prompt due to size budget", omitted)
	}
	lang := cfg.DigestLanguage
	if lang == "" {
		lang = config.DefaultDigestLanguage
	}
	system := fmt.Sprintf(
		"You help a family plan ahead. Summarize the following emails into a clear, actionable digest "+
			"covering roughly the next %d week(s). Use short sections and bullet points. "+
			"Call out dates, times, locations, deadlines, and who should do what when the emails imply it. "+
			"Do not invent events or commitments not supported by the email text. "+
			"IMPORTANT: Write the entire digest in %s. If the email content is in another language, translate it as needed. "+
			"Keep names, email addresses, phone numbers, URLs, and exact dates/times intact. "+
			"Citations: Each input block is labeled \"--- Source n ---\". For every bullet that draws on a source, "+
			"end the <li> text with an inline citation like (n) matching that Source number. "+
			"If a bullet combines multiple sources, use multiple citations like (1)(3). "+
			"When a source block includes a \"Label:\" line (Gmail label search), mention that label in the digest "+
			"(inline or short subheadings) so readers can see which label bucket each item came from. "+
			"At the END of the html (after the main digest), add a section <h2>Källor</h2> followed by <ol> where each <li> is "+
			"exactly: (n) Subject — Date using ONLY the Subject and Date lines from the corresponding Source block "+
			"(if Date was \"(saknas)\", omit the em dash and date part and use only the subject). "+
			"If the source had a Label line, append \" — \" and that label text to the Källor line. "+
			"Do not invent source numbers; only use n values that appear in the input. "+
			"Return ONLY valid JSON (no markdown, no code fences) with this shape: "+
			"{\"text\":\"...plain text...\",\"html\":\"...HTML...\"}. "+
			"The html value MUST be a complete HTML fragment (no markdown) and should use headings (h2/h3) "+
			"and bullet lists (ul/li) for readability. The text field should mirror the same citations and end with "+
			"a \"Källor\" section listing (n) Subject — Date lines, each with an appended \" — Label\" when that source had a Label line."+
			"Do not include events that have already happened, use the information in the email (or email sent/received date if no date is available in the email body) to determine if the event has already happened. Today's date is %s."+
			"Important: Always check the email header for week numbers and use them to determine if the event has already happened. If the event has already happened, do not include it in the digest.",
		weeks,
		lang,
		today,
	)
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
		// Fallback: treat content as plain text and generate simple HTML
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
	// Sometimes models wrap JSON in whitespace/newlines; don't accept markdown fences.
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
	// Minimal fallback: keep it simple; HTML is already meant for email, so we don't
	// attempt full parsing here.
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
