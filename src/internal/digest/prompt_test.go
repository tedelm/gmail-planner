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
