package gmail

// InboxMessage is a JSON-friendly summary of a Gmail message.
type InboxMessage struct {
	ID       string `json:"id"`
	ThreadID string `json:"threadId"`
	Headline string `json:"headline"`
	// From is the decoded From header when present.
	From string `json:"from,omitempty"`
	Body string `json:"body"`
	// DateLocal is the message internal date in Europe/Stockholm (YYYY-MM-DD HH:MM), or empty if unknown.
	DateLocal string `json:"dateLocal,omitempty"`
	// DigestLabel names the Gmail label search(es) that included this message (digest mode); comma-separated if several.
	DigestLabel string `json:"digestLabel,omitempty"`
}
