package ui

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderUserInput_HasPromptPrefix(t *testing.T) {
	styles := DefaultStyles()
	out := stripANSI(RenderUserInput(styles, "hello world"))
	if !strings.HasPrefix(out, "> ") {
		t.Errorf("expected '> ' prefix, got: %q", out)
	}
	if !strings.Contains(out, "hello world") {
		t.Errorf("expected input echoed, got: %q", out)
	}
}

func TestRenderAssistantResponse_FallsBackOnNilRenderer(t *testing.T) {
	styles := DefaultStyles()
	out := RenderAssistantResponse(styles, nil, "plain content")
	if !strings.Contains(stripANSI(out), "plain content") {
		t.Errorf("expected fallback to render content, got: %q", out)
	}
}

func TestRenderAssistantResponse_UsesGlamourWhenAvailable(t *testing.T) {
	styles := DefaultStyles()
	md, _ := NewMarkdownRenderer(80)
	out := RenderAssistantResponse(styles, md, "**bold**")
	stripped := stripANSI(out)
	if !strings.Contains(stripped, "bold") {
		t.Errorf("expected 'bold' word in rendered output, got: %q", stripped)
	}
}

func TestRenderToolCall_TruncatesLongArgs(t *testing.T) {
	styles := DefaultStyles()
	longArgs := strings.Repeat("a", ToolArgsMaxRunes+50)
	out := stripANSI(RenderToolCall(styles, "read_file", longArgs))

	if !strings.Contains(out, "[read_file]") {
		t.Errorf("expected tool name header, got: %q", out)
	}
	if !strings.HasSuffix(strings.TrimSpace(out), EllipsisRunes) {
		// Border characters may appear after the args, so check for
		// the ellipsis substring instead of a strict suffix match.
		if !strings.Contains(out, EllipsisRunes) {
			t.Errorf("expected truncation ellipsis, got: %q", out)
		}
	}
}

func TestRenderToolCall_ShortArgsNotTruncated(t *testing.T) {
	styles := DefaultStyles()
	out := stripANSI(RenderToolCall(styles, "read_file", `{"path":"a.go"}`))
	if !strings.Contains(out, `{"path":"a.go"}`) {
		t.Errorf("short args should pass through unchanged, got: %q", out)
	}
	if strings.Contains(out, EllipsisRunes) {
		t.Errorf("short args should not be truncated, got: %q", out)
	}
}

func TestRenderToolResult_ElidesAfterMaxLines(t *testing.T) {
	styles := DefaultStyles()
	var lines []string
	for i := 0; i < ToolResultMaxLines+5; i++ {
		lines = append(lines, "line")
	}
	out := stripANSI(RenderToolResult(styles, strings.Join(lines, "\n"), "100ms"))

	if !strings.Contains(out, "5 more line(s) elided") {
		t.Errorf("expected elision footer, got: %q", out)
	}
	if !strings.Contains(out, "(100ms)") {
		t.Errorf("expected duration suffix, got: %q", out)
	}
}

func TestRenderToolResult_ShortOutputNoElision(t *testing.T) {
	styles := DefaultStyles()
	out := stripANSI(RenderToolResult(styles, "ok", "5ms"))
	if strings.Contains(out, "elided") {
		t.Errorf("expected no elision for short output, got: %q", out)
	}
	if !strings.Contains(out, "(5ms)") {
		t.Errorf("expected duration, got: %q", out)
	}
}

func TestRenderError_NilReturnsEmpty(t *testing.T) {
	styles := DefaultStyles()
	if got := RenderError(styles, nil); got != "" {
		t.Errorf("nil error should render to empty string, got: %q", got)
	}
}

func TestRenderError_PrefixesWithErrorLabel(t *testing.T) {
	styles := DefaultStyles()
	out := stripANSI(RenderError(styles, errors.New("boom")))
	if !strings.HasPrefix(out, "Error: ") {
		t.Errorf("expected 'Error:' prefix, got: %q", out)
	}
	if !strings.Contains(out, "boom") {
		t.Errorf("expected error message, got: %q", out)
	}
}

func TestRenderStatusBar_FillsExactWidth(t *testing.T) {
	styles := DefaultStyles()
	out := RenderStatusBar(styles, "qwen2.5-coder", 1234, 4096, 80)
	// lipgloss.Width strips ANSI and counts runes, so this is the
	// visible width the user sees.
	w := lipgloss.Width(out)
	// StatusBar style adds Padding(0, 1) which adds 2 columns.
	// The content fills exactly width=80; with horizontal padding it's 82.
	if w != 80 && w != 82 {
		t.Errorf("expected status bar width 80 (or 82 with padding), got %d: %q", w, stripANSI(out))
	}
}

func TestRenderStatusBar_ShowsModelAndTokens(t *testing.T) {
	styles := DefaultStyles()
	out := stripANSI(RenderStatusBar(styles, "llama3", 500, 2000, 100))
	if !strings.Contains(out, "llama3") {
		t.Errorf("expected model name, got: %q", out)
	}
	if !strings.Contains(out, "500/2000") {
		t.Errorf("expected token ratio, got: %q", out)
	}
	if !strings.Contains(out, "25%") {
		t.Errorf("expected percentage, got: %q", out)
	}
}

func TestRenderStatusBar_UnknownMaxOmitsPercent(t *testing.T) {
	styles := DefaultStyles()
	out := stripANSI(RenderStatusBar(styles, "x", 42, 0, 60))
	if !strings.Contains(out, "42 tokens") {
		t.Errorf("expected '42 tokens' fallback when max unknown, got: %q", out)
	}
	if strings.Contains(out, "%") {
		t.Errorf("expected no percent sign with unknown max, got: %q", out)
	}
}

func TestTruncateRunes_HandlesMultibyte(t *testing.T) {
	// Six emoji = 6 runes, ~24 bytes. Truncate to 4 runes → 1 emoji + "..."
	in := "🎉🎈🎁🎂🎃🎆"
	got := truncateRunes(in, 4)
	if utf8.RuneCountInString(got) > 4 {
		t.Errorf("expected ≤4 runes, got %d (%q)", utf8.RuneCountInString(got), got)
	}
	if !strings.HasSuffix(got, EllipsisRunes) {
		t.Errorf("expected ellipsis suffix, got: %q", got)
	}
	// Ensure no broken bytes — every rune valid.
	if !utf8.ValidString(got) {
		t.Errorf("output contains invalid UTF-8: %q", got)
	}
}

func TestTruncateRunes_TinyMaxReturnsEllipsis(t *testing.T) {
	got := truncateRunes("hello world", 2)
	if got != EllipsisRunes {
		t.Errorf("expected just %q for very small max, got: %q", EllipsisRunes, got)
	}
}

func TestTruncateRunes_NoTruncationIfShort(t *testing.T) {
	in := "short"
	got := truncateRunes(in, 100)
	if got != in {
		t.Errorf("short input should pass through, got: %q", got)
	}
}
