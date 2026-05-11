package config

import "strings"

// BuildDigestGmailQuery returns the Gmail list `q` string for digest mode.
// qOverride (e.g. CLI -q) replaces GMAIL_DIGEST_QUERY when non-empty after trim.
// Each entry in dcfg.GmailDigestLabels is appended as a label: search term (AND).
func BuildDigestGmailQuery(dcfg *DigestConfig, qOverride string) string {
	if dcfg == nil {
		return strings.TrimSpace(qOverride)
	}
	q := strings.TrimSpace(qOverride)
	if q == "" {
		q = strings.TrimSpace(dcfg.GmailDigestQuery)
	}
	parts := make([]string, 0, 1+len(dcfg.GmailDigestLabels))
	if q != "" {
		parts = append(parts, q)
	}
	for _, label := range dcfg.GmailDigestLabels {
		if term := gmailLabelSearchTerm(label); term != "" {
			parts = append(parts, term)
		}
	}
	return strings.Join(parts, " ")
}

func gmailLabelSearchTerm(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if strings.ContainsAny(name, " \t\"\\") {
		var b strings.Builder
		b.Grow(len(name) + len(`label:"`))
		b.WriteString(`label:"`)
		for _, r := range name {
			switch r {
			case '\\':
				b.WriteString(`\\`)
			case '"':
				b.WriteString(`\"`)
			default:
				b.WriteRune(r)
			}
		}
		b.WriteByte('"')
		return b.String()
	}
	return "label:" + name
}
