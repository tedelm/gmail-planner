package digest

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/tedelm/gmail-planner/internal/calendar"
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

// BuildMailDigestPrompt builds the user message text from inbox summaries only.
// Prefer BuildDigestPrompt when calendar events may also be included.
// It returns how many messages were omitted after the budget was reached.
func BuildMailDigestPrompt(msgs []gmail.InboxMessage, maxBodyRunes int, budgetRunes int) (string, int) {
	prompt, _, omittedMsgs, _ := BuildDigestPrompt(msgs, nil, maxBodyRunes, budgetRunes)
	return prompt, omittedMsgs
}

// BuildDigestPrompt builds the user message from emails then calendar events as
// numbered Source blocks. It returns the prompt, citations for included sources,
// and how many messages/events were omitted after the budget was reached.
func BuildDigestPrompt(msgs []gmail.InboxMessage, events []calendar.Event, maxBodyRunes int, budgetRunes int) (string, []CitedSource, int, int) {
	if budgetRunes < 512 {
		budgetRunes = 512
	}
	header := "Below are sources numbered Source 1, Source 2, ... " +
		"(emails first, then Google Calendar events). " +
		"Use these numbers as inline citations (1), (2), ... in the digest and in the final reference list.\n\n"
	var b strings.Builder
	b.WriteString(header)
	used := utf8.RuneCountInString(b.String())
	var cited []CitedSource
	sourceNum := 0
	omittedMsgs := 0
	omittedEvents := 0

	for i, m := range msgs {
		body := TruncateRunes(m.Body, maxBodyRunes)
		dateStr := m.DateLocal
		if strings.TrimSpace(dateStr) == "" {
			dateStr = "(saknas)"
		}
		sourceNum++
		block := fmt.Sprintf("--- Source %d ---\nType: email\nGmailMessageId: %s\nSubject: %s\nDate: %s\nThread: %s\n",
			sourceNum, m.ID, m.Headline, dateStr, m.ThreadID)
		if from := strings.TrimSpace(m.From); from != "" {
			block += fmt.Sprintf("From: %s\n", from)
		}
		if lab := strings.TrimSpace(m.DigestLabel); lab != "" {
			block += fmt.Sprintf("Label: %s\n", lab)
		}
		block += fmt.Sprintf("\n%s\n\n", body)
		add := utf8.RuneCountInString(block)
		if used+add > budgetRunes {
			omittedMsgs = len(msgs) - i
			omittedEvents = len(events)
			sourceNum--
			break
		}
		b.WriteString(block)
		used += add
		cited = append(cited, CitedSource{
			Subject: strings.TrimSpace(m.Headline),
			Date:    dateStr,
			Label:   strings.TrimSpace(m.DigestLabel),
		})
	}

	if omittedMsgs == 0 {
		for i, ev := range events {
			sourceNum++
			block := formatCalendarSourceBlock(sourceNum, ev)
			add := utf8.RuneCountInString(block)
			if used+add > budgetRunes {
				omittedEvents = len(events) - i
				sourceNum--
				break
			}
			b.WriteString(block)
			used += add
			dateStr := strings.TrimSpace(ev.StartLocal)
			if dateStr == "" {
				dateStr = "(saknas)"
			}
			cited = append(cited, CitedSource{
				Subject: strings.TrimSpace(ev.Summary),
				Date:    dateStr,
				Label:   "Kalender",
			})
		}
	}

	return b.String(), cited, omittedMsgs, omittedEvents
}

func formatCalendarSourceBlock(n int, ev calendar.Event) string {
	dateStr := strings.TrimSpace(ev.StartLocal)
	if dateStr == "" {
		dateStr = "(saknas)"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "--- Source %d ---\nType: calendar\nCalendarEventId: %s\nSubject: %s\nDate: %s\n",
		n, ev.ID, ev.Summary, dateStr)
	if end := strings.TrimSpace(ev.EndLocal); end != "" {
		fmt.Fprintf(&b, "End: %s\n", end)
	}
	if ev.AllDay {
		b.WriteString("AllDay: true\n")
	}
	if loc := strings.TrimSpace(ev.Location); loc != "" {
		fmt.Fprintf(&b, "Location: %s\n", loc)
	}
	b.WriteString("\n")
	return b.String()
}
