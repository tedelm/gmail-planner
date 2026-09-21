package digest

import (
	"html"
	"regexp"
	"strings"

	"github.com/tedelm/gmail-planner/internal/gmail"
)

var (
	reHTMLCloseBreak = regexp.MustCompile(`(?i)</(h[1-6]|p|li|div|tr|table|ul|ol|br)\s*>`)
	reHTMLBR         = regexp.MustCompile(`(?i)<br\s*/?>`)
	reHTMLTag        = regexp.MustCompile(`(?i)<[^>]+>`)
	reMultiNewline   = regexp.MustCompile(`\n{3,}`)
)

// normalizeDigestOutput fixes common model mistakes so the HTML part renders and
// the plain-text part never contains raw HTML tags.
func normalizeDigestOutput(out DigestOutput) DigestOutput {
	out.Text = strings.TrimSpace(out.Text)
	out.HTML = strings.TrimSpace(out.HTML)

	if looksEntityEscapedHTML(out.HTML) {
		out.HTML = html.UnescapeString(out.HTML)
	}
	if looksEntityEscapedHTML(out.Text) && !looksLikeHTMLMarkup(out.HTML) {
		// Model sometimes put escaped HTML only in text.
		unescaped := html.UnescapeString(out.Text)
		if looksLikeHTMLMarkup(unescaped) {
			out.HTML = unescaped
		}
	}
	if !looksLikeHTMLMarkup(out.HTML) && looksLikeHTMLMarkup(out.Text) {
		out.HTML = out.Text
	}

	if !looksLikeHTMLMarkup(out.HTML) {
		plain := out.Text
		if plain == "" {
			plain = out.HTML
		}
		if plain == "" {
			return out
		}
		out.HTML = gmail.TextToSimpleHTML(plain)
		out.Text = plain
		return out
	}

	out.Text = htmlFragmentToPlainText(out.HTML)
	return out
}

func looksEntityEscapedHTML(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, "&lt;h2") ||
		strings.Contains(lower, "&lt;h3") ||
		strings.Contains(lower, "&lt;ul") ||
		strings.Contains(lower, "&lt;li") ||
		strings.Contains(lower, "&lt;ol")
}

func looksLikeHTMLMarkup(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, "<h2") ||
		strings.Contains(lower, "<h3") ||
		strings.Contains(lower, "<ul") ||
		strings.Contains(lower, "<li") ||
		strings.Contains(lower, "<ol") ||
		strings.Contains(lower, "<p>") ||
		strings.Contains(lower, "<br")
}

// htmlFragmentToPlainText converts an HTML fragment to readable plain text.
func htmlFragmentToPlainText(s string) string {
	s = reHTMLBR.ReplaceAllString(s, "\n")
	s = reHTMLCloseBreak.ReplaceAllString(s, "\n")
	s = reHTMLTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	s = strings.Join(lines, "\n")
	s = reMultiNewline.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

// stripMarkdownJSONFence removes a leading ``` / ```json fence and trailing ``` if present.
func stripMarkdownJSONFence(content string) string {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "```") {
		return content
	}
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSpace(content)
	if len(content) >= 4 && strings.EqualFold(content[:4], "json") {
		content = strings.TrimSpace(content[4:])
	}
	if idx := strings.LastIndex(content, "```"); idx >= 0 {
		content = content[:idx]
	}
	return strings.TrimSpace(content)
}
