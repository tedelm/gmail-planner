package calendar

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	calendarapi "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// Client wraps a Google Calendar API service.
type Client struct {
	svc *calendarapi.Service
}

// NewClient builds a Calendar client from an already-authenticated HTTP client
// (same OAuth token as Gmail).
func NewClient(ctx context.Context, hc *http.Client) (*Client, error) {
	if hc == nil {
		return nil, fmt.Errorf("http client is nil")
	}
	svc, err := calendarapi.NewService(ctx, option.WithHTTPClient(hc))
	if err != nil {
		return nil, fmt.Errorf("calendar service: %w", err)
	}
	return &Client{svc: svc}, nil
}

// ListEvents returns events on calendarID with start in [from, to), ordered by start time.
// Recurring events are expanded into single instances.
func (c *Client) ListEvents(ctx context.Context, calendarID string, from, to time.Time) ([]Event, error) {
	if c == nil || c.svc == nil {
		return nil, fmt.Errorf("calendar client is nil")
	}
	calendarID = strings.TrimSpace(calendarID)
	if calendarID == "" {
		return nil, fmt.Errorf("calendar ID is empty")
	}
	if !to.After(from) {
		return nil, fmt.Errorf("time window empty: to must be after from")
	}

	loc := stockholmLocation()
	var out []Event
	pageToken := ""
	for {
		call := c.svc.Events.List(calendarID).
			TimeMin(from.UTC().Format(time.RFC3339)).
			TimeMax(to.UTC().Format(time.RFC3339)).
			SingleEvents(true).
			OrderBy("startTime").
			MaxResults(250)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		resp, err := call.Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("list calendar events: %w", err)
		}
		for _, item := range resp.Items {
			if item == nil {
				continue
			}
			ev, ok := apiEventToEvent(item, loc)
			if !ok {
				continue
			}
			out = append(out, ev)
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return out, nil
}

func apiEventToEvent(item *calendarapi.Event, loc *time.Location) (Event, bool) {
	if item.Start == nil {
		return Event{}, false
	}
	startLocal, endLocal, allDay, ok := formatEventBounds(item.Start, item.End, loc)
	if !ok {
		return Event{}, false
	}
	summary := strings.TrimSpace(item.Summary)
	if summary == "" {
		summary = "(utan titel)"
	}
	return Event{
		ID:         item.Id,
		Summary:    summary,
		StartLocal: startLocal,
		EndLocal:   endLocal,
		AllDay:     allDay,
		Location:   strings.TrimSpace(item.Location),
	}, true
}

func formatEventBounds(start, end *calendarapi.EventDateTime, loc *time.Location) (startLocal, endLocal string, allDay bool, ok bool) {
	if start == nil {
		return "", "", false, false
	}
	if d := strings.TrimSpace(start.Date); d != "" {
		allDay = true
		t, err := time.ParseInLocation("2006-01-02", d, loc)
		if err != nil {
			return "", "", false, false
		}
		startLocal = FormatLocalWithWeekday(t, false)
		if end != nil {
			if ed := strings.TrimSpace(end.Date); ed != "" {
				if et, err := time.ParseInLocation("2006-01-02", ed, loc); err == nil {
					endLocal = FormatLocalWithWeekday(et, false)
				}
			}
		}
		return startLocal, endLocal, true, true
	}
	dt := strings.TrimSpace(start.DateTime)
	if dt == "" {
		return "", "", false, false
	}
	t, err := time.Parse(time.RFC3339, dt)
	if err != nil {
		return "", "", false, false
	}
	startLocal = FormatLocalWithWeekday(t.In(loc), true)
	if end != nil {
		if edt := strings.TrimSpace(end.DateTime); edt != "" {
			if et, err := time.Parse(time.RFC3339, edt); err == nil {
				endLocal = FormatLocalWithWeekday(et.In(loc), true)
			}
		}
	}
	return startLocal, endLocal, false, true
}

var swedishWeekdays = [...]string{
	time.Sunday:    "söndag",
	time.Monday:    "måndag",
	time.Tuesday:   "tisdag",
	time.Wednesday: "onsdag",
	time.Thursday:  "torsdag",
	time.Friday:    "fredag",
	time.Saturday:  "lördag",
}

// FormatLocalWithWeekday formats t as Swedish weekday plus date, optionally with time.
// Examples: "tisdag 2026-05-12 18:00", "måndag 2026-05-12".
func FormatLocalWithWeekday(t time.Time, withTime bool) string {
	day := swedishWeekdays[t.Weekday()]
	if withTime {
		return day + " " + t.Format("2006-01-02 15:04")
	}
	return day + " " + t.Format("2006-01-02")
}

// FormatStockholmDate formats now (or any instant) as a Stockholm calendar date with weekday.
func FormatStockholmDate(t time.Time) string {
	return FormatLocalWithWeekday(t.In(stockholmLocation()), false)
}

func stockholmLocation() *time.Location {
	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		return time.UTC
	}
	return loc
}
