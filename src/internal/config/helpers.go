package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// LoadDotEnv loads key=value pairs from a .env file into the process environment.
// Lookup order when DOTENV_PATH is unset: ".env", "src/.env", then "../.env".
// If DOTENV_PATH is set, only that file is loaded and it must exist.
// If DOTENV_PATH is unset and no candidate file exists, LoadDotEnv returns nil.
func LoadDotEnv() error {
	if p := strings.TrimSpace(os.Getenv("DOTENV_PATH")); p != "" {
		if err := godotenv.Load(p); err != nil {
			return fmt.Errorf("load DOTENV_PATH %s: %w", p, err)
		}
		return nil
	}
	candidates := []string{
		".env",
		filepath.Join("src", ".env"),
		filepath.Join("..", ".env"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if err := godotenv.Load(p); err != nil {
			return fmt.Errorf("load %s: %w", p, err)
		}
		return nil
	}
	return nil
}

// LoadGmailOAuthFromEnv reads GMAIL_CLIENT_ID, GMAIL_CLIENT_SECRET, and optional
// GMAIL_TOKEN_PATH, GMAIL_OAUTH_REDIRECT_URL from the environment.
func LoadGmailOAuthFromEnv() (*GmailOAuthConfig, error) {
	id := getenvTrim("GMAIL_CLIENT_ID")
	secret := getenvTrim("GMAIL_CLIENT_SECRET")
	if id == "" {
		return nil, fmt.Errorf("GMAIL_CLIENT_ID is required")
	}
	if secret == "" {
		return nil, fmt.Errorf("GMAIL_CLIENT_SECRET is required")
	}
	tokenPath := getenvTrim("GMAIL_TOKEN_PATH")
	if tokenPath == "" {
		tokenPath = DefaultGmailTokenPath
	}
	redirect := getenvTrim("GMAIL_OAUTH_REDIRECT_URL")
	if redirect == "" {
		redirect = DefaultGmailOAuthRedirectURL
	}
	return &GmailOAuthConfig{
		ClientID:     id,
		ClientSecret: secret,
		TokenPath:    tokenPath,
		RedirectURL:  redirect,
	}, nil
}

// LoadDigestConfigFromEnv loads digest/OpenAI settings. OPENAI_API_KEY and
// DIGEST_TO_EMAIL must be non-empty.
func LoadDigestConfigFromEnv() (*DigestConfig, error) {
	key := getenvTrim("OPENAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required for digest mode")
	}
	toRaw := getenvTrim("DIGEST_TO_EMAIL")
	toList := splitCSV(toRaw)
	if len(toList) == 0 {
		return nil, fmt.Errorf("DIGEST_TO_EMAIL is required for digest mode")
	}
	model := getenvTrim("OPENAI_MODEL")
	if model == "" {
		model = DefaultOpenAIModel
	}
	weeks := getenvInt("DIGEST_WEEKS", DefaultDigestWeeks)
	if weeks < 1 {
		weeks = DefaultDigestWeeks
	}
	maxBody := getenvInt("DIGEST_MAX_BODY_CHARS", DefaultDigestMaxBodyChars)
	if maxBody < 256 {
		maxBody = 256
	}
	lang := getenvTrim("DIGEST_LANGUAGE")
	if lang == "" {
		lang = DefaultDigestLanguage
	}
	return &DigestConfig{
		OpenAIAPIKey:       key,
		OpenAIModel:        model,
		DigestToEmails:     toList,
		DigestSubject:      getenvTrim("DIGEST_SUBJECT"),
		DigestWeeks:        weeks,
		GmailDigestQuery:   getenvTrim("GMAIL_DIGEST_QUERY"),
		DigestMaxBodyChars: maxBody,
		PromptBudgetRunes:  DefaultDigestPromptBudget,
		DigestLanguage:     lang,
	}, nil
}

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func getenvTrim(key string) string {
	return strings.TrimSpace(strings.TrimSuffix(os.Getenv(key), "\r"))
}

func getenvFloat64(key string, defaultValue float64) float64 {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return defaultValue
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return defaultValue
	}
	return f
}

func getenvInt(key string, defaultValue int) int {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return i
}

func getenvCSV(key string) []string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return nil
	}
	parts := strings.Split(val, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
