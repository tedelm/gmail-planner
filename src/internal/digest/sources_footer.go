package digest

import (
	"html"
	"strconv"
	"strings"

	"github.com/tedelm/gmail-planner/internal/gmail"
)

// messagesCitedInPrompt returns the inbox slice in the same order and length as
// "Source n" blocks in BuildMailDigestPrompt (omitted tail excluded).
func messagesCitedInPrompt(msgs []gmail.InboxMessage, omitted int) []gmail.InboxMessage {
	if omitted < 0 {
		omitted = 0
	}
	if omitted > len(msgs) {
		omitted = len(msgs)
	}
	n := len(msgs) - omitted
	if n <= 0 {
		return nil
	}
	return msgs[:n]
}

// EnsureDigestSources strips any trailing Källor section from model output and
// appends a canonical HTML and plain-text reference list for cited messages.
func EnsureDigestSources(out DigestOutput, cited []gmail.InboxMessage) DigestOutput {
	if len(cited) == 0 {
		return out
	}
	out.HTML = strings.TrimSpace(stripHTMLSourcesFooter(out.HTML)) + "\n" + buildHTMLSourcesFooter(cited)
	out.Text = strings.TrimSpace(stripTextSourcesFooter(out.Text)) + "\n\nKällor\n" + buildTextSourcesFooter(cited)
	return out
}

func stripHTMLSourcesFooter(s string) string {
	s = strings.TrimRight(s, " \t\r\n")
	if s == "" {
		return s
	}
	lower := strings.ToLower(s)
	needle := "<h2>källor</h2>"
	idx := strings.LastIndex(lower, needle)
	if idx < 0 {
		return s
	}
	return strings.TrimSpace(s[:idx])
}

func stripTextSourcesFooter(s string) string {
	s = strings.TrimRight(s, " \t\r\n")
	if s == "" {
		return s
	}
	lower := strings.ToLower(s)
	for _, needle := range []string{"\n\nKällor\n", "\nKällor\n"} {
		idx := strings.LastIndex(lower, strings.ToLower(needle))
		if idx >= 0 {
			return strings.TrimSpace(s[:idx])
		}
	}
	return s
}

func sourceListLine(n int, m gmail.InboxMessage) string {
	subject := strings.TrimSpace(m.Headline)
	date := strings.TrimSpace(m.DateLocal)
	label := strings.TrimSpace(m.DigestLabel)
	var b strings.Builder
	b.WriteString("(")
	b.WriteString(strconv.Itoa(n))
	b.WriteString(") ")
	b.WriteString(subject)
	if date != "" && !strings.EqualFold(date, "(saknas)") {
		b.WriteString(" — ")
		b.WriteString(date)
	}
	if label != "" {
		b.WriteString(" — ")
		b.WriteString(label)
	}
	return b.String()
}

func buildHTMLSourcesFooter(cited []gmail.InboxMessage) string {
	var b strings.Builder
	b.WriteString("<h2>Källor</h2>\n<ol>")
	for i := range cited {
		line := sourceListLine(i+1, cited[i])
		b.WriteString("<li>")
		b.WriteString(html.EscapeString(line))
		b.WriteString("</li>")
	}
	b.WriteString("</ol>")
	return b.String()
}

func buildTextSourcesFooter(cited []gmail.InboxMessage) string {
	var b strings.Builder
	for i := range cited {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(sourceListLine(i+1, cited[i]))
	}
	return b.String()
}
