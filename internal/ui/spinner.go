package ui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// SpinnerModel wraps a bubbles spinner with a verb message that scrolls
// alongside it. Embeds the bubbles model rather than aliasing so callers can
// always reach the underlying spinner if they need to swap shapes.
type SpinnerModel struct {
	spinner spinner.Model
	verb    string
	styles  *Styles
}

// NewSpinner builds a spinner using the Dot shape and a random thinking
// verb. The styles pointer determines how the verb is rendered (italic
// muted by default).
func NewSpinner(styles *Styles) SpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return SpinnerModel{
		spinner: s,
		verb:    RandomVerb(ThinkingVerbs),
		styles:  styles,
	}
}

// Init returns the initial Tick command — bubbletea calls this once at
// startup to begin the animation loop.
func (m SpinnerModel) Init() tea.Cmd { return m.spinner.Tick }

// Update advances the spinner frame on every spinner.TickMsg. Returns the
// receiver by value (bubbles' convention) so the parent model swaps the new
// state in via reassignment.
func (m SpinnerModel) Update(msg tea.Msg) (SpinnerModel, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

// View renders the spinner glyph followed by the styled verb text:
//   ⠋ Pondering...
func (m SpinnerModel) View() string {
	return m.spinner.View() + " " + m.styles.SpinnerText.Render(m.verb)
}

// SetVerb swaps the displayed verb. Pointer receiver because callers want
// to mutate the spinner in place when a tool call begins
// (e.g. "Reading the source..." → "Hunting for matches...").
func (m *SpinnerModel) SetVerb(verb string) { m.verb = verb }

// Verb returns the current verb — useful for tests asserting that
// SetVerb actually took effect.
func (m SpinnerModel) Verb() string { return m.verb }
