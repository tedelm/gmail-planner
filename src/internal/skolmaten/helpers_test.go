package skolmaten

import (
	"strings"
	"testing"
	"time"
)

func TestParseMenuResponse_ExtractsDaysAndMeals(t *testing.T) {
	const fixture = `{
  "name": "Lunds kommun - Österskolans kök",
  "WeekState": {
    "week": 48,
    "year": 2026,
    "Days": [
      {
        "date": "2026-11-24T00:00:00.000Z",
        "Meals": [
          {"name": "Currygryta med ananas"},
          {"name": "Pastagratäng med färskost och oregano"},
          {"name": "  "}
        ]
      },
      {
        "date": "2026-11-25T00:00:00.000Z",
        "Meals": [
          {"name": "Sprödbakad fisk"}
        ]
      }
    ]
  },
  "School": {"name": "Österskolan"}
}`
	sw, err := ParseMenuResponse([]byte(fixture), "osterskolan")
	if err != nil {
		t.Fatalf("ParseMenuResponse: %v", err)
	}
	if sw.Slug != "osterskolan" || sw.SchoolName != "Österskolan" {
		t.Fatalf("identity: %#v", sw)
	}
	if sw.Year != 2026 || sw.Week != 48 {
		t.Fatalf("week: year=%d week=%d", sw.Year, sw.Week)
	}
	if len(sw.Days) != 2 {
		t.Fatalf("days: got %d", len(sw.Days))
	}
	if sw.Days[0].Date != "2026-11-24" {
		t.Fatalf("date: got %q", sw.Days[0].Date)
	}
	if len(sw.Days[0].Meals) != 2 || sw.Days[0].Meals[1] != "Pastagratäng med färskost och oregano" {
		t.Fatalf("meals: %#v", sw.Days[0].Meals)
	}
}

func TestParseMenuResponse_InvalidJSON(t *testing.T) {
	_, err := ParseMenuResponse([]byte(`{`), "x")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestISOWeeksCovering_SingleWeek(t *testing.T) {
	loc := time.UTC
	// Wednesday in ISO week 48 of 2026
	from := time.Date(2026, 11, 25, 12, 0, 0, 0, loc)
	got := ISOWeeksCovering(from, 1, loc)
	if len(got) != 1 || got[0].Year != 2026 || got[0].Week != 48 {
		t.Fatalf("got %#v", got)
	}
}

func TestISOWeeksCovering_ConsecutiveWeeks(t *testing.T) {
	loc := time.UTC
	from := time.Date(2026, 11, 25, 12, 0, 0, 0, loc)
	got := ISOWeeksCovering(from, 2, loc)
	if len(got) != 2 {
		t.Fatalf("expected 2 weeks, got %#v", got)
	}
	if got[0].Week != 48 || got[1].Week != 49 {
		t.Fatalf("got %#v", got)
	}
}

func TestNextISOWeek_YearBoundary(t *testing.T) {
	y, w := nextISOWeek(2025, 52)
	if y != 2026 || w != 1 {
		// 2025-W52 → 2026-W01 (2025 has 52 ISO weeks)
		t.Fatalf("got year=%d week=%d", y, w)
	}
}

func TestNormalizeAPIDate(t *testing.T) {
	if got := normalizeAPIDate("2026-11-24T00:00:00.000Z"); got != "2026-11-24" {
		t.Fatalf("got %q", got)
	}
	if got := normalizeAPIDate("2026-11-24"); got != "2026-11-24" {
		t.Fatalf("got %q", got)
	}
}

func TestParseMenuResponse_FallsBackToMenuName(t *testing.T) {
	fixture := `{"name":"Fallback kök","WeekState":{"week":1,"year":2026,"Days":[]},"School":{"name":""}}`
	sw, err := ParseMenuResponse([]byte(fixture), "slug")
	if err != nil {
		t.Fatal(err)
	}
	if sw.SchoolName != "Fallback kök" {
		t.Fatalf("got %q", sw.SchoolName)
	}
	if !strings.HasPrefix(sw.Slug, "slug") {
		t.Fatalf("slug: %q", sw.Slug)
	}
}
