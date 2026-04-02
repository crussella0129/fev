# Plan 3: Terminal UI

> **Prerequisite:** Plan 2 (Memory & Compaction) should be complete, but Plan 3 can be started after Plan 1 if you want to work on UI first.
> **Goal:** Replace the basic `bufio.Scanner` REPL with a polished terminal interface using the Charm stack (bubbletea, lipgloss, glamour, bubbles).
> **Estimated effort:** 7 tasks, ~50 TDD steps

---

## What This Plan Builds

Right now, Fev uses `fmt.Print` for output and `bufio.Scanner` for input. It works, but it's ugly. This plan replaces it with:

1. **Bubbletea app shell** — A proper terminal application with structured views
2. **Lipgloss styling** — Colors, borders, and visual hierarchy
3. **Streaming display** — Token-by-token output that feels responsive
4. **Diff rendering** — Colorized unified diffs when files are edited
5. **Spinner system** — Creative loading messages ("Pondering your codebase...")
6. **Markdown rendering** — Terminal-friendly markdown via glamour
7. **Status bar** — Model name, token usage, session duration at the bottom

---

## Go Concepts You'll Need

### The Elm Architecture (bubbletea's pattern)

Bubbletea uses the **Elm Architecture** — a pattern from functional programming. It has three parts:

1. **Model** — A struct that holds ALL of your application state
2. **Update** — A function that receives a message and returns the new model
3. **View** — A function that takes the model and returns a string (what to display)

```go
// Your entire app state
type model struct {
    input    textinput.Model   // the text input box
    messages []string          // chat history
    spinner  spinner.Model     // loading indicator
    width    int               // terminal width
    height   int               // terminal height
}

// Init runs once at startup — return an initial command (or nil)
func (m model) Init() tea.Cmd {
    return textinput.Blink  // start the cursor blinking
}

// Update handles events (key presses, window resize, custom messages)
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "enter" {
            // User pressed enter — process their input
            userInput := m.input.Value()
            m.messages = append(m.messages, "> "+userInput)
            m.input.Reset()
            return m, nil  // or return a Cmd to do async work
        }
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
    }
    return m, nil
}

// View renders the current state as a string
func (m model) View() string {
    // Combine all visual elements into one string
    return lipgloss.JoinVertical(
        lipgloss.Left,
        renderMessages(m.messages),
        m.input.View(),
        renderStatusBar(m),
    )
}
```

### Key concept: tea.Cmd

A `tea.Cmd` is a function that runs asynchronously and sends back a `tea.Msg`:

```go
// This runs the agent in the background and sends back the result
func runAgent(agent *agent.Agent, input string) tea.Cmd {
    return func() tea.Msg {
        result, err := agent.Run(context.Background(), input)
        return agentResponseMsg{result: result, err: err}
    }
}

// You define the message type
type agentResponseMsg struct {
    result string
    err    error
}
```

### Lipgloss (styling)

```go
// Define styles once, reuse everywhere
var (
    userStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("12")).  // blue
        Bold(true)
    
    assistantStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("10"))   // green
    
    errorStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("9")).   // red
        Bold(true)
    
    borderStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        Padding(0, 1)
)

// Use them
fmt.Println(userStyle.Render("> hello"))
fmt.Println(assistantStyle.Render("Hi there!"))
```

---

## New Dependencies

```bash
cd C:/Users/charl/Fev
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/bubbles@latest
go get github.com/charmbracelet/glamour@latest
```

**What each does:**
- **bubbletea** — The application framework (like React for terminals)
- **lipgloss** — CSS-like styling (colors, borders, padding)
- **bubbles** — Pre-built components (text input, spinner, viewport, etc.)
- **glamour** — Markdown-to-terminal renderer (handles headers, code blocks, tables, etc.)

---

## File Structure

```
internal/
  ui/
    app.go            # Bubbletea model — the main application
    app_test.go
    styles.go         # All lipgloss style definitions
    views.go          # Rendering functions for each visual element
    views_test.go
    spinner.go        # Spinner component with creative verbs
    spinner_test.go
    diff.go           # Colorized unified diff rendering
    diff_test.go
    markdown.go       # Glamour markdown rendering wrapper
    markdown_test.go
    verbs.go          # The creative spinner verb collection
```

---

## Task 1: Styles and Theme System

**What you're building:** All the visual styles used throughout the UI. Defined once, used everywhere.

**Files:** `internal/ui/styles.go`

### What to implement

```go
package ui

import "github.com/charmbracelet/lipgloss"

// Theme holds all colors for a visual theme.
type Theme struct {
    Primary   lipgloss.Color // main accent (blue)
    Secondary lipgloss.Color // secondary accent (cyan)
    Success   lipgloss.Color // good things (green)
    Warning   lipgloss.Color // warnings (yellow)
    Error     lipgloss.Color // errors (red)
    Muted     lipgloss.Color // deemphasized text (gray)
    Text      lipgloss.Color // normal text
    BG        lipgloss.Color // background
}

// DarkTheme is the default.
var DarkTheme = Theme{
    Primary:   lipgloss.Color("12"),  // bright blue
    Secondary: lipgloss.Color("14"),  // cyan
    Success:   lipgloss.Color("10"),  // green
    Warning:   lipgloss.Color("11"),  // yellow
    Error:     lipgloss.Color("9"),   // red
    Muted:     lipgloss.Color("8"),   // gray
    Text:      lipgloss.Color("15"),  // white
    BG:        lipgloss.Color("0"),   // black
}

// Styles built from the active theme.
type Styles struct {
    UserInput    lipgloss.Style // "> user text" — bold primary
    Assistant    lipgloss.Style // assistant responses — text color
    ToolCall     lipgloss.Style // "[read_file] main.go" — secondary with border
    ToolResult   lipgloss.Style // tool output — muted
    ErrorText    lipgloss.Style // error messages — red bold
    SystemMsg    lipgloss.Style // system messages — muted italic
    StatusBar    lipgloss.Style // bottom bar — inverted colors
    DiffAdd      lipgloss.Style // added lines — green
    DiffRemove   lipgloss.Style // removed lines — red
    DiffHeader   lipgloss.Style // @@ line —- yellow bold
    Prompt       lipgloss.Style // the ">" prompt character
    SpinnerText  lipgloss.Style // "Pondering..." — muted italic
}

// NewStyles creates styles from a theme.
func NewStyles(theme Theme) *Styles {
    return &Styles{
        UserInput:   lipgloss.NewStyle().Foreground(theme.Primary).Bold(true),
        Assistant:   lipgloss.NewStyle().Foreground(theme.Text),
        ToolCall:    lipgloss.NewStyle().Foreground(theme.Secondary).
            Border(lipgloss.RoundedBorder()).Padding(0, 1),
        ToolResult:  lipgloss.NewStyle().Foreground(theme.Muted),
        ErrorText:   lipgloss.NewStyle().Foreground(theme.Error).Bold(true),
        SystemMsg:   lipgloss.NewStyle().Foreground(theme.Muted).Italic(true),
        StatusBar:   lipgloss.NewStyle().Background(theme.Primary).
            Foreground(theme.BG).Padding(0, 1),
        DiffAdd:     lipgloss.NewStyle().Foreground(theme.Success),
        DiffRemove:  lipgloss.NewStyle().Foreground(theme.Error),
        DiffHeader:  lipgloss.NewStyle().Foreground(theme.Warning).Bold(true),
        Prompt:      lipgloss.NewStyle().Foreground(theme.Primary).Bold(true),
        SpinnerText: lipgloss.NewStyle().Foreground(theme.Muted).Italic(true),
    }
}

// DefaultStyles returns styles using DarkTheme.
func DefaultStyles() *Styles {
    return NewStyles(DarkTheme)
}
```

No formal tests for styles (visual), but verify it compiles.

### Commit
```bash
git commit -m "feat: theme system and lipgloss style definitions"
```

---

## Task 2: Spinner Verbs

**What you're building:** The collection of creative loading messages.

**Files:** `internal/ui/verbs.go`, `internal/ui/spinner.go`, `internal/ui/spinner_test.go`

### Why this matters
UX details like this are the difference between a tool people use once and a tool people love. Claude Code has 187 spinner verbs. Fev should have its own personality.

### What to implement

```go
// verbs.go
package ui

import "math/rand"

// Verb categories — each maps to a tool or action type.
var (
    ThinkingVerbs = []string{
        "Pondering...",
        "Reasoning through this...",
        "Thinking it over...",
        "Considering the options...",
        "Working through it...",
        "Mulling this over...",
        "Connecting the dots...",
        "Following the thread...",
    }
    
    ReadingVerbs = []string{
        "Reading the source...",
        "Scanning the codebase...",
        "Rummaging through files...",
        "Studying the code...",
        "Examining the contents...",
        "Poring over the source...",
    }
    
    SearchingVerbs = []string{
        "Hunting for matches...",
        "Sifting through the project...",
        "Tracing references...",
        "Searching far and wide...",
        "Following the trail...",
        "Narrowing it down...",
    }
    
    ExecutingVerbs = []string{
        "Running the command...",
        "Executing...",
        "Waiting on the shell...",
        "Processing...",
        "Working on it...",
    }
    
    WritingVerbs = []string{
        "Crafting the code...",
        "Writing changes...",
        "Editing the source...",
        "Applying modifications...",
    }
    
    CompactingVerbs = []string{
        "Tidying up context...",
        "Consolidating memory...",
        "Making room to think...",
        "Organizing thoughts...",
    }
    
    SubagentVerbs = []string{
        "Dispatching helpers...",
        "Agents at work...",
        "Gathering intel...",
        "Coordinating...",
    }
)

// RandomVerb picks a random verb from a category.
func RandomVerb(verbs []string) string {
    if len(verbs) == 0 {
        return "Working..."
    }
    return verbs[rand.Intn(len(verbs))]
}

// VerbForTool returns the appropriate verb category for a tool name.
func VerbForTool(toolName string) string {
    switch toolName {
    case "read_file":
        return RandomVerb(ReadingVerbs)
    case "grep", "glob", "list_dir":
        return RandomVerb(SearchingVerbs)
    case "bash":
        return RandomVerb(ExecutingVerbs)
    case "write_file", "edit_file":
        return RandomVerb(WritingVerbs)
    case "git":
        return RandomVerb(ExecutingVerbs)
    default:
        return RandomVerb(ThinkingVerbs)
    }
}
```

```go
// spinner.go — wraps bubbles spinner with verb display
package ui

import (
    "github.com/charmbracelet/bubbles/spinner"
    tea "github.com/charmbracelet/bubbletea"
)

// SpinnerModel wraps a bubbles spinner with a verb message.
type SpinnerModel struct {
    spinner spinner.Model
    verb    string
    styles  *Styles
}

func NewSpinner(styles *Styles) SpinnerModel {
    s := spinner.New()
    s.Spinner = spinner.Dot  // or spinner.MiniDot, spinner.Pulse, etc.
    return SpinnerModel{
        spinner: s,
        verb:    RandomVerb(ThinkingVerbs),
        styles:  styles,
    }
}

func (m SpinnerModel) Init() tea.Cmd   { return m.spinner.Tick }
func (m SpinnerModel) Update(msg tea.Msg) (SpinnerModel, tea.Cmd) {
    var cmd tea.Cmd
    m.spinner, cmd = m.spinner.Update(msg)
    return m, cmd
}
func (m SpinnerModel) View() string {
    return m.spinner.View() + " " + m.styles.SpinnerText.Render(m.verb)
}

// SetVerb changes the displayed verb.
func (m *SpinnerModel) SetVerb(verb string) { m.verb = verb }
```

### Tests
- VerbForTool("read_file") returns a string from ReadingVerbs
- VerbForTool("unknown") returns from ThinkingVerbs
- RandomVerb with empty slice returns "Working..."

### Commit
```bash
git commit -m "feat: creative spinner verbs and spinner component"
```

---

## Task 3: Diff Rendering

**What you're building:** Colorized display of file changes (like `git diff` but prettier).

**Files:** `internal/ui/diff.go`, `internal/ui/diff_test.go`

### What to implement

```go
package ui

import (
    "strings"
    "github.com/sergi/go-diff/diffmatchpatch"
)

// RenderDiff produces a colorized unified diff between old and new content.
func RenderDiff(styles *Styles, filename, oldContent, newContent string) string {
    dmp := diffmatchpatch.New()
    diffs := dmp.DiffMain(oldContent, newContent, true)
    
    var b strings.Builder
    b.WriteString(styles.DiffHeader.Render("--- " + filename) + "\n")
    b.WriteString(styles.DiffHeader.Render("+++ " + filename) + "\n")
    
    for _, diff := range diffs {
        lines := strings.Split(diff.Text, "\n")
        for _, line := range lines {
            if line == "" { continue }
            switch diff.Type {
            case diffmatchpatch.DiffInsert:
                b.WriteString(styles.DiffAdd.Render("+ " + line) + "\n")
            case diffmatchpatch.DiffDelete:
                b.WriteString(styles.DiffRemove.Render("- " + line) + "\n")
            case diffmatchpatch.DiffEqual:
                b.WriteString("  " + line + "\n")
            }
        }
    }
    return b.String()
}
```

### Tests
- Diff with additions shows green lines
- Diff with deletions shows red lines
- Diff with no changes returns minimal output

### Commit
```bash
git commit -m "feat: colorized diff rendering"
```

---

## Task 4: Markdown Rendering

**Files:** `internal/ui/markdown.go`, `internal/ui/markdown_test.go`

### What to implement

```go
package ui

import (
    "github.com/charmbracelet/glamour"
)

// MarkdownRenderer wraps glamour for terminal markdown.
type MarkdownRenderer struct {
    renderer *glamour.TermRenderer
}

func NewMarkdownRenderer(width int) (*MarkdownRenderer, error) {
    r, err := glamour.NewTermRenderer(
        glamour.WithAutoStyle(),        // detect dark/light terminal
        glamour.WithWordWrap(width),    // wrap to terminal width
    )
    if err != nil {
        return nil, err
    }
    return &MarkdownRenderer{renderer: r}, nil
}

// Render converts markdown to styled terminal output.
func (m *MarkdownRenderer) Render(markdown string) (string, error) {
    return m.renderer.Render(markdown)
}
```

### Tests
- Render a heading → contains styled output
- Render code block → contains the code
- Render with empty string → no error

### Commit
```bash
git commit -m "feat: terminal markdown rendering via glamour"
```

---

## Task 5: Views (Rendering Functions)

**Files:** `internal/ui/views.go`, `internal/ui/views_test.go`

### What to implement

Individual render functions for each UI element:

```go
package ui

import (
    "fmt"
    "strings"
    "github.com/charmbracelet/lipgloss"
)

// RenderUserInput formats user input for display.
func RenderUserInput(styles *Styles, input string) string {
    return styles.Prompt.Render("> ") + styles.UserInput.Render(input)
}

// RenderAssistantResponse formats assistant text.
func RenderAssistantResponse(styles *Styles, md *MarkdownRenderer, content string) string {
    rendered, err := md.Render(content)
    if err != nil {
        return styles.Assistant.Render(content) // fallback to plain
    }
    return rendered
}

// RenderToolCall formats a tool invocation.
func RenderToolCall(styles *Styles, toolName, args string) string {
    header := fmt.Sprintf("[%s]", toolName)
    // Truncate args for display
    if len(args) > 120 {
        args = args[:120] + "..."
    }
    return styles.ToolCall.Render(header + " " + args)
}

// RenderToolResult formats tool output.
func RenderToolResult(styles *Styles, output string, duration string) string {
    footer := styles.Muted.Render(fmt.Sprintf("(%s)", duration))
    // Limit preview to 10 lines
    lines := strings.Split(output, "\n")
    if len(lines) > 10 {
        output = strings.Join(lines[:10], "\n") + "\n..."
    }
    return styles.ToolResult.Render(output) + " " + footer
}

// RenderError formats an error message.
func RenderError(styles *Styles, err error) string {
    return styles.ErrorText.Render("Error: " + err.Error())
}

// RenderStatusBar formats the bottom status bar.
func RenderStatusBar(styles *Styles, modelName string, tokenUsage int, 
    maxTokens int, width int) string {
    left := fmt.Sprintf(" %s ", modelName)
    right := fmt.Sprintf(" %d/%d tokens ", tokenUsage, maxTokens)
    gap := width - lipgloss.Width(left) - lipgloss.Width(right)
    if gap < 0 { gap = 0 }
    bar := left + strings.Repeat(" ", gap) + right
    return styles.StatusBar.Render(bar)
}
```

### Tests
- RenderUserInput contains ">" prefix
- RenderToolCall truncates long args
- RenderStatusBar shows model name and tokens
- RenderError contains error message

### Commit
```bash
git commit -m "feat: UI view rendering functions"
```

---

## Task 6: Bubbletea Application Model

**What you're building:** The main application — this replaces the `bufio.Scanner` REPL entirely.

**Files:** `internal/ui/app.go`, `internal/ui/app_test.go`

### The big picture

The app has states:
1. **Idle** — waiting for user input (text input focused)
2. **Generating** — LLM is thinking (spinner visible, input disabled)
3. **Confirming** — asking user to approve a dangerous tool (yes/no prompt)

### What to implement

```go
package ui

import (
    "github.com/charmbracelet/bubbles/textinput"
    "github.com/charmbracelet/bubbles/viewport"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type AppState int
const (
    StateIdle       AppState = iota
    StateGenerating
    StateConfirming
)

// Custom message types
type agentResponseMsg struct {
    content string
    err     error
}
type streamChunkMsg struct { text string }
type toolCallMsg struct { name, args string }
type toolResultMsg struct { output, duration string }

// App is the bubbletea application model.
type App struct {
    // Components
    input    textinput.Model
    viewport viewport.Model
    spinner  SpinnerModel
    
    // State
    state    AppState
    styles   *Styles
    md       *MarkdownRenderer
    history  []string  // rendered lines for the viewport
    
    // Config
    modelName string
    tokens    int
    maxTokens int
    width     int
    height    int
    
    // Agent callback (set by the caller)
    onSubmit func(string) tea.Cmd  // called when user presses enter
}

func NewApp(styles *Styles, modelName string, maxTokens int, 
    onSubmit func(string) tea.Cmd) *App {
    ti := textinput.New()
    ti.Placeholder = "Ask anything..."
    ti.Focus()
    
    md, _ := NewMarkdownRenderer(80)
    
    return &App{
        input:     ti,
        spinner:   NewSpinner(styles),
        styles:    styles,
        md:        md,
        modelName: modelName,
        maxTokens: maxTokens,
        state:     StateIdle,
        onSubmit:  onSubmit,
    }
}

// Init, Update, View implement the tea.Model interface.
// (Implement the full Elm Architecture loop here)
```

This is the most complex task. The `Update` function needs to handle:
- `tea.KeyMsg` — Enter (submit), Ctrl+C (quit), slash commands
- `tea.WindowSizeMsg` — resize viewport
- `agentResponseMsg` — display result, return to idle
- `streamChunkMsg` — append to current response
- `toolCallMsg` — show tool call block
- `toolResultMsg` — show result block
- `spinner.TickMsg` — animate the spinner

### Tests
- NewApp creates valid model
- Input focus is on text input in idle state
- State transitions: idle → generating → idle

### Commit
```bash
git commit -m "feat: bubbletea application model with states"
```

---

## Task 7: Wire UI into main.go

**What you're building:** Replace the old REPL with the new bubbletea app.

**Files:** Modify `cmd/fev/main.go`

### What changes

The `runInteractive` function currently uses `bufio.Scanner`. Replace it with:

```go
func runInteractive(cmd *cobra.Command, args []string) error {
    // ... same setup as before (config, workspace, client, registry, ctxMgr, agent) ...
    
    styles := ui.DefaultStyles()
    
    onSubmit := func(input string) tea.Cmd {
        return func() tea.Msg {
            result, err := agent.Run(context.Background(), input)
            return ui.AgentResponseMsg{Content: result, Err: err}
        }
    }
    
    app := ui.NewApp(styles, cfg.Model.ModelName, ctxLen, onSubmit)
    
    p := tea.NewProgram(app, tea.WithAltScreen())
    _, err := p.Run()
    return err
}
```

### Tests
- Build test: `go build ./cmd/fev`
- Manual test: run `./bin/fev`, type something, see styled output

### Commit
```bash
git commit -m "feat: replace REPL with bubbletea UI"
git tag v0.3.0
```

---

## Summary of What You'll Have After Plan 3

A terminal application that:
- Shows styled, color-coded output (user = blue, assistant = white, errors = red)
- Renders markdown (headers, code blocks, tables) in the terminal
- Shows colorized diffs when files are edited
- Displays creative spinner messages during LLM generation and tool execution
- Has a status bar showing model name and token usage
- Resizes gracefully when you change terminal size
- Handles Ctrl+C cleanly
