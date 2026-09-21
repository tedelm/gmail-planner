package calendar

import (
	"strings"
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
	// 2026-05-12 is a Tuesday; UTC+2 in May → 18:00 / 19:30
	if s != "tisdag 2026-05-12 18:00" {
		t.Fatalf("start: got %q", s)
	}
	if e != "tisdag 2026-05-12 19:30" {
		t.Fatalf("end: got %q", e)
	}
}

func TestFormatEventBounds_AllDay(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		t.Fatal(err)
	}
	start := &calendarapi.EventDateTime{Date: "2026-05-12"}
	end := &calendarapi.EventDateTime{Date: "2026-05-13"}
	s, e, allDay, ok := formatEventBounds(start, end, loc)
	if !ok || !allDay {
		t.Fatalf("ok=%v allDay=%v", ok, allDay)
	}
	if s != "tisdag 2026-05-12" || e != "onsdag 2026-05-13" {
		t.Fatalf("got start=%q end=%q", s, e)
	}
}

func TestApiEventToEvent_EmptySummary(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		t.Fatal(err)
	}
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
	if !strings.HasPrefix(ev.StartLocal, "torsdag ") {
		t.Fatalf("StartLocal weekday: %q", ev.StartLocal)
	}
}

func TestFormatStockholmDate_IncludesWeekday(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		t.Fatal(err)
	}
	// Fixed instant: Wednesday 2026-09-16 12:00 Stockholm (CEST)
	ti := time.Date(2026, 9, 16, 12, 0, 0, 0, loc)
	got := FormatStockholmDate(ti)
	if got != "onsdag 2026-09-16" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatLocalWithWeekday(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		t.Fatal(err)
	}
	ti := time.Date(2026, 5, 12, 18, 0, 0, 0, loc)
	if got := FormatLocalWithWeekday(ti, true); got != "tisdag 2026-05-12 18:00" {
		t.Fatalf("with time: %q", got)
	}
	if got := FormatLocalWithWeekday(ti, false); got != "tisdag 2026-05-12" {
		t.Fatalf("date only: %q", got)
	}
}
