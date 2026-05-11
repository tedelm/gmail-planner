package config

import "testing"

func TestBuildDigestGmailQuery_nilConfig(t *testing.T) {
	got := BuildDigestGmailQuery(nil, "  newer_than:1d  ")
	if got != "newer_than:1d" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildDigestGmailQuery_empty(t *testing.T) {
	cfg := &DigestConfig{}
	if got := BuildDigestGmailQuery(cfg, ""); got != "" {
		t.Fatalf("got %q want empty", got)
	}
}

func TestBuildDigestGmailQuery_labelsOnly(t *testing.T) {
	cfg := &DigestConfig{GmailDigestLabels: []string{"Work", "Family"}}
	got := BuildDigestGmailQuery(cfg, "")
	if got != "label:Work label:Family" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildDigestGmailQuery_queryAndLabels(t *testing.T) {
	cfg := &DigestConfig{
		GmailDigestQuery:  "newer_than:14d",
		GmailDigestLabels: []string{"Work"},
	}
	got := BuildDigestGmailQuery(cfg, "")
	if got != "newer_than:14d label:Work" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildDigestGmailQuery_overrideDisplacesEnvQueryButKeepsLabels(t *testing.T) {
	cfg := &DigestConfig{
		GmailDigestQuery:  "from:old@example.com",
		GmailDigestLabels: []string{"Urgent"},
	}
	got := BuildDigestGmailQuery(cfg, "newer_than:7d")
	if got != "newer_than:7d label:Urgent" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildDigestGmailQuery_labelWithSpaceQuoted(t *testing.T) {
	cfg := &DigestConfig{GmailDigestLabels: []string{"School Events"}}
	got := BuildDigestGmailQuery(cfg, "")
	if got != `label:"School Events"` {
		t.Fatalf("got %q", got)
	}
}

func TestBuildDigestGmailQuery_labelEscapesQuoteAndBackslash(t *testing.T) {
	cfg := &DigestConfig{GmailDigestLabels: []string{`a"b\c`}}
	got := BuildDigestGmailQuery(cfg, "")
	if got != `label:"a\"b\\c"` {
		t.Fatalf("got %q", got)
	}
}

func TestLoadDigestConfigFromEnv_GmailDigestLabels(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "k")
	t.Setenv("DIGEST_TO_EMAIL", "to@example.com")
	t.Setenv("GMAIL_DIGEST_LABELS", " Work , , Family ")
	cfg, err := LoadDigestConfigFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.GmailDigestLabels) != 2 || cfg.GmailDigestLabels[0] != "Work" || cfg.GmailDigestLabels[1] != "Family" {
		t.Fatalf("GmailDigestLabels: %#v", cfg.GmailDigestLabels)
	}
}
