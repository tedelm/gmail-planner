package config

import "testing"

func TestBuildDigestBaseQuery_nilConfig(t *testing.T) {
	got := BuildDigestBaseQuery(nil, "  newer_than:1d  ")
	if got != "newer_than:1d" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildDigestBaseQuery_empty(t *testing.T) {
	cfg := &DigestConfig{}
	if got := BuildDigestBaseQuery(cfg, ""); got != "" {
		t.Fatalf("got %q want empty", got)
	}
}

func TestBuildDigestBaseQuery_fromEnv(t *testing.T) {
	cfg := &DigestConfig{GmailDigestQuery: "newer_than:14d", GmailDigestLabels: []string{"Work"}}
	got := BuildDigestBaseQuery(cfg, "")
	if got != "newer_than:14d" {
		t.Fatalf("got %q (labels must not appear in base)", got)
	}
}

func TestBuildDigestBaseQuery_overrideIgnoresEnvQuery(t *testing.T) {
	cfg := &DigestConfig{
		GmailDigestQuery:  "from:old@example.com",
		GmailDigestLabels: []string{"Urgent"},
	}
	got := BuildDigestBaseQuery(cfg, "newer_than:7d")
	if got != "newer_than:7d" {
		t.Fatalf("got %q", got)
	}
}

func TestBuildDigestGmailQuery_aliasMatchesBase(t *testing.T) {
	cfg := &DigestConfig{GmailDigestQuery: "x", GmailDigestLabels: []string{"L"}}
	if BuildDigestGmailQuery(cfg, "") != BuildDigestBaseQuery(cfg, "") {
		t.Fatal("alias mismatch")
	}
}

func TestJoinGmailQueryParts(t *testing.T) {
	if got := JoinGmailQueryParts("  a  ", "", "b"); got != "a b" {
		t.Fatalf("got %q", got)
	}
	if JoinGmailQueryParts() != "" {
		t.Fatalf("empty join")
	}
}

func TestJoinGmailQueryParts_baseAndLabel(t *testing.T) {
	got := JoinGmailQueryParts("newer_than:14d", DigestLabelSearchTerm("Work"))
	if got != "newer_than:14d label:Work" {
		t.Fatalf("got %q", got)
	}
}

func TestDigestLabelSearchTerm_simple(t *testing.T) {
	if DigestLabelSearchTerm("Work") != "label:Work" {
		t.Fatal()
	}
}

func TestDigestLabelSearchTerm_spaceQuoted(t *testing.T) {
	if DigestLabelSearchTerm("School Events") != `label:"School Events"` {
		t.Fatalf("got %q", DigestLabelSearchTerm("School Events"))
	}
}

func TestDigestLabelSearchTerm_escapesQuoteAndBackslash(t *testing.T) {
	if DigestLabelSearchTerm(`a"b\c`) != `label:"a\"b\\c"` {
		t.Fatalf("got %q", DigestLabelSearchTerm(`a"b\c`))
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
