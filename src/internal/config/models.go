package config

// Gmail OAuth and inbox listing defaults (env keys documented in .env-example).
const (
	DefaultGmailTokenPath        = "gmail-token.json"
	DefaultGmailOAuthRedirectURL = "http://127.0.0.1:8765/"

	DefaultGmailInboxListLimit int64 = 10
	GmailInboxListMin          int64 = 1
	GmailInboxListMax          int64 = 500

	DefaultOpenAIModel        = "gpt-6.1-sol"
	DefaultDigestWeeks        = 1
	DefaultDigestMaxBodyChars = 4000
	DefaultDigestPromptBudget = 80000 // runes cap for assembled user prompt (excluding system text)

	DefaultDigestLanguage = "sv-SE"

	// DefaultSkolmatenClientToken is the public SPA client token used by skolmaten.se itself.
	// Override with SKOLMATEN_CLIENT_TOKEN when registering a dedicated app token.
	DefaultSkolmatenClientToken = "web-eaa12e50-c84c-4b4a-9cfe-4e3fcbcd9165"
)

// GmailOAuthConfig holds Gmail OAuth client settings and token storage path.
type GmailOAuthConfig struct {
	ClientID      string
	ClientSecret  string
	TokenPath     string
	RedirectURL   string
	EphemeralAuth bool // when true: always browser login; never read/write TokenPath
}

// DigestConfig holds OpenAI + family digest email settings (see .env-example).
type DigestConfig struct {
	OpenAIAPIKey       string
	OpenAIModel        string
	DigestToEmails     []string
	DigestSubject      string
	DigestWeeks        int
	GmailDigestQuery   string
	GmailDigestLabels  []string
	GmailCalendarID       string // Google Calendar ID (e.g. shared calendar owner email); empty skips calendar fetch
	SkolmatenClientToken  string // Client-Token for skolmaten.se API; empty uses DefaultSkolmatenClientToken
	Children              []DigestChild
	DigestMaxBodyChars    int
	PromptBudgetRunes     int
	DigestLanguage        string
}

// DigestChild maps a child display name to Gmail label hints, class codes, and optional skolmaten slug.
type DigestChild struct {
	Name          string
	Labels        []string
	ClassCodes    []string
	SkolmatenSlug string // skolmaten.se URL slug, e.g. "osterskolan"; empty skips menu fetch for this child
}

// MaxDigestChildren is the maximum number of DIGEST_CHILD_N_* profiles loaded from env.
const MaxDigestChildren = 8
