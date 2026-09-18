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
			"Citations: Each input block is labeled \"--- Source n ---\". For every bullet that draws on a source, "+
			"end the <li> text with an inline citation like (n) matching that Source number. "+
			"If a bullet combines multiple sources, use multiple citations like (1)(3). "+
			"When a source block includes a \"Label:\" line (Gmail label search), mention that label in the digest "+
			"(inline or short subheadings) so readers can see which label bucket each item came from. ",
		weeks, lang)

	b.WriteString(childAttributionInstructions(children))
	b.WriteString(homeworkInstructions())

	b.WriteString(
		"HTML STRUCTURE (fixed section order; omit any section or subsection that has no content; do not invent filler): " +
			"<h2>Kalender</h2> for upcoming Google Calendar events (times, places, titles chronologically when helpful). " +
			"Then <h2>Barnen</h2> with subsections <h3>Skola</h3> (school, fritids, class info")
	if len(children) > 0 {
		b.WriteString("; under Skola use one <h4>{child name}</h4> per configured child that has content, omit empty child headings; " +
			"if an item cannot be attributed, use <h4>Båda / oklart</h4>")
	}
	b.WriteString(
		") and " +
			"<h3>Föreningsliv</h3> (SportAdmin and other club/sports info for the children: matches, practices, cups, sign-ups, fees, times/places). " +
			"Put emails from SportAdmin (detect via From, Subject, or body) and other children's association/sports club mail under Föreningsliv. ")
	if len(children) > 0 {
		b.WriteString("In Föreningsliv bullets, name the child when label/class code/body identifies them. ")
	}
	b.WriteString(
		"Then <h2>Familj & praktiskt</h2> for other items that affect family planning, " +
			"then <h2>Deadlines & att göra</h2> for concrete deadlines and action items. " +
			"Use bullet lists (ul/li) under each heading. Do not invent other top-level section names. " +
			"At the END of the html (after the main digest), add a section <h2>Källor</h2> followed by <ol> where each <li> is " +
			"exactly: (n) Subject — Date using ONLY the Subject and Date lines from the corresponding Source block " +
			"(if Date was \"(saknas)\", omit the em dash and date part and use only the subject). " +
			"If the source had a Label line, append \" — \" and that label text to the Källor line. " +
			"For calendar sources, append \" — Kalender\" in Källor. " +
			"Do not invent source numbers; only use n values that appear in the input. " +
			"Return ONLY valid JSON (no markdown, no code fences) with this shape: " +
			"{\"text\":\"...plain text...\",\"html\":\"...HTML...\"}. " +
			"The html value MUST be a complete HTML fragment (no markdown) following the fixed section structure above. " +
			"The text field should mirror the same section headings, citations, and end with " +
			"a \"Källor\" section listing (n) Subject — Date lines, each with an appended \" — Label\" when that source had a Label line " +
			"(and \" — Kalender\" for calendar sources).")
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
		"Place each homework item under the correct child's Skola heading when attribution is known. " +
		"Summarize homework in depth and structurally: type of homework, deadline or week, and keep concrete content from the email " +
		"(full word lists for Veckans ord, specific tasks, page/chapter numbers, links) — do not reduce to \"there is homework\". " +
		"Do not invent words or tasks absent from the source; cite sources as usual. "
}
