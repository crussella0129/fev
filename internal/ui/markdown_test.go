package ui

import (
	"strings"
	"testing"
)

func TestNewMarkdownRenderer_DefaultsWidth(t *testing.T) {
	m, err := NewMarkdownRenderer(0)
	if err != nil {
		t.Fatalf("NewMarkdownRenderer(0): %v", err)
	}
	if m.Width() != 80 {
		t.Errorf("expected default width 80, got %d", m.Width())
	}

	m, err = NewMarkdownRenderer(-5)
	if err != nil {
		t.Fatalf("NewMarkdownRenderer(-5): %v", err)
	}
	if m.Width() != 80 {
		t.Errorf("expected default width 80 for negative input, got %d", m.Width())
	}
}

func TestNewMarkdownRenderer_RespectsExplicitWidth(t *testing.T) {
	m, err := NewMarkdownRenderer(120)
	if err != nil {
		t.Fatalf("NewMarkdownRenderer: %v", err)
	}
	if m.Width() != 120 {
		t.Errorf("expected width 120, got %d", m.Width())
	}
}

func TestRender_EmptyInputReturnsEmpty(t *testing.T) {
	m, _ := NewMarkdownRenderer(80)
	out, err := m.Render("")
	if err != nil {
		t.Fatalf("Render(\"\"): %v", err)
	}
	if out != "" {
		t.Errorf("expected empty output for empty input, got %q", out)
	}
}

func TestRender_HeadingProducesOutput(t *testing.T) {
	m, _ := NewMarkdownRenderer(80)
	out, err := m.Render("# Hello World\n")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// Glamour formats headings with extra whitespace and styling — we
	// don't pin the exact bytes, just confirm the heading text survived.
	if !strings.Contains(out, "Hello World") {
		t.Errorf("expected 'Hello World' in rendered output, got: %q", out)
	}
}

func TestRender_CodeBlockPreservesContent(t *testing.T) {
	m, _ := NewMarkdownRenderer(80)
	out, err := m.Render("```go\nfmt.Println(\"hi\")\n```\n")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "fmt.Println") {
		t.Errorf("code block content missing from output: %q", out)
	}
}

func TestRender_ListItems(t *testing.T) {
	m, _ := NewMarkdownRenderer(80)
	out, err := m.Render("- one\n- two\n- three\n")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	for _, want := range []string{"one", "two", "three"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in rendered list, got: %q", want, out)
		}
	}
}
