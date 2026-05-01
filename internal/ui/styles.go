// Package ui contains the terminal UI for Fev — themes, styles, views, and
// the bubbletea application model that replaces the bufio.Scanner REPL.
//
// Styles are defined once from a Theme and reused everywhere, so changing
// colors or borders project-wide is a single-struct edit.
package ui

import "github.com/charmbracelet/lipgloss"

// Theme holds the eight colors that drive every visual style.
//
// Colors are ANSI 16-color slot IDs (as strings) rather than hex codes so the
// terminal's existing palette mapping is respected. Users who have customized
// their bright-blue (slot 12) to something else will see Fev pick that up
// automatically, instead of being forced into a hard-coded shade.
type Theme struct {
	Primary   lipgloss.Color // main accent — user input, prompt, status bar bg
	Secondary lipgloss.Color // tool calls, supporting accents
	Success   lipgloss.Color // diff additions, "ok" badges
	Warning   lipgloss.Color // diff headers, cautions
	Error     lipgloss.Color // diff deletions, error text
	Muted     lipgloss.Color // tool results, system messages, spinner verb
	Text      lipgloss.Color // assistant body text
	BG        lipgloss.Color // background — used to invert into the status bar
}

// DarkTheme is the default. ANSI bright variants (slot >= 8) for legibility
// against any dark terminal background.
var DarkTheme = Theme{
	Primary:   lipgloss.Color("12"), // bright blue
	Secondary: lipgloss.Color("14"), // cyan
	Success:   lipgloss.Color("10"), // bright green
	Warning:   lipgloss.Color("11"), // yellow
	Error:     lipgloss.Color("9"),  // bright red
	Muted:     lipgloss.Color("8"),  // bright black / dark gray
	Text:      lipgloss.Color("15"), // bright white
	BG:        lipgloss.Color("0"),  // black
}

// Styles is the resolved set of lipgloss styles for one theme. Each field
// names a specific visual context so call sites read like prose:
//   styles.UserInput.Render("> hello")
type Styles struct {
	UserInput   lipgloss.Style // bold primary — "> user text"
	Assistant   lipgloss.Style // assistant body
	ToolCall    lipgloss.Style // bordered block — "[read_file] main.go"
	ToolResult  lipgloss.Style // muted block under a tool call
	ErrorText   lipgloss.Style // bold error — "Error: ..."
	SystemMsg   lipgloss.Style // italic muted — "/exit", "/clear", etc.
	StatusBar   lipgloss.Style // inverted bar at the bottom
	DiffAdd     lipgloss.Style // green "+ ..." line
	DiffRemove  lipgloss.Style // red "- ..." line
	DiffHeader  lipgloss.Style // bold yellow "@@ ..." line
	Prompt      lipgloss.Style // the literal "> " prefix
	SpinnerText lipgloss.Style // italic muted verb next to the spinner
	Muted       lipgloss.Style // generic muted style for inline use
}

// NewStyles resolves a Theme into ready-to-render Styles. Pure function — no
// global state — so callers can swap themes per session if needed.
func NewStyles(theme Theme) *Styles {
	return &Styles{
		UserInput: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true),
		Assistant: lipgloss.NewStyle().
			Foreground(theme.Text),
		ToolCall: lipgloss.NewStyle().
			Foreground(theme.Secondary).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Secondary).
			Padding(0, 1),
		ToolResult: lipgloss.NewStyle().
			Foreground(theme.Muted),
		ErrorText: lipgloss.NewStyle().
			Foreground(theme.Error).
			Bold(true),
		SystemMsg: lipgloss.NewStyle().
			Foreground(theme.Muted).
			Italic(true),
		StatusBar: lipgloss.NewStyle().
			Background(theme.Primary).
			Foreground(theme.BG).
			Padding(0, 1),
		DiffAdd: lipgloss.NewStyle().
			Foreground(theme.Success),
		DiffRemove: lipgloss.NewStyle().
			Foreground(theme.Error),
		DiffHeader: lipgloss.NewStyle().
			Foreground(theme.Warning).
			Bold(true),
		Prompt: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true),
		SpinnerText: lipgloss.NewStyle().
			Foreground(theme.Muted).
			Italic(true),
		Muted: lipgloss.NewStyle().
			Foreground(theme.Muted),
	}
}

// DefaultStyles returns Styles built from DarkTheme. Used by the bubbletea
// app at startup; callers who want a custom theme should call NewStyles
// directly.
func DefaultStyles() *Styles {
	return NewStyles(DarkTheme)
}
