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
