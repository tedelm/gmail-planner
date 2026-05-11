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
	header := "Below are email sources numbered Source 1, Source 2, ... Use these numbers as inline citations (1), (2), ... in the digest and in the final reference list.\n\n"
	var b strings.Builder
	b.WriteString(header)
	used := utf8.RuneCountInString(b.String())
	omitted := 0
	for i, m := range msgs {
		body := TruncateRunes(m.Body, maxBodyRunes)
		dateStr := m.DateLocal
		if strings.TrimSpace(dateStr) == "" {
			dateStr = "(saknas)"
		}
		block := fmt.Sprintf("--- Source %d ---\nGmailMessageId: %s\nSubject: %s\nDate: %s\nThread: %s\n\n%s\n\n",
			i+1, m.ID, m.Headline, dateStr, m.ThreadID, body)
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
