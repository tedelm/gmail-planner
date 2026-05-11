package gmail

// InboxMessage is a JSON-friendly summary of a Gmail message.
type InboxMessage struct {
	ID       string `json:"id"`
	ThreadID string `json:"threadId"`
	Headline string `json:"headline"`
	Body     string `json:"body"`
	// DateLocal is the message internal date in Europe/Stockholm (YYYY-MM-DD HH:MM), or empty if unknown.
	DateLocal string `json:"dateLocal,omitempty"`
}
