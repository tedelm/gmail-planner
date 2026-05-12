package config

import "strings"

// BuildDigestBaseQuery returns the Gmail list `q` base string for digest mode
// (GMAIL_DIGEST_QUERY / -q only). Digest labels are applied per search in the CLI.
func BuildDigestBaseQuery(dcfg *DigestConfig, qOverride string) string {
	if dcfg == nil {
		return strings.TrimSpace(qOverride)
	}
	q := strings.TrimSpace(qOverride)
	if q == "" {
		q = strings.TrimSpace(dcfg.GmailDigestQuery)
	}
	return q
}

// BuildDigestGmailQuery is an alias for BuildDigestBaseQuery (legacy name).
// Multiple labels are no longer AND-merged here; each label uses a separate list call.
func BuildDigestGmailQuery(dcfg *DigestConfig, qOverride string) string {
	return BuildDigestBaseQuery(dcfg, qOverride)
}

// JoinGmailQueryParts joins non-empty trimmed parts with a single space for Gmail `q`.
func JoinGmailQueryParts(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, " ")
}

// DigestLabelSearchTerm returns a Gmail search fragment for one label name (label: or label:"...").
func DigestLabelSearchTerm(name string) string {
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
