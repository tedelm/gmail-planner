package gmail

// InboxMessage is a JSON-friendly summary of a Gmail message.
type InboxMessage struct {
	ID       string `json:"id"`
	ThreadID string `json:"threadId"`
	Headline string `json:"headline"`
	Body     string `json:"body"`
}
