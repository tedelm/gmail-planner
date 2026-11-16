package digest

import (
	"fmt"
	"strings"

	"github.com/tedelm/gmail-planner/internal/config"
)

// buildFamilyDigestSystemPrompt builds the OpenAI system message for the family digest.
func buildFamilyDigestSystemPrompt(weeks int, lang, today string, children []config.DigestChild) string {
	if weeks < 1 {
		weeks = 1
	}
	if lang == "" {
		lang = config.DefaultDigestLanguage
	}

	var b strings.Builder
	fmt.Fprintf(&b,
		"You help a family plan ahead. Summarize the following emails and Google Calendar events into a clear, actionable digest "+
			"covering roughly the next %d week(s). "+
			"Call out dates, times, locations, deadlines, and who should do what when the sources imply it. "+
			"Do not invent events or commitments not supported by the email or calendar text. "+
			"Calendar sources (Type: calendar) are authoritative for scheduled times; emails may add context or action items "+
			"but must not invent conflicting times for the same commitment. "+
			"IMPORTANT: Write the entire digest in %s. If the content is in another language, translate it as needed. "+
			"Keep names, email addresses, phone numbers, URLs, and exact dates/times intact. "+
			"Citations: Each input block is labeled \"--- Source n ---\". For every row or item that draws on a source, "+
			"include an inline citation like (n) matching that Source number (in the last table column or at the end of the text). "+
			"If an item combines multiple sources, use multiple citations like (1)(3). ",
		weeks, lang)

	b.WriteString(childAttributionInstructions(children))
	b.WriteString(homeworkInstructions())
	b.WriteString(themeStyleCheatSheet())

	b.WriteString(
		"HTML STRUCTURE (inner fragment only — no <html>/<body>, no outer wrapper, no header chrome, no Källor section; " +
			"fixed section order; omit any section or subsection that has no content; do not invent filler): " +
			"<h2 style=\"" + styleH2 + "\">Kalender</h2> then a data table with columns När | Vad | Var | (citation). " +
			"När cell: date as <strong> plus time on a second line in muted color. Vad: title plus optional child pill. " +
			"Then <h2 style=\"" + styleH2 + "\">Barnen</h2> with <h3 style=\"" + styleH3 + "\">Skola</h3> ")
	if len(children) > 0 {
		b.WriteString(
			"(under Skola: one school card table per configured child that has content — card header with strong name pill " +
				"plus optional Läxa/week pills; card body with title and details; omit empty children; " +
				"if an item cannot be attributed, use a card headed Båda / oklart). ")
	} else {
		b.WriteString("(school, fritids, class info as cards or a compact table). ")
	}
	b.WriteString(
		"Then <h3 style=\"" + styleH3 + "\">Föreningsliv</h3> " +
			"(SportAdmin and other club/sports info: matches, practices, cups, sign-ups, fees, times/places) as a table " +
			"Vem | Vad | Detalj with child pills and urgent pills when a reply deadline exists. " +
			"Put emails from SportAdmin (detect via From, Subject, or body) and other children's association/sports club mail under Föreningsliv. ")
	if len(children) > 0 {
		b.WriteString("In Föreningsliv rows, name the child when label/class code/body identifies them. ")
	}
	b.WriteString(
		"Then <h2 style=\"" + styleH2 + "\">Familj & praktiskt</h2> as stacked rows/cards with category pills, " +
			"then <h2 style=\"" + styleH2 + "\">Deadlines & att göra</h2> as a table När | Uppgift | Vem " +
			"(use urgent date pills for soonest deadlines). " +
			"Do not invent other top-level section names. " +
			"Do NOT include a Källor section — it is appended by the application. " +
			"Return ONLY valid JSON (no markdown, no code fences) with this shape: " +
			"{\"text\":\"...plain text...\",\"html\":\"...HTML...\"}. " +
			"The html value MUST be a complete HTML fragment with real tags (table, h2, h3, span) and the STYLING above " +
			"(do not entity-escape tags as &lt;h2&gt;; no markdown). " +
			"The text field MUST be plain text only — no HTML tags — mirroring the same section headings and citations, " +
			"ending without a Källor block (the application appends sources).")
	fmt.Fprintf(&b,
		"Do not include events that have already happened, use the information in the email or calendar "+
			"(or email sent/received date if no date is available in the email body) to determine if the event has already happened. Today's date is %s."+
			"Important: Always check the email header for week numbers and use them to determine if the event has already happened. "+
			"If the event has already happened, do not include it in the digest.",
		today)
	return b.String()
}

func childAttributionInstructions(children []config.DigestChild) string {
	if len(children) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("CHILD MAP (attribute school and homework items to the correct child; do not invent children): ")
	for i, c := range children {
		if i > 0 {
			b.WriteString(" ")
		}
		fmt.Fprintf(&b, "%s:", c.Name)
		if len(c.Labels) > 0 {
			fmt.Fprintf(&b, " Gmail Label substrings [%s];", strings.Join(c.Labels, ", "))
		}
		if len(c.ClassCodes) > 0 {
			fmt.Fprintf(&b, " class codes (case-insensitive in Subject/From/body) [%s];", strings.Join(c.ClassCodes, ", "))
		}
	}
	b.WriteString(
		" Attribution priority: (1) Source Label: line contains a child label substring, " +
			"(2) class code match in Subject/From/body, (3) child's first name clearly in the text. " +
			"If a source has multiple labels, the more specific child label wins. " +
			"Very short or ambiguous class codes (e.g. only digits like \"17\") may be used only when school/class context is clear " +
			"(avoid matching random years or dates). " +
			"If still unclear, put under Båda / oklart — never guess. ")
	return b.String()
}

func homeworkInstructions() string {
	return "HOMEWORK: Detect homework via Subject/body (e.g. läxa, Veckans ord, hemläxa, inlämning, skolplattform). " +
		"Place each homework item under the correct child's Skola card when attribution is known. " +
		"Summarize homework in depth and structurally: type of homework, deadline or week, and keep concrete content from the email " +
		"(full word lists for Veckans ord in a small two-column table when helpful, specific tasks, page/chapter numbers, links) — do not reduce to \"there is homework\". " +
		"Do not invent words or tasks absent from the source; cite sources as usual. "
}
