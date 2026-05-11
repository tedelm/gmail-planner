package config

// Gmail OAuth and inbox listing defaults (env keys documented in .env-example).
const (
	DefaultGmailTokenPath        = "gmail-token.json"
	DefaultGmailOAuthRedirectURL = "http://127.0.0.1:8765/"

	DefaultGmailInboxListLimit int64 = 10
	GmailInboxListMin          int64 = 1
	GmailInboxListMax          int64 = 500

	DefaultOpenAIModel        = "gpt-4o-mini"
	DefaultDigestWeeks        = 1
	DefaultDigestMaxBodyChars = 4000
	DefaultDigestPromptBudget = 80000 // runes cap for assembled user prompt (excluding system text)

	DefaultDigestLanguage = "sv-SE"
)

// GmailOAuthConfig holds Gmail OAuth client settings and token storage path.
type GmailOAuthConfig struct {
	ClientID     string
	ClientSecret string
	TokenPath    string
	RedirectURL  string
}

// DigestConfig holds OpenAI + family digest email settings (see .env-example).
type DigestConfig struct {
	OpenAIAPIKey       string
	OpenAIModel        string
	DigestToEmail      string
	DigestSubject      string
	DigestWeeks        int
	GmailDigestQuery   string
	DigestMaxBodyChars int
	PromptBudgetRunes  int
	DigestLanguage     string
}
