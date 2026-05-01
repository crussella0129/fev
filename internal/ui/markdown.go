package ui

import (
	"github.com/charmbracelet/glamour"
)

// MarkdownRenderer turns markdown source into styled terminal output. Wraps
// glamour.TermRenderer with a fixed word-wrap width set at construction.
//
// On terminal resize the bubbletea app should call NewMarkdownRenderer
// again with the new width — glamour does not observe size changes
// internally.
type MarkdownRenderer struct {
	renderer *glamour.TermRenderer
	width    int
}

// NewMarkdownRenderer builds a renderer wrapped to the given width.
// width <= 0 falls back to 80 — a sensible default for narrow panes and
// safer than asking glamour to handle a non-positive wrap.
func NewMarkdownRenderer(width int) (*MarkdownRenderer, error) {
	if width <= 0 {
		width = 80
	}
	r, err := glamour.NewTermRenderer(
		// Auto-detect dark vs light based on COLORFGBG / terminal hints.
		// Falls back to dark if the env doesn't say.
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil, err
	}
	return &MarkdownRenderer{renderer: r, width: width}, nil
}

// Render converts markdown source into ANSI-styled terminal text.
// Empty input returns empty output without invoking glamour.
func (m *MarkdownRenderer) Render(markdown string) (string, error) {
	if markdown == "" {
		return "", nil
	}
	return m.renderer.Render(markdown)
}

// Width returns the wrap width the renderer was built with — useful for
// tests and for the app model deciding whether to rebuild on resize.
func (m *MarkdownRenderer) Width() int { return m.width }
