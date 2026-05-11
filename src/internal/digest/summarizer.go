package digest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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

// NewSummarizer returns a summarizer using cfg.OpenAIAPIKey and cfg.OpenAIModel.
func NewSummarizer(cfg *config.DigestConfig, hc *http.Client) *Summarizer {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &Summarizer{apiKey: cfg.OpenAIAPIKey, model: cfg.OpenAIModel, hc: hc}
}

// SummarizeFamilyWeeks produces a plain-text family digest from the given messages.
func (s *Summarizer) SummarizeFamilyWeeks(ctx context.Context, msgs []gmail.InboxMessage, cfg *config.DigestConfig, logger *logging.Logger) (string, error) {
	weeks := cfg.DigestWeeks
	if weeks < 1 {
		weeks = 1
	}
	userPrompt, omitted := BuildMailDigestPrompt(msgs, cfg.DigestMaxBodyChars, cfg.PromptBudgetRunes)
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
			"Keep names, email addresses, phone numbers, URLs, and exact dates/times intact.",
		weeks,
		lang,
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
		return "", fmt.Errorf("marshal openai request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIChatURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("openai request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	resp, err := s.hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai http: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("openai read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("openai status %s: %s", resp.Status, truncateForErr(string(body), 500))
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
		return "", fmt.Errorf("openai decode: %w", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("openai api error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 || parsed.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("openai: empty choices or content")
	}
	return parsed.Choices[0].Message.Content, nil
}

func truncateForErr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
