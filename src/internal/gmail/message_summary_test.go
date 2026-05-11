package gmail

import (
	"encoding/base64"
	"testing"

	gmailapi "google.golang.org/api/gmail/v1"
)

func TestMessageToInboxSummary_SubjectAndPlainBody(t *testing.T) {
	plain := "hello world"
	enc := base64.RawURLEncoding.EncodeToString([]byte(plain))
	m := &gmailapi.Message{
		Id:       "m1",
		ThreadId: "t1",
		Payload: &gmailapi.MessagePart{
			MimeType: "multipart/alternative",
			Headers: []*gmailapi.MessagePartHeader{
				{Name: "Subject", Value: "Test =?UTF-8?B?8J+RqfCflZE=?="},
			},
			Parts: []*gmailapi.MessagePart{
				{
					MimeType: "text/plain",
					Body:     &gmailapi.MessagePartBody{Data: enc},
				},
			},
		},
	}
	got := messageToInboxSummary(m)
	if got.ID != "m1" || got.ThreadID != "t1" {
		t.Fatalf("ids: got %+v", got)
	}
	if got.Headline == "" {
		t.Fatal("expected decoded headline")
	}
	if got.Body != plain {
		t.Fatalf("body: got %q want %q", got.Body, plain)
	}
}

func TestMessageToInboxSummary_FallbackSnippetWhenNoPlain(t *testing.T) {
	m := &gmailapi.Message{
		Id:       "m2",
		ThreadId: "t2",
		Snippet:  "preview text",
		Payload: &gmailapi.MessagePart{
			MimeType: "text/html",
			Headers:  []*gmailapi.MessagePartHeader{{Name: "Subject", Value: "Hi"}},
			Body:     &gmailapi.MessagePartBody{Data: base64.RawURLEncoding.EncodeToString([]byte("<p>x</p>"))},
		},
	}
	got := messageToInboxSummary(m)
	if got.Headline != "Hi" {
		t.Fatalf("headline: got %q", got.Headline)
	}
	if got.Body != "preview text" {
		t.Fatalf("body should fall back to snippet: got %q", got.Body)
	}
}

func TestSubjectFromPart_Nested(t *testing.T) {
	p := &gmailapi.MessagePart{
		MimeType: "multipart/mixed",
		Parts: []*gmailapi.MessagePart{
			{
				MimeType: "text/plain",
				Headers:  []*gmailapi.MessagePartHeader{{Name: "Subject", Value: "Nested"}},
			},
		},
	}
	if s := subjectFromPart(p); s != "Nested" {
		t.Fatalf("got %q", s)
	}
}
