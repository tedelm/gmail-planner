package digest

import (
	"strings"
	"testing"
	"time"
)

func TestWrapDigestEmailHTML_IncludesChromeAndInner(t *testing.T) {
	inner := `<h2 style="` + styleH2 + `">Kalender</h2><p>Inner content</p>`
	got := WrapDigestEmailHTML(inner, DigestEmailMeta{
		Weeks:       1,
		RangeLabel:  "16–22 nov 2026",
		SourceCount: 5,
	})
	for _, want := range []string{
		"<!DOCTYPE html>",
		colorBgOuter,
		colorCard,
		colorAccent,
		"Familjeöversikt",
		"Veckan framåt",
		"16–22 nov 2026",
		"7 dagar",
		"5 källor",
		"Inner content",
		"Kalender",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in wrapped HTML", want)
		}
	}
	if !strings.Contains(got, "<html") {
		t.Fatal("expected full HTML document")
	}
}

func TestFormatDigestRangeLabel_SameMonth(t *testing.T) {
	from := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
	got := FormatDigestRangeLabel(from, to)
	if got != "16–22 nov 2026" {
		t.Fatalf("got %q", got)
	}
}

func TestDigestRangeForWeeks(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, 11, 16, 15, 0, 0, 0, loc)
	start, end := DigestRangeForWeeks(now, 1, loc)
	if start.Day() != 16 || end.Day() != 22 {
		t.Fatalf("start=%v end=%v", start, end)
	}
}

func TestStripHTMLSourcesFooter_StyledH2(t *testing.T) {
	in := `<p>body</p><h2 style="color:red">Källor</h2><ol><li>old</li></ol>`
	got := stripHTMLSourcesFooter(in)
	if strings.Contains(got, "Källor") || strings.Contains(got, "old") {
		t.Fatalf("expected styled Källor stripped: %q", got)
	}
	if !strings.Contains(got, "<p>body</p>") {
		t.Fatalf("body lost: %q", got)
	}
}
