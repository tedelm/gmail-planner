package digest

import (
	"strings"
	"testing"

	"github.com/tedelm/gmail-planner/internal/gmail"
)

func TestTruncateRunes(t *testing.T) {
	got := TruncateRunes("abcdef", 4)
	if !strings.Contains(got, "…[truncated]") || strings.Contains(got, "ef") {
		t.Fatalf("got %q", got)
	}
	if TruncateRunes("hi", 10) != "hi" {
		t.Fatal("short string changed")
	}
}

func TestBuildMailDigestPrompt_BudgetOmitsTail(t *testing.T) {
	msgs := make([]gmail.InboxMessage, 5)
	for i := range msgs {
		msgs[i] = gmail.InboxMessage{
			ID:       "id",
			ThreadID: "th",
			Headline: "Subj",
			Body:     strings.Repeat("x", 200),
		}
	}
	_, omitted := BuildMailDigestPrompt(msgs, 50, 400)
	if omitted == 0 {
		t.Fatal("expected some messages omitted")
	}
}

func TestBuildMailDigestPrompt_IncludesDateWhenPresent(t *testing.T) {
	msgs := []gmail.InboxMessage{{
		ID:        "id1",
		ThreadID:  "th1",
		Headline:  "Hej",
		Body:      "Body",
		DateLocal: "2026-05-11 14:00",
	}}
	prompt, _ := BuildMailDigestPrompt(msgs, 100, 5000)
	if !strings.Contains(prompt, "Date: 2026-05-11 14:00") {
		t.Fatalf("expected date in prompt: %s", prompt)
	}
	if !strings.Contains(prompt, "--- Source 1 ---") {
		t.Fatalf("expected source header: %s", prompt)
	}
}

func TestBuildMailDigestPrompt_IncludesDigestLabelWhenSet(t *testing.T) {
	msgs := []gmail.InboxMessage{{
		ID:          "id1",
		ThreadID:    "th1",
		Headline:    "Hej",
		Body:        "Body",
		DateLocal:   "2026-05-11 14:00",
		DigestLabel: "Work",
	}}
	prompt, _ := BuildMailDigestPrompt(msgs, 100, 5000)
	if !strings.Contains(prompt, "Label: Work") {
		t.Fatalf("expected label line: %s", prompt)
	}
}
