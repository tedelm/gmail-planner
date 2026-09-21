package gmail

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestBuildGmailSendRaw_ContainsHeadersAndCRLF(t *testing.T) {
	raw, err := BuildGmailSendRaw("me@example.com", "family@example.com", "Hello", "Line1\nLine2")
	if err != nil {
		t.Fatal(err)
	}
	dec, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("decode raw: %v", err)
	}
	s := string(dec)
	if !strings.Contains(s, "From: me@example.com") {
		t.Fatalf("missing From: %q", s)
	}
	if !strings.Contains(s, "To: family@example.com") {
		t.Fatalf("missing To")
	}
	if !strings.Contains(s, "Subject: Hello") {
		t.Fatalf("missing Subject")
	}
	if !strings.Contains(s, "Content-Type: text/plain; charset=UTF-8") {
		t.Fatalf("missing content-type")
	}
	if !strings.Contains(s, "Line1\r\nLine2") && !strings.Contains(s, "Line1\r\nLine2\r\n") {
		t.Fatalf("body CRLF: %q", s)
	}
}

func TestBuildGmailSendRaw_FromToRequired(t *testing.T) {
	_, err := BuildGmailSendRaw("", "a@b.com", "s", "b")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSubjectEncoding_AllowsSwedishChars(t *testing.T) {
	raw, err := BuildGmailSendRaw("me@example.com", "family@example.com", "ÅÄÖ", "Body")
	if err != nil {
		t.Fatal(err)
	}
	dec, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("decode raw: %v", err)
	}
	s := string(dec)
	if !strings.Contains(s, "Subject: =?utf-8?q?") {
		t.Fatalf("expected RFC2047 encoded subject, got: %q", s)
	}
}

func TestTextToSimpleHTML_EscapesAndKeepsUnicode(t *testing.T) {
	html := TextToSimpleHTML("Hej ÅÄÖ <tag>")
	if !strings.Contains(html, "Hej ÅÄÖ") {
		t.Fatalf("unicode lost: %q", html)
	}
	if strings.Contains(html, "<tag>") {
		t.Fatalf("expected html escaping: %q", html)
	}
}

func TestBuildGmailSendRawHTML_QuotedBoundaryAndDocumentWrap(t *testing.T) {
	raw, err := BuildGmailSendRawHTML("me@example.com", "to@example.com", "Subj", "plain", "<h2>Hej</h2>")
	if err != nil {
		t.Fatal(err)
	}
	dec, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	s := string(dec)
	if !strings.Contains(s, `boundary="gmailplanner_`) {
		t.Fatalf("expected quoted boundary: %q", s)
	}
	if !strings.Contains(s, "Content-Type: text/html; charset=UTF-8") {
		t.Fatalf("missing html content-type: %q", s)
	}
	if !strings.Contains(s, "<!doctype html>") || !strings.Contains(s, "<h2>Hej</h2>") {
		t.Fatalf("expected html document wrap: %q", s)
	}
	if !strings.Contains(s, "plain") {
		t.Fatalf("missing plain body: %q", s)
	}
}

func TestEnsureHTMLDocument_WrapsFragment(t *testing.T) {
	got := ensureHTMLDocument("<h2>X</h2>")
	if !strings.Contains(got, "<!doctype html>") || !strings.Contains(got, "<h2>X</h2>") {
		t.Fatalf("got %q", got)
	}
	full := "<!doctype html><html><body>Y</body></html>"
	if ensureHTMLDocument(full) != full {
		t.Fatal("should not double-wrap")
	}
}
