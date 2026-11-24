package digest

import (
	"strings"
	"testing"

	"github.com/tedelm/gmail-planner/internal/calendar"
	"github.com/tedelm/gmail-planner/internal/gmail"
	"github.com/tedelm/gmail-planner/internal/skolmaten"
)

func TestTruncateRunes(t *testing.T) {
	got := TruncateRunes("abcdef", 4)
	if !strings.Contains(got, "…[truncated]") || strings.Contains(got, "ef") {
		t.Fatalf("got %q", got)
	}
	if TruncateRunes("hi", 10) != "hi" {
		t.Fatal("short string changed")
	}
}

func TestBuildMailDigestPrompt_BudgetOmitsTail(t *testing.T) {
	msgs := make([]gmail.InboxMessage, 5)
	for i := range msgs {
		msgs[i] = gmail.InboxMessage{
			ID:       "id",
			ThreadID: "th",
			Headline: "Subj",
			Body:     strings.Repeat("x", 200),
		}
	}
	_, omitted := BuildMailDigestPrompt(msgs, 50, 400)
	if omitted == 0 {
		t.Fatal("expected some messages omitted")
	}
}

func TestBuildMailDigestPrompt_IncludesDateWhenPresent(t *testing.T) {
	msgs := []gmail.InboxMessage{{
		ID:        "id1",
		ThreadID:  "th1",
		Headline:  "Hej",
		Body:      "Body",
		DateLocal: "2026-05-11 14:00",
	}}
	prompt, _ := BuildMailDigestPrompt(msgs, 100, 5000)
	if !strings.Contains(prompt, "Date: 2026-05-11 14:00") {
		t.Fatalf("expected date in prompt: %s", prompt)
	}
	if !strings.Contains(prompt, "--- Source 1 ---") {
		t.Fatalf("expected source header: %s", prompt)
	}
}

func TestBuildMailDigestPrompt_IncludesDigestLabelWhenSet(t *testing.T) {
	msgs := []gmail.InboxMessage{{
		ID:          "id1",
		ThreadID:    "th1",
		Headline:    "Hej",
		Body:        "Body",
		DateLocal:   "2026-05-11 14:00",
		DigestLabel: "Work",
	}}
	prompt, _ := BuildMailDigestPrompt(msgs, 100, 5000)
	if !strings.Contains(prompt, "Label: Work") {
		t.Fatalf("expected label line: %s", prompt)
	}
}

func TestBuildMailDigestPrompt_IncludesFromWhenSet(t *testing.T) {
	msgs := []gmail.InboxMessage{{
		ID:        "id1",
		ThreadID:  "th1",
		Headline:  "Match",
		From:      "SportAdmin <noreply@sportadmin.se>",
		Body:      "Body",
		DateLocal: "2026-05-11 14:00",
	}}
	prompt, _ := BuildMailDigestPrompt(msgs, 100, 5000)
	if !strings.Contains(prompt, "From: SportAdmin <noreply@sportadmin.se>") {
		t.Fatalf("expected From line: %s", prompt)
	}
}

func TestBuildDigestPrompt_IncludesCalendarAfterEmail(t *testing.T) {
	msgs := []gmail.InboxMessage{{
		ID:        "m1",
		ThreadID:  "th1",
		Headline:  "Mail",
		Body:      "Hej",
		DateLocal: "2026-05-11 14:00",
	}}
	events := []calendar.Event{{
		ID:         "e1",
		Summary:    "Match",
		StartLocal: "tisdag 2026-05-12 18:00",
		EndLocal:   "tisdag 2026-05-12 19:30",
		Location:   "Hallen",
	}}
	prompt, cited, omittedMsgs, omittedEvents, omittedMenus := BuildDigestPrompt(msgs, events, nil, 100, 8000)
	if omittedMsgs != 0 || omittedEvents != 0 || omittedMenus != 0 {
		t.Fatalf("unexpected omit: msgs=%d events=%d menus=%d", omittedMsgs, omittedEvents, omittedMenus)
	}
	if !strings.Contains(prompt, "Type: email") || !strings.Contains(prompt, "Type: calendar") {
		t.Fatalf("expected email and calendar types: %s", prompt)
	}
	if !strings.Contains(prompt, "--- Source 2 ---") || !strings.Contains(prompt, "Subject: Match") {
		t.Fatalf("expected calendar as source 2: %s", prompt)
	}
	if !strings.Contains(prompt, "Location: Hallen") {
		t.Fatalf("expected location: %s", prompt)
	}
	if len(cited) != 2 || cited[1].Label != "Kalender" {
		t.Fatalf("cited: %#v", cited)
	}
}

func TestBuildDigestPrompt_IncludesSkolmatenAfterCalendar(t *testing.T) {
	msgs := []gmail.InboxMessage{{
		ID:        "m1",
		ThreadID:  "th1",
		Headline:  "Mail",
		Body:      "Hej",
		DateLocal: "2026-05-11 14:00",
	}}
	events := []calendar.Event{{
		ID:         "e1",
		Summary:    "Match",
		StartLocal: "tisdag 2026-05-12 18:00",
	}}
	menus := []skolmaten.SchoolWeek{{
		Slug:       "osterskolan",
		SchoolName: "Österskolan",
		Year:       2026,
		Week:       48,
		ChildNames: []string{"Annie"},
		Days: []skolmaten.DayMenu{
			{Date: "2026-11-24", Meals: []string{"Currygryta", "Pastagratäng"}},
		},
	}}
	prompt, cited, omittedMsgs, omittedEvents, omittedMenus := BuildDigestPrompt(msgs, events, menus, 100, 12000)
	if omittedMsgs != 0 || omittedEvents != 0 || omittedMenus != 0 {
		t.Fatalf("unexpected omit: msgs=%d events=%d menus=%d", omittedMsgs, omittedEvents, omittedMenus)
	}
	if !strings.Contains(prompt, "Type: skolmaten") {
		t.Fatalf("expected skolmaten type: %s", prompt)
	}
	if !strings.Contains(prompt, "--- Source 3 ---") || !strings.Contains(prompt, "Child: Annie") {
		t.Fatalf("expected skolmaten as source 3: %s", prompt)
	}
	if !strings.Contains(prompt, "Meals: Currygryta; Pastagratäng") {
		t.Fatalf("expected meals: %s", prompt)
	}
	if len(cited) != 3 || cited[2].Label != "Skolmaten" {
		t.Fatalf("cited: %#v", cited)
	}
}
