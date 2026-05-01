package ui

import (
	"strings"
	"testing"
)

func TestRenderDiff_Identical(t *testing.T) {
	styles := DefaultStyles()
	out := RenderDiff(styles, "main.go", "hello\nworld\n", "hello\nworld\n")
	stripped := stripANSI(out)

	if !strings.Contains(stripped, "--- main.go") {
		t.Errorf("expected --- header, got: %q", stripped)
	}
	if !strings.Contains(stripped, "+++ main.go") {
		t.Errorf("expected +++ header, got: %q", stripped)
	}
	// No body lines — i.e. nothing starting with "+ ", "- ", or "  ".
	// (Header lines start with "+++" or "---" so they don't trip these.)
	for _, line := range strings.Split(stripped, "\n") {
		if strings.HasPrefix(line, "+ ") ||
			strings.HasPrefix(line, "- ") ||
			strings.HasPrefix(line, "  ") {
			t.Errorf("expected no body line for identical input, got: %q", line)
		}
	}
}

func TestRenderDiff_PureAddition(t *testing.T) {
	styles := DefaultStyles()
	out := RenderDiff(styles, "main.go", "hello\n", "hello\nworld\n")
	stripped := stripANSI(out)

	if !strings.Contains(stripped, "  hello") {
		t.Errorf("expected unchanged line 'hello', got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "+ world") {
		t.Errorf("expected added line '+ world', got:\n%s", stripped)
	}
}

func TestRenderDiff_PureDeletion(t *testing.T) {
	styles := DefaultStyles()
	out := RenderDiff(styles, "main.go", "hello\nworld\n", "hello\n")
	stripped := stripANSI(out)

	if !strings.Contains(stripped, "  hello") {
		t.Errorf("expected unchanged line 'hello', got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "- world") {
		t.Errorf("expected removed line '- world', got:\n%s", stripped)
	}
}

func TestRenderDiff_Replacement(t *testing.T) {
	styles := DefaultStyles()
	old := "alpha\nbravo\ncharlie\n"
	new := "alpha\nBRAVO\ncharlie\n"
	out := RenderDiff(styles, "main.go", old, new)
	stripped := stripANSI(out)

	if !strings.Contains(stripped, "- bravo") {
		t.Errorf("expected '- bravo', got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "+ BRAVO") {
		t.Errorf("expected '+ BRAVO', got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "  alpha") {
		t.Errorf("expected unchanged '  alpha', got:\n%s", stripped)
	}
	if !strings.Contains(stripped, "  charlie") {
		t.Errorf("expected unchanged '  charlie', got:\n%s", stripped)
	}
}

func TestRenderDiff_LineModeAvoidsCharSplits(t *testing.T) {
	// A rename within one line: line-mode should produce exactly one
	// "- old" and one "+ new", not multiple inline char-level segments.
	styles := DefaultStyles()
	out := RenderDiff(styles, "main.go",
		"foo := getValue()\n",
		"foo := getCachedValue()\n",
	)
	stripped := stripANSI(out)

	dashCount := strings.Count(stripped, "- foo := ")
	plusCount := strings.Count(stripped, "+ foo := ")

	if dashCount != 1 {
		t.Errorf("expected exactly 1 removed line, got %d:\n%s", dashCount, stripped)
	}
	if plusCount != 1 {
		t.Errorf("expected exactly 1 added line, got %d:\n%s", plusCount, stripped)
	}
}

func TestRenderDiff_NoTrailingEmptyLine(t *testing.T) {
	// DiffLinesToChars terminates each line with \n. After CharsToLines,
	// strings.Split on \n produces a trailing "". We strip it — verify
	// no spurious "+ " or "- " or "  " bare-prefix line appears.
	styles := DefaultStyles()
	out := RenderDiff(styles, "f", "a\n", "b\n")
	stripped := stripANSI(out)

	for _, line := range strings.Split(stripped, "\n") {
		// Allow exactly the empty trailing line from the final \n write.
		if line == "+ " || line == "- " || line == "  " {
			t.Errorf("found bare-prefix line in output:\n%s", stripped)
		}
	}
}

// stripANSI removes ANSI escape sequences so tests can assert on plain text
// without depending on lipgloss's exact formatting. Minimal SGR-only impl —
// covers the foreground/background codes lipgloss emits for our styles.
func stripANSI(s string) string {
	var b strings.Builder
	inEscape := false
	for _, r := range s {
		if r == 0x1b { // ESC
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
