package digest

import (
	"strings"
	"testing"
)

func TestParseDigestOutput_JSON(t *testing.T) {
	in := `{"text":"Hej","html":"<h2>Hej</h2><ul><li>Test</li></ul>"}`
	out, err := parseDigestOutput(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Text != "Hej" {
		t.Fatalf("text: got %q", out.Text)
	}
	if out.HTML == "" {
		t.Fatal("expected html")
	}
}

func TestParseDigestOutput_AcceptsFencedJSON(t *testing.T) {
	in := "```json\n{\"text\":\"Hej\",\"html\":\"<h2>Hej</h2>\"}\n```"
	out, err := parseDigestOutput(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Text != "Hej" || !strings.Contains(out.HTML, "<h2>Hej</h2>") {
		t.Fatalf("got %#v", out)
	}
}

func TestNormalizeDigestOutput_UnescapesEntityHTML(t *testing.T) {
	in := DigestOutput{
		Text: "Barnen",
		HTML: "&lt;h2&gt;Barnen&lt;/h2&gt;&lt;ul&gt;&lt;li&gt;Läxa (1)&lt;/li&gt;&lt;/ul&gt;",
	}
	out := normalizeDigestOutput(in)
	if !strings.Contains(out.HTML, "<h2>Barnen</h2>") {
		t.Fatalf("html not unescaped: %q", out.HTML)
	}
	if strings.Contains(out.Text, "<h2>") || strings.Contains(out.Text, "&lt;") {
		t.Fatalf("text should be plain: %q", out.Text)
	}
	if !strings.Contains(out.Text, "Barnen") || !strings.Contains(out.Text, "Läxa") {
		t.Fatalf("text missing content: %q", out.Text)
	}
}

func TestNormalizeDigestOutput_UsesTextWhenHTMLMissingMarkup(t *testing.T) {
	in := DigestOutput{
		Text: "<h2>Skola</h2><ul><li>Matte (2)</li></ul>",
		HTML: "plain summary without tags",
	}
	out := normalizeDigestOutput(in)
	if !strings.Contains(out.HTML, "<h2>Skola</h2>") {
		t.Fatalf("expected html from text: %q", out.HTML)
	}
	if strings.Contains(out.Text, "<h2>") {
		t.Fatalf("text should strip tags: %q", out.Text)
	}
}

func TestNormalizeDigestOutput_PlainFallback(t *testing.T) {
	in := DigestOutput{Text: "Just text <notatag", HTML: ""}
	out := normalizeDigestOutput(in)
	if !strings.Contains(out.HTML, "<!doctype html>") && !strings.Contains(out.HTML, "<pre") {
		t.Fatalf("expected simple html wrap: %q", out.HTML)
	}
	if strings.Contains(out.HTML, "<notatag") {
		t.Fatalf("angle brackets should be escaped in fallback: %q", out.HTML)
	}
}

func TestHtmlFragmentToPlainText(t *testing.T) {
	got := htmlFragmentToPlainText("<h2>Titel</h2><ul><li>A</li><li>B</li></ul>")
	if strings.Contains(got, "<") {
		t.Fatalf("tags left: %q", got)
	}
	if !strings.Contains(got, "Titel") || !strings.Contains(got, "A") {
		t.Fatalf("got %q", got)
	}
}
