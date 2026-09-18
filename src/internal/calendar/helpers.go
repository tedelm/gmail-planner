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
		startLocal = d
		if end != nil {
			endLocal = strings.TrimSpace(end.Date)
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
	startLocal = t.In(loc).Format("2006-01-02 15:04")
	if end != nil {
		if edt := strings.TrimSpace(end.DateTime); edt != "" {
			if et, err := time.Parse(time.RFC3339, edt); err == nil {
				endLocal = et.In(loc).Format("2006-01-02 15:04")
			}
		}
	}
	return startLocal, endLocal, false, true
}

func stockholmLocation() *time.Location {
	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		return time.UTC
	}
	return loc
}
