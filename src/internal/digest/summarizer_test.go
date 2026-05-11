package digest

import "testing"

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

func TestParseDigestOutput_RejectsFences(t *testing.T) {
	_, err := parseDigestOutput("```json\n{}\n```")
	if err == nil {
		t.Fatal("expected error")
	}
}
