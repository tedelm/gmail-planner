package digest

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"
)

// Charcoal / mint palette (docs/digest-mockup.html).
const (
	colorBgOuter  = "#0F1115"
	colorCard     = "#171A21"
	colorSurface  = "#1E222B"
	colorLine     = "#2A2F3A"
	colorText     = "#F3F4F6"
	colorMuted    = "#9CA3AF"
	colorAccent   = "#34D399"
	colorAccent2  = "#6EE7B7"
)

// Shared inline styles for model prompt + Go-built chrome/footer.
const (
	styleH2 = "margin:0 0 12px 0;font-size:18px;color:#F3F4F6;border-bottom:2px solid #34D399;padding-bottom:8px;"
	styleH3 = "margin:16px 0 10px 0;font-size:15px;color:#6EE7B7;"
	styleH2Sources = "margin:0 0 10px 0;font-size:15px;color:#9CA3AF;border-bottom:1px solid #2A2F3A;padding-bottom:8px;"
	stylePill = "display:inline-block;background:#2A2F3A;color:#6EE7B7;font-size:11px;font-weight:600;padding:3px 9px;border-radius:999px;"
	stylePillStrong = "display:inline-block;background:#2A2F3A;color:#34D399;font-size:12px;font-weight:700;padding:4px 10px;border-radius:999px;"
	stylePillMuted = "display:inline-block;background:#2A2F3A;color:#9CA3AF;font-size:11px;font-weight:600;padding:3px 9px;border-radius:999px;"
	stylePillUrgent = "display:inline-block;background:#1E222B;color:#34D399;font-size:11px;font-weight:600;padding:2px 8px;border-radius:999px;border:1px solid #34D399;"
	styleTable = "border-collapse:collapse;margin:0 0 28px 0;border:1px solid #2A2F3A;"
	styleTh = "padding:10px 12px;font-size:11px;text-transform:uppercase;letter-spacing:0.04em;color:#6EE7B7;border-bottom:1px solid #2A2F3A;"
	styleTd = "padding:12px;font-size:14px;border-bottom:1px solid #2A2F3A;vertical-align:top;color:#F3F4F6;"
	styleTdMuted = "padding:12px;font-size:13px;color:#9CA3AF;border-bottom:1px solid #2A2F3A;vertical-align:top;"
)

// DigestEmailMeta drives the themed email chrome around the model HTML fragment.
type DigestEmailMeta struct {
	Weeks       int
	RangeLabel  string
	SourceCount int
}

// FormatDigestRangeLabel formats [from, to] inclusive as e.g. "16–22 nov 2026" (Swedish month abbr).
func FormatDigestRangeLabel(from, to time.Time) string {
	months := [...]string{"jan", "feb", "mar", "apr", "maj", "jun", "jul", "aug", "sep", "okt", "nov", "dec"}
	// Caller should pass times already in the digest timezone (e.g. Europe/Stockholm).
	fm, tm := int(from.Month()), int(to.Month())
	fy, ty := from.Year(), to.Year()
	fd, td := from.Day(), to.Day()
	if fy == ty && fm == tm {
		return fmt.Sprintf("%d–%d %s %d", fd, td, months[fm-1], fy)
	}
	if fy == ty {
		return fmt.Sprintf("%d %s–%d %s %d", fd, months[fm-1], td, months[tm-1], fy)
	}
	return fmt.Sprintf("%d %s %d–%d %s %d", fd, months[fm-1], fy, td, months[tm-1], ty)
}

// DigestRangeForWeeks returns start/end of the digest window in loc (inclusive end).
func DigestRangeForWeeks(now time.Time, weeks int, loc *time.Location) (time.Time, time.Time) {
	if weeks < 1 {
		weeks = 1
	}
	if loc == nil {
		loc = time.Local
	}
	start := now.In(loc)
	y, m, d := start.Date()
	start = time.Date(y, m, d, 0, 0, 0, 0, loc)
	end := start.AddDate(0, 0, weeks*7-1)
	return start, end
}

// WrapDigestEmailHTML wraps an inner HTML fragment in the mockup email chrome.
func WrapDigestEmailHTML(inner string, meta DigestEmailMeta) string {
	inner = strings.TrimSpace(inner)
	weeks := meta.Weeks
	if weeks < 1 {
		weeks = 1
	}
	rangeLabel := strings.TrimSpace(meta.RangeLabel)
	if rangeLabel == "" {
		rangeLabel = "—"
	}
	daysLabel := strconv.Itoa(weeks*7) + " dagar"
	srcLabel := strconv.Itoa(meta.SourceCount) + " källor"

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"sv\"><head><meta charset=\"UTF-8\">")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">")
	b.WriteString("<title>Familjedigest</title></head>")
	b.WriteString("<body style=\"margin:0;padding:0;background:")
	b.WriteString(colorBgOuter)
	b.WriteString(";font-family:Segoe UI,Helvetica,Arial,sans-serif;color:")
	b.WriteString(colorText)
	b.WriteString(";line-height:1.45;\">")
	b.WriteString("<table role=\"presentation\" width=\"100%\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" style=\"background:")
	b.WriteString(colorBgOuter)
	b.WriteString(";\"><tr><td align=\"center\" style=\"padding:24px 12px;\">")
	b.WriteString("<table role=\"presentation\" width=\"640\" cellpadding=\"0\" cellspacing=\"0\" border=\"0\" style=\"max-width:640px;width:100%;background:")
	b.WriteString(colorCard)
	b.WriteString(";border-radius:12px;overflow:hidden;border:1px solid ")
	b.WriteString(colorLine)
	b.WriteString(";\">")

	// Header
	b.WriteString("<tr><td style=\"background:")
	b.WriteString(colorSurface)
	b.WriteString(";padding:28px 28px 22px 28px;border-bottom:3px solid ")
	b.WriteString(colorAccent)
	b.WriteString(";\">")
	b.WriteString("<p style=\"margin:0 0 6px 0;font-size:13px;letter-spacing:0.04em;text-transform:uppercase;color:")
	b.WriteString(colorAccent2)
	b.WriteString(";font-weight:600;\">Familjeöversikt</p>")
	b.WriteString("<h1 style=\"margin:0 0 14px 0;font-size:26px;line-height:1.2;color:")
	b.WriteString(colorText)
	b.WriteString(";font-weight:700;\">Veckan framåt</h1><p style=\"margin:0;\">")
	writeHeaderPill(&b, rangeLabel, colorAccent2)
	writeHeaderPill(&b, daysLabel, colorAccent2)
	writeHeaderPill(&b, srcLabel, colorAccent)
	b.WriteString("</p></td></tr>")

	// Body
	b.WriteString("<tr><td style=\"padding:28px 28px 8px 28px;background:")
	b.WriteString(colorCard)
	b.WriteString(";\">")
	b.WriteString(inner)
	b.WriteString("</td></tr>")

	// Footer
	b.WriteString("<tr><td style=\"padding:16px 28px 24px 28px;background:")
	b.WriteString(colorBgOuter)
	b.WriteString(";border-top:1px solid ")
	b.WriteString(colorLine)
	b.WriteString(";\"><p style=\"margin:0;font-size:11px;color:")
	b.WriteString(colorMuted)
	b.WriteString(";\">Familjedigest</p></td></tr>")

	b.WriteString("</table></td></tr></table></body></html>")
	return b.String()
}

func writeHeaderPill(b *strings.Builder, text, textColor string) {
	b.WriteString("<span style=\"display:inline-block;background:")
	b.WriteString(colorLine)
	b.WriteString(";color:")
	b.WriteString(textColor)
	b.WriteString(";font-size:12px;font-weight:600;padding:5px 11px;border-radius:999px;margin:0 6px 6px 0;\">")
	b.WriteString(html.EscapeString(text))
	b.WriteString("</span>")
}

// themeStyleCheatSheet is embedded in the system prompt so the model uses the same inline styles.
func themeStyleCheatSheet() string {
	return "STYLING (email-safe inline styles; use these exact style attribute values): " +
		"h2: style=\"" + styleH2 + "\" " +
		"h3: style=\"" + styleH3 + "\" " +
		"pill (child/label): <span style=\"" + stylePill + "\">…</span> " +
		"pill strong (child name in card header): <span style=\"" + stylePillStrong + "\">…</span> " +
		"pill muted: <span style=\"" + stylePillMuted + "\">…</span> " +
		"pill urgent (deadline): <span style=\"" + stylePillUrgent + "\">…</span> " +
		"data table: style=\"" + styleTable + "\" with header row background #1E222B and th style=\"" + styleTh + "\" " +
		"td primary style=\"" + styleTd + "\" td muted style=\"" + styleTdMuted + "\" " +
		"alternate body rows with background #171A21 and #1E222B. " +
		"School card: outer table border 1px solid #2A2F3A; header cell background #1E222B; body cell background #171A21. "
}
