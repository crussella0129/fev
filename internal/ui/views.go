package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// Truncation limits for the rendered views. Tweak here rather than at call
// sites so the look stays consistent across the app.
const (
	// ToolArgsMaxRunes caps the argument string shown next to a tool name.
	// Long args (e.g. a 5KB write_file payload) blow past terminal width
	// and bury the conversation. 120 runes ≈ one wrapped line on most
	// terminals; the full args still go to the tool itself.
	ToolArgsMaxRunes = 120

	// ToolResultMaxLines caps the inline preview of a tool's output. The
	// rest is still saved to the session's tool-results/ folder, so users
	// can grep the file if they need the full content.
	ToolResultMaxLines = 10

	// EllipsisRunes is appended after a truncation. Three dots — kept short
	// so the truncated string remains readable.
	EllipsisRunes = "..."
)

// RenderUserInput formats what the user typed for echo back into the
// transcript. The literal "> " prefix uses Prompt style; the input itself
// uses UserInput style. We split them so the prompt stays visually
// anchored even if the input wraps.
func RenderUserInput(styles *Styles, input string) string {
	return styles.Prompt.Render("> ") + styles.UserInput.Render(input)
}

// RenderAssistantResponse formats the assistant's text body. Falls through
// to plain Assistant styling if glamour fails (rare — usually only when the
// terminal is in some pathological state). Never returns an error: the
// transcript should keep flowing even if formatting hiccups.
func RenderAssistantResponse(styles *Styles, md *MarkdownRenderer, content string) string {
	if md == nil {
		return styles.Assistant.Render(content)
	}
	rendered, err := md.Render(content)
	if err != nil || rendered == "" {
		return styles.Assistant.Render(content)
	}
	return rendered
}

// RenderToolCall formats a tool invocation block:
//
//   ╭───────────────────────────╮
//   │ [read_file] {"path":"..."}│
//   ╰───────────────────────────╯
//
// args is truncated by rune count (not byte count) so multi-byte UTF-8
// never breaks mid-codepoint.
func RenderToolCall(styles *Styles, toolName, args string) string {
	header := fmt.Sprintf("[%s]", toolName)
	display := truncateRunes(args, ToolArgsMaxRunes)
	return styles.ToolCall.Render(header + " " + display)
}

// RenderToolResult formats a tool's output for inline display. Caps at
// ToolResultMaxLines and appends a "(... N lines elided)" footer when the
// tail was dropped, so users know more exists. duration is appended at the
// end as muted text — e.g. "(123ms)".
func RenderToolResult(styles *Styles, output, duration string) string {
	lines := strings.Split(output, "\n")
	elided := 0
	if len(lines) > ToolResultMaxLines {
		elided = len(lines) - ToolResultMaxLines
		lines = lines[:ToolResultMaxLines]
	}
	preview := strings.Join(lines, "\n")

	body := styles.ToolResult.Render(preview)
	tail := ""
	if elided > 0 {
		tail = styles.Muted.Render(
			fmt.Sprintf("\n... %d more line(s) elided", elided))
	}
	if duration != "" {
		tail += " " + styles.Muted.Render(fmt.Sprintf("(%s)", duration))
	}
	return body + tail
}

// RenderError formats an error message — bold red, "Error: ..." prefix.
// Nil errors render as empty so callers don't need to nil-check before
// calling.
func RenderError(styles *Styles, err error) string {
	if err == nil {
		return ""
	}
	return styles.ErrorText.Render("Error: " + err.Error())
}

// RenderSystemMessage formats internal app messages (e.g. slash command
// echoes, "session saved" notices). Italic muted so they recede next to
// the assistant's main reply.
func RenderSystemMessage(styles *Styles, msg string) string {
	return styles.SystemMsg.Render(msg)
}

// RenderStatusBar formats the bottom bar:
//
//   [model name]                    1234/4096 tokens (30%)
//
// Pads the middle with spaces to span exactly width columns. lipgloss.Width
// handles ANSI escapes correctly — using len() would over-count and leave
// the bar short.
func RenderStatusBar(styles *Styles, modelName string, tokenUsage, maxTokens, width int) string {
	left := fmt.Sprintf(" %s ", modelName)
	right := formatTokenStatus(tokenUsage, maxTokens)
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	bar := left + strings.Repeat(" ", gap) + right
	return styles.StatusBar.Render(bar)
}

// formatTokenStatus produces "1234/4096 (30%)" when maxTokens > 0,
// "1234 tokens" otherwise (max unknown). Trailing space ensures it doesn't
// hug the right edge of the terminal.
func formatTokenStatus(used, max int) string {
	if max <= 0 {
		return fmt.Sprintf(" %d tokens ", used)
	}
	pct := 0
	if used > 0 {
		pct = (used * 100) / max
	}
	return fmt.Sprintf(" %d/%d (%d%%) ", used, max, pct)
}

// truncateRunes cuts a string to maxRunes-EllipsisRunes characters and
// appends "...". Pure rune-aware truncation — never lands mid-codepoint.
func truncateRunes(s string, maxRunes int) string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	keep := maxRunes - utf8.RuneCountInString(EllipsisRunes)
	if keep <= 0 {
		return EllipsisRunes
	}
	count := 0
	for i := range s {
		if count == keep {
			return s[:i] + EllipsisRunes
		}
		count++
	}
	return s + EllipsisRunes
}
