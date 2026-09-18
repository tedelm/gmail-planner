package calendar

import (
	"testing"
	"time"

	calendarapi "google.golang.org/api/calendar/v3"
)

func TestFormatEventBounds_DateTime(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		t.Fatal(err)
	}
	start := &calendarapi.EventDateTime{DateTime: "2026-05-12T16:00:00Z"}
	end := &calendarapi.EventDateTime{DateTime: "2026-05-12T17:30:00Z"}
	s, e, allDay, ok := formatEventBounds(start, end, loc)
	if !ok || allDay {
		t.Fatalf("ok=%v allDay=%v", ok, allDay)
	}
	if s != "2026-05-12 18:00" { // UTC+2 in May
		t.Fatalf("start: got %q", s)
	}
	if e != "2026-05-12 19:30" {
		t.Fatalf("end: got %q", e)
	}
}

func TestFormatEventBounds_AllDay(t *testing.T) {
	loc := time.UTC
	start := &calendarapi.EventDateTime{Date: "2026-05-12"}
	end := &calendarapi.EventDateTime{Date: "2026-05-13"}
	s, e, allDay, ok := formatEventBounds(start, end, loc)
	if !ok || !allDay {
		t.Fatalf("ok=%v allDay=%v", ok, allDay)
	}
	if s != "2026-05-12" || e != "2026-05-13" {
		t.Fatalf("got start=%q end=%q", s, e)
	}
}

func TestApiEventToEvent_EmptySummary(t *testing.T) {
	loc := time.UTC
	ev, ok := apiEventToEvent(&calendarapi.Event{
		Id:    "x",
		Start: &calendarapi.EventDateTime{Date: "2026-01-01"},
	}, loc)
	if !ok {
		t.Fatal("expected ok")
	}
	if ev.Summary != "(utan titel)" {
		t.Fatalf("summary: %q", ev.Summary)
	}
}
