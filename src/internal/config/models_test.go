package config

import "testing"

func TestLoadGmailOAuthFromEnv_MissingClientID(t *testing.T) {
	t.Setenv("GMAIL_CLIENT_ID", "")
	t.Setenv("GMAIL_CLIENT_SECRET", "s")
	_, err := LoadGmailOAuthFromEnv()
	if err == nil {
		t.Fatal("expected error for missing GMAIL_CLIENT_ID")
	}
}

func TestLoadGmailOAuthFromEnv_MissingClientSecret(t *testing.T) {
	t.Setenv("GMAIL_CLIENT_ID", "id")
	t.Setenv("GMAIL_CLIENT_SECRET", "")
	_, err := LoadGmailOAuthFromEnv()
	if err == nil {
		t.Fatal("expected error for missing GMAIL_CLIENT_SECRET")
	}
}

func TestLoadGmailOAuthFromEnv_UsesDefaults(t *testing.T) {
	t.Setenv("GMAIL_CLIENT_ID", "id")
	t.Setenv("GMAIL_CLIENT_SECRET", "secret")
	t.Setenv("GMAIL_TOKEN_PATH", "")
	t.Setenv("GMAIL_OAUTH_REDIRECT_URL", "")
	cfg, err := LoadGmailOAuthFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.TokenPath != DefaultGmailTokenPath {
		t.Fatalf("TokenPath: got %q want %q", cfg.TokenPath, DefaultGmailTokenPath)
	}
	if cfg.RedirectURL != DefaultGmailOAuthRedirectURL {
		t.Fatalf("RedirectURL: got %q want %q", cfg.RedirectURL, DefaultGmailOAuthRedirectURL)
	}
}

func TestLoadGmailOAuthFromEnv_CustomPaths(t *testing.T) {
	t.Setenv("GMAIL_CLIENT_ID", "id")
	t.Setenv("GMAIL_CLIENT_SECRET", "secret")
	t.Setenv("GMAIL_TOKEN_PATH", "C:\\tmp\\my.token.json")
	t.Setenv("GMAIL_OAUTH_REDIRECT_URL", "http://localhost:9999/")
	cfg, err := LoadGmailOAuthFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.TokenPath != `C:\tmp\my.token.json` {
		t.Fatalf("TokenPath: got %q", cfg.TokenPath)
	}
	if cfg.RedirectURL != "http://localhost:9999/" {
		t.Fatalf("RedirectURL: got %q", cfg.RedirectURL)
	}
}

func TestLoadDigestConfigFromEnv_DefaultLanguage(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "k")
	t.Setenv("DIGEST_TO_EMAIL", "to@example.com")
	t.Setenv("DIGEST_LANGUAGE", "")
	cfg, err := LoadDigestConfigFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.DigestToEmails) != 1 || cfg.DigestToEmails[0] != "to@example.com" {
		t.Fatalf("DigestToEmails: got %#v", cfg.DigestToEmails)
	}
	if cfg.DigestLanguage != DefaultDigestLanguage {
		t.Fatalf("DigestLanguage: got %q want %q", cfg.DigestLanguage, DefaultDigestLanguage)
	}
}

func TestLoadDigestConfigFromEnv_ToEmailsCSV(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "k")
	t.Setenv("DIGEST_TO_EMAIL", "a@example.com, b@example.com,,c@example.com ")
	cfg, err := LoadDigestConfigFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.DigestToEmails) != 3 {
		t.Fatalf("DigestToEmails: got %#v", cfg.DigestToEmails)
	}
	if cfg.DigestToEmails[1] != "b@example.com" {
		t.Fatalf("DigestToEmails[1]: got %q", cfg.DigestToEmails[1])
	}
}
