package calendar

// Event is a simplified Google Calendar event for digest prompts.
type Event struct {
	ID         string
	Summary    string
	StartLocal string // Europe/Stockholm: "tisdag 2006-01-02 15:04" or all-day "tisdag 2006-01-02"
	EndLocal   string
	AllDay     bool
	Location   string
}
