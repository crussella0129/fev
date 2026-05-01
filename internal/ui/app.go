package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AppState is the bubbletea-app state machine. States exist so Update can
// route key presses correctly: Enter submits in Idle, but is ignored while
// the agent is mid-response.
type AppState int

const (
	StateIdle       AppState = iota // text input focused, waiting for user
	StateGenerating                 // agent is running, spinner shown
)

// AgentResponseMsg is the terminal message of one agent turn. Sent by the
// caller's onSubmit callback once Run returns. Exported so cmd/fev/main.go
// can construct it.
type AgentResponseMsg struct {
	Content string
	Err     error
}

// StreamChunkMsg is one slice of streamed assistant output. Buffered into
// the current-response builder; final assembly happens on AgentResponseMsg.
type StreamChunkMsg struct {
	Text string
}

// ToolStartMsg announces a tool invocation — surface a tool-call block in
// the transcript and switch the spinner verb.
type ToolStartMsg struct {
	Name string
	Args string
}

// ToolEndMsg announces tool completion — surface the tool-result block and
// reset spinner to a thinking verb.
type ToolEndMsg struct {
	Output   string
	Duration string
}

// TokenUsageMsg updates the status bar's token counter mid-conversation.
type TokenUsageMsg struct {
	Used int
}

// SubmitFunc is the bridge into the agent loop. The app calls this when
// the user presses Enter in Idle state. Implementations live in main.go
// and typically wrap agent.Run in a tea.Cmd.
type SubmitFunc func(input string) tea.Cmd

// App is the top-level bubbletea model. Holds every piece of state the
// terminal UI needs across frames.
type App struct {
	input    textinput.Model
	viewport viewport.Model
	spinner  SpinnerModel

	state   AppState
	styles  *Styles
	md      *MarkdownRenderer
	history []string        // each entry is one rendered transcript block
	current strings.Builder // streamed-chunk buffer; flushed on response

	modelName string
	tokens    int
	maxTokens int
	width     int
	height    int

	onSubmit SubmitFunc
}

// NewApp wires the components. modelName and maxTokens populate the status
// bar; onSubmit is the only callback the caller must provide.
func NewApp(styles *Styles, modelName string, maxTokens int, onSubmit SubmitFunc) *App {
	ti := textinput.New()
	ti.Placeholder = "Ask anything... (/exit to quit)"
	ti.Focus()
	ti.CharLimit = 0 // unbounded — we trust the agent layer to handle huge inputs

	vp := viewport.New(80, 20)

	md, _ := NewMarkdownRenderer(80)

	return &App{
		input:     ti,
		viewport:  vp,
		spinner:   NewSpinner(styles),
		state:     StateIdle,
		styles:    styles,
		md:        md,
		modelName: modelName,
		maxTokens: maxTokens,
		width:     80,
		height:    24,
		onSubmit:  onSubmit,
	}
}

// Init kicks off the cursor blink and any other startup commands. Spinner
// only starts ticking once we transition to StateGenerating.
func (m *App) Init() tea.Cmd {
	return textinput.Blink
}

// Update is the event loop. Each message either mutates state, fires a
// command, or both. The fallback at the bottom routes uncaught keys to
// the text input so typing always works.
func (m *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Reserve 2 lines: one for input, one for status bar.
		m.viewport.Width = msg.Width
		m.viewport.Height = max(0, msg.Height-2)
		// Rebuild the markdown renderer at the new width so wrapping
		// matches the visible area.
		if md, err := NewMarkdownRenderer(msg.Width); err == nil {
			m.md = md
		}
		m.refreshViewport()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+d":
			return m, tea.Quit
		case "enter":
			if m.state == StateIdle {
				input := strings.TrimSpace(m.input.Value())
				if input == "" {
					return m, nil
				}
				m.appendHistory(RenderUserInput(m.styles, input))
				m.input.Reset()
				m.state = StateGenerating
				m.current.Reset()
				m.spinner.SetVerb(RandomVerb(ThinkingVerbs))
				cmds = append(cmds, m.onSubmit(input), m.spinner.Init())
				return m, tea.Batch(cmds...)
			}
			// Enter while generating: ignore so the user can't double-submit.
			return m, nil
		}

	case AgentResponseMsg:
		// Final message of a turn. Flush any streamed buffer first; if
		// the response was sent whole (no streaming), use msg.Content.
		var body string
		if m.current.Len() > 0 {
			body = m.current.String()
			m.current.Reset()
		} else {
			body = msg.Content
		}
		if msg.Err != nil {
			m.appendHistory(RenderError(m.styles, msg.Err))
		} else if body != "" {
			m.appendHistory(RenderAssistantResponse(m.styles, m.md, body))
		}
		m.state = StateIdle
		m.input.Focus()
		return m, nil

	case StreamChunkMsg:
		m.current.WriteString(msg.Text)
		// Re-render the in-progress response inline so the user sees
		// progress. Plain (un-glamour'd) text — re-rendering markdown
		// every chunk is too expensive.
		m.refreshViewportWithCurrent()
		return m, nil

	case ToolStartMsg:
		m.appendHistory(RenderToolCall(m.styles, msg.Name, msg.Args))
		m.spinner.SetVerb(VerbForTool(msg.Name))
		return m, nil

	case ToolEndMsg:
		m.appendHistory(RenderToolResult(m.styles, msg.Output, msg.Duration))
		m.spinner.SetVerb(RandomVerb(ThinkingVerbs))
		return m, nil

	case TokenUsageMsg:
		m.tokens = msg.Used
		return m, nil

	case spinner.TickMsg:
		if m.state == StateGenerating {
			var c tea.Cmd
			m.spinner, c = m.spinner.Update(msg)
			return m, c
		}
		return m, nil
	}

	// Default: route to text input so typing works regardless of state.
	var c tea.Cmd
	m.input, c = m.input.Update(msg)
	return m, c
}

// View composes the visible layout: viewport on top, then either spinner
// (generating) or text input (idle), then status bar at the bottom.
func (m *App) View() string {
	var bottom string
	if m.state == StateGenerating {
		bottom = m.spinner.View()
	} else {
		bottom = m.input.View()
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewport.View(),
		bottom,
		RenderStatusBar(m.styles, m.modelName, m.tokens, m.maxTokens, m.width),
	)
}

// State exposes the current state for tests and the caller's status checks.
func (m *App) State() AppState { return m.state }

// History returns a copy of the rendered transcript blocks. Useful for
// tests asserting that messages got appended.
func (m *App) History() []string {
	out := make([]string, len(m.history))
	copy(out, m.history)
	return out
}

// appendHistory adds one rendered block and refreshes the viewport.
// Centralized so every code path keeps the viewport in sync.
func (m *App) appendHistory(block string) {
	m.history = append(m.history, block)
	m.refreshViewport()
}

// refreshViewport joins history into the viewport content and scrolls to
// the bottom — the latest entry is what users care about.
func (m *App) refreshViewport() {
	m.viewport.SetContent(strings.Join(m.history, "\n\n"))
	m.viewport.GotoBottom()
}

// refreshViewportWithCurrent renders the in-progress streamed response
// below the committed history. Plain text — no markdown rendering — for
// throughput.
func (m *App) refreshViewportWithCurrent() {
	body := strings.Join(m.history, "\n\n")
	if m.current.Len() > 0 {
		if body != "" {
			body += "\n\n"
		}
		body += m.styles.Assistant.Render(m.current.String())
	}
	m.viewport.SetContent(body)
	m.viewport.GotoBottom()
}
