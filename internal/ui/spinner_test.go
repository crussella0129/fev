package ui

import (
	"slices"
	"strings"
	"testing"
)

func TestRandomVerb_EmptySliceFallback(t *testing.T) {
	got := RandomVerb(nil)
	if got != "Working..." {
		t.Errorf("RandomVerb(nil) = %q, want %q", got, "Working...")
	}
	got = RandomVerb([]string{})
	if got != "Working..." {
		t.Errorf("RandomVerb(empty) = %q, want %q", got, "Working...")
	}
}

func TestRandomVerb_PicksFromSlice(t *testing.T) {
	verbs := []string{"a", "b", "c"}
	for i := 0; i < 50; i++ {
		got := RandomVerb(verbs)
		if !slices.Contains(verbs, got) {
			t.Errorf("RandomVerb returned %q, not in input", got)
		}
	}
}

func TestVerbForTool_KnownTools(t *testing.T) {
	cases := map[string][]string{
		"read_file":  ReadingVerbs,
		"grep":       SearchingVerbs,
		"glob":       SearchingVerbs,
		"list_dir":   SearchingVerbs,
		"bash":       ExecutingVerbs,
		"git":        ExecutingVerbs,
		"write_file": WritingVerbs,
		"edit_file":  WritingVerbs,
	}
	for tool, pool := range cases {
		got := VerbForTool(tool)
		if !slices.Contains(pool, got) {
			t.Errorf("VerbForTool(%q) = %q, not in expected pool", tool, got)
		}
	}
}

func TestVerbForTool_UnknownFallsBackToThinking(t *testing.T) {
	got := VerbForTool("totally_made_up_tool")
	if !slices.Contains(ThinkingVerbs, got) {
		t.Errorf("VerbForTool(unknown) = %q, not in ThinkingVerbs", got)
	}
}

func TestVerbCategories_NonEmpty(t *testing.T) {
	categories := map[string][]string{
		"ThinkingVerbs":   ThinkingVerbs,
		"ReadingVerbs":    ReadingVerbs,
		"SearchingVerbs":  SearchingVerbs,
		"ExecutingVerbs":  ExecutingVerbs,
		"WritingVerbs":    WritingVerbs,
		"CompactingVerbs": CompactingVerbs,
		"SubagentVerbs":   SubagentVerbs,
	}
	for name, verbs := range categories {
		if len(verbs) == 0 {
			t.Errorf("%s is empty — at least one verb required", name)
		}
		for _, v := range verbs {
			if v == "" {
				t.Errorf("%s contains empty string", name)
			}
			if !strings.HasSuffix(v, "...") {
				t.Errorf("%s entry %q missing trailing ellipsis", name, v)
			}
		}
	}
}

func TestSpinnerModel_SetAndGetVerb(t *testing.T) {
	styles := DefaultStyles()
	s := NewSpinner(styles)

	if s.Verb() == "" {
		t.Fatal("NewSpinner should pick an initial verb")
	}

	s.SetVerb("Custom verb...")
	if s.Verb() != "Custom verb..." {
		t.Errorf("SetVerb didn't take effect: got %q", s.Verb())
	}
}

func TestSpinnerModel_ViewIncludesVerb(t *testing.T) {
	styles := DefaultStyles()
	s := NewSpinner(styles)
	s.SetVerb("Testing the spinner...")

	view := s.View()
	if !strings.Contains(view, "Testing the spinner...") {
		t.Errorf("View missing verb: %q", view)
	}
}
