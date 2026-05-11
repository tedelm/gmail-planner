package digest

import (
	"strings"
	"testing"

	"github.com/tedelm/gmail-planner/internal/gmail"
)

func TestMessagesCitedInPrompt(t *testing.T) {
	msgs := []gmail.InboxMessage{
		{Headline: "A"},
		{Headline: "B"},
		{Headline: "C"},
	}
	got := messagesCitedInPrompt(msgs, 1)
	if len(got) != 2 || got[0].Headline != "A" || got[1].Headline != "B" {
		t.Fatalf("cited with omitted=1: got %#v", got)
	}
	if messagesCitedInPrompt(msgs, len(msgs)) != nil {
		t.Fatal("all omitted should yield nil")
	}
}

func TestEnsureDigestSources_emptyCited(t *testing.T) {
	in := DigestOutput{Text: "body", HTML: "<p>body</p>"}
	out := EnsureDigestSources(in, nil)
	if out.Text != in.Text || out.HTML != in.HTML {
		t.Fatalf("expected unchanged, got text=%q html=%q", out.Text, out.HTML)
	}
}

func TestEnsureDigestSources_appendsHTMLAndText(t *testing.T) {
	cited := []gmail.InboxMessage{
		{Headline: "Utflykt", DateLocal: "2026-05-12 09:00"},
		{Headline: "Läxa", DateLocal: "(saknas)"},
	}
	in := DigestOutput{
		Text: "Planering.\n\n- Punkt (1)(2)",
		HTML: "<h2>Vecka</h2><ul><li>Punkt (1)(2)</li></ul>",
	}
	out := EnsureDigestSources(in, cited)

	if !strings.Contains(out.HTML, "<h2>Källor</h2>") || !strings.Contains(out.HTML, "<ol>") {
		t.Fatalf("html missing Källor block: %s", out.HTML)
	}
	if !strings.Contains(out.HTML, "<li>(1) Utflykt — 2026-05-12 09:00</li>") {
		t.Fatalf("html li 1: %s", out.HTML)
	}
	if !strings.Contains(out.HTML, "<li>(2) Läxa</li>") {
		t.Fatalf("html li 2 (no date): %s", out.HTML)
	}
	if n := strings.Count(out.HTML, "<h2>Källor</h2>"); n != 1 {
		t.Fatalf("expected single Källor h2, count=%d", n)
	}

	if !strings.Contains(out.Text, "\n\nKällor\n") {
		t.Fatalf("text missing Källor heading: %q", out.Text)
	}
	if !strings.Contains(out.Text, "(1) Utflykt — 2026-05-12 09:00") || !strings.Contains(out.Text, "(2) Läxa") {
		t.Fatalf("text lines: %q", out.Text)
	}
}

func TestEnsureDigestSources_stripsExistingFooter(t *testing.T) {
	cited := []gmail.InboxMessage{{Headline: "Ny", DateLocal: "2026-01-02"}}
	in := DigestOutput{
		Text: "Brödtext\n\nKällor\n(9) Gammal — fel",
		HTML: "<p>x</p><h2>Källor</h2><ol><li>(9) Gammal</li></ol>",
	}
	out := EnsureDigestSources(in, cited)
	if strings.Count(out.HTML, "<h2>Källor</h2>") != 1 {
		t.Fatalf("html: %s", out.HTML)
	}
	if strings.Contains(out.HTML, "Gammal") {
		t.Fatalf("old footer should be stripped: %s", out.HTML)
	}
	if strings.Contains(out.Text, "Gammal") {
		t.Fatalf("old text footer should be stripped: %q", out.Text)
	}
	if !strings.Contains(out.Text, "(1) Ny — 2026-01-02") {
		t.Fatalf("text: %q", out.Text)
	}
}

func TestEnsureDigestSources_escapesHTMLInSubject(t *testing.T) {
	cited := []gmail.InboxMessage{{Headline: "A & B <tag>", DateLocal: "2026-01-01"}}
	out := EnsureDigestSources(DigestOutput{HTML: "<p>x</p>", Text: "x"}, cited)
	if strings.Contains(out.HTML, "<tag>") {
		t.Fatalf("subject should be escaped: %s", out.HTML)
	}
	if !strings.Contains(out.HTML, "&lt;tag&gt;") {
		t.Fatalf("expected escaped tag: %s", out.HTML)
	}
}
