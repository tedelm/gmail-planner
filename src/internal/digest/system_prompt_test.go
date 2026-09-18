package digest

import (
	"strings"
	"testing"

	"github.com/tedelm/gmail-planner/internal/config"
)

func TestBuildFamilyDigestSystemPrompt_IncludesChildrenAndHomework(t *testing.T) {
	children := []config.DigestChild{
		{Name: "Annie", Labels: []string{"0---kidsen-annie", "kidsen-annie"}, ClassCodes: []string{"F2017", "F17", "17"}},
		{Name: "Oscar", Labels: []string{"0---kidsen-oscar", "kidsen-oscar"}, ClassCodes: []string{"P2014", "14e"}},
	}
	got := buildFamilyDigestSystemPrompt(1, "sv-SE", "2026-09-18", children)
	for _, want := range []string{
		"Annie",
		"Oscar",
		"F2017",
		"P2014",
		"kidsen-annie",
		"kidsen-oscar",
		"CHILD MAP",
		"HOMEWORK",
		"Veckans ord",
		"<h4>",
		"Båda / oklart",
		"2026-09-18",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in system prompt", want)
		}
	}
}

func TestBuildFamilyDigestSystemPrompt_NoChildrenOmitsChildMap(t *testing.T) {
	got := buildFamilyDigestSystemPrompt(1, "sv-SE", "2026-09-18", nil)
	if strings.Contains(got, "CHILD MAP") {
		t.Fatal("expected no CHILD MAP without children")
	}
	if !strings.Contains(got, "HOMEWORK") {
		t.Fatal("expected HOMEWORK instructions always")
	}
	if strings.Contains(got, "Båda / oklart") {
		t.Fatal("Båda / oklart only when children configured")
	}
}
