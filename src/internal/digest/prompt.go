package digest

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/tedelm/gmail-planner/internal/gmail"
)

// TruncateRunes truncates s to at most n runes, appending a marker if truncated.
func TruncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n]) + "\n…[truncated]"
}

// BuildMailDigestPrompt builds the user message text from inbox summaries.
// It returns how many messages were omitted after the budget was reached.
func BuildMailDigestPrompt(msgs []gmail.InboxMessage, maxBodyRunes int, budgetRunes int) (string, int) {
	if budgetRunes < 512 {
		budgetRunes = 512
	}
	header := "Below are email messages (subject + body). Summarize them for the family digest.\n\n"
	var b strings.Builder
	b.WriteString(header)
	used := utf8.RuneCountInString(b.String())
	omitted := 0
	for i, m := range msgs {
		body := TruncateRunes(m.Body, maxBodyRunes)
		block := fmt.Sprintf("--- Message %d ---\nID: %s\nThread: %s\nSubject: %s\n\n%s\n\n",
			i+1, m.ID, m.ThreadID, m.Headline, body)
		add := utf8.RuneCountInString(block)
		if used+add > budgetRunes {
			omitted = len(msgs) - i
			break
		}
		b.WriteString(block)
		used += add
	}
	return b.String(), omitted
}
