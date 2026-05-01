package ui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// noopSubmit is a placeholder SubmitFunc for tests that don't care about
// the dispatched command.
func noopSubmit(input string) tea.Cmd {
	return func() tea.Msg { return AgentResponseMsg{Content: "ok", Err: nil} }
}

func TestNewApp_StartsIdle(t *testing.T) {
	app := NewApp(DefaultStyles(), "model-x", 4096, noopSubmit)
	if app.State() != StateIdle {
		t.Errorf("expected StateIdle, got %v", app.State())
	}
	if !app.input.Focused() {
		t.Error("expected text input to be focused at startup")
	}
}

func TestApp_WindowSizeUpdatesDimensions(t *testing.T) {
	app := NewApp(DefaultStyles(), "m", 100, noopSubmit)
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	updated := model.(*App)

	if updated.width != 120 {
		t.Errorf("expected width 120, got %d", updated.width)
	}
	if updated.height != 40 {
		t.Errorf("expected height 40, got %d", updated.height)
	}
	if updated.md.Width() != 120 {
		t.Errorf("expected markdown renderer at width 120, got %d", updated.md.Width())
	}
}

func TestApp_EnterOnEmptyInputDoesNothing(t *testing.T) {
	app := NewApp(DefaultStyles(), "m", 100, noopSubmit)
	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(*App)

	if updated.State() != StateIdle {
		t.Errorf("empty submit should stay Idle, got %v", updated.State())
	}
	if cmd != nil {
		t.Errorf("expected no command for empty submit, got %v", cmd)
	}
}

func TestApp_EnterTransitionsToGenerating(t *testing.T) {
	called := false
	submit := func(input string) tea.Cmd {
		called = true
		if input != "hello" {
			t.Errorf("expected input 'hello', got %q", input)
		}
		return func() tea.Msg { return AgentResponseMsg{Content: "hi", Err: nil} }
	}
	app := NewApp(DefaultStyles(), "m", 100, submit)
	app.input.SetValue("hello")

	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(*App)

	if updated.State() != StateGenerating {
		t.Errorf("expected StateGenerating, got %v", updated.State())
	}
	if cmd == nil {
		t.Error("expected a command to be returned (onSubmit + spinner.Init)")
	}
	if updated.input.Value() != "" {
		t.Errorf("expected input cleared, got %q", updated.input.Value())
	}
	if len(updated.History()) != 1 {
		t.Errorf("expected one history entry (the user message), got %d", len(updated.History()))
	}
	if !strings.Contains(stripANSI(updated.History()[0]), "hello") {
		t.Errorf("expected user input echoed in history, got %q", updated.History()[0])
	}

	// Run the returned command — should call submit.
	_ = cmd()
	if !called {
		t.Error("expected onSubmit to be invoked")
	}
}

func TestApp_AgentResponseTransitionsBackToIdle(t *testing.T) {
	app := NewApp(DefaultStyles(), "m", 100, noopSubmit)
	app.state = StateGenerating

	model, _ := app.Update(AgentResponseMsg{Content: "answer", Err: nil})
	updated := model.(*App)

	if updated.State() != StateIdle {
		t.Errorf("expected return to StateIdle, got %v", updated.State())
	}
	if len(updated.History()) != 1 {
		t.Fatalf("expected one history entry, got %d", len(updated.History()))
	}
	if !strings.Contains(stripANSI(updated.History()[0]), "answer") {
		t.Errorf("expected response in history, got %q", updated.History()[0])
	}
}

func TestApp_AgentResponseError(t *testing.T) {
	app := NewApp(DefaultStyles(), "m", 100, noopSubmit)
	app.state = StateGenerating

	model, _ := app.Update(AgentResponseMsg{Err: errors.New("upstream broken")})
	updated := model.(*App)

	if updated.State() != StateIdle {
		t.Errorf("expected return to StateIdle even on error, got %v", updated.State())
	}
	last := stripANSI(updated.History()[len(updated.History())-1])
	if !strings.Contains(last, "Error:") {
		t.Errorf("expected error rendered with prefix, got %q", last)
	}
	if !strings.Contains(last, "upstream broken") {
		t.Errorf("expected error message in history, got %q", last)
	}
}

func TestApp_StreamChunksBuffered(t *testing.T) {
	app := NewApp(DefaultStyles(), "m", 100, noopSubmit)
	app.state = StateGenerating

	app.Update(StreamChunkMsg{Text: "hel"})
	app.Update(StreamChunkMsg{Text: "lo "})
	model, _ := app.Update(StreamChunkMsg{Text: "world"})
	updated := model.(*App)

	if updated.current.String() != "hello world" {
		t.Errorf("expected buffered 'hello world', got %q", updated.current.String())
	}
	// Buffered chunks should NOT yet appear in history — only on flush.
	if len(updated.History()) != 0 {
		t.Errorf("expected no history entries during streaming, got %d", len(updated.History()))
	}
}

func TestApp_StreamFlushesOnAgentResponse(t *testing.T) {
	app := NewApp(DefaultStyles(), "m", 100, noopSubmit)
	app.state = StateGenerating
	app.Update(StreamChunkMsg{Text: "streamed text"})

	// AgentResponseMsg with empty Content — should flush buffer.
	model, _ := app.Update(AgentResponseMsg{})
	updated := model.(*App)

	if updated.current.Len() != 0 {
		t.Errorf("expected current buffer cleared, got %q", updated.current.String())
	}
	if len(updated.History()) != 1 {
		t.Fatalf("expected one history entry from flush, got %d", len(updated.History()))
	}
	if !strings.Contains(stripANSI(updated.History()[0]), "streamed text") {
		t.Errorf("expected flushed text in history, got %q", updated.History()[0])
	}
}

func TestApp_ToolStartAndEndAppendBlocks(t *testing.T) {
	app := NewApp(DefaultStyles(), "m", 100, noopSubmit)
	app.state = StateGenerating

	model, _ := app.Update(ToolStartMsg{Name: "read_file", Args: `{"path":"main.go"}`})
	updated := model.(*App)
	if len(updated.History()) != 1 {
		t.Fatalf("expected one history entry after tool start, got %d", len(updated.History()))
	}
	if !strings.Contains(stripANSI(updated.History()[0]), "[read_file]") {
		t.Errorf("expected tool name in history, got %q", updated.History()[0])
	}

	model, _ = updated.Update(ToolEndMsg{Output: "file content", Duration: "12ms"})
	updated = model.(*App)
	if len(updated.History()) != 2 {
		t.Fatalf("expected two history entries after tool end, got %d", len(updated.History()))
	}
	last := stripANSI(updated.History()[1])
	if !strings.Contains(last, "file content") {
		t.Errorf("expected tool output in history, got %q", last)
	}
	if !strings.Contains(last, "12ms") {
		t.Errorf("expected duration in history, got %q", last)
	}
}

func TestApp_TokenUsageUpdatesCounter(t *testing.T) {
	app := NewApp(DefaultStyles(), "m", 4096, noopSubmit)
	model, _ := app.Update(TokenUsageMsg{Used: 1500})
	updated := model.(*App)

	if updated.tokens != 1500 {
		t.Errorf("expected tokens=1500, got %d", updated.tokens)
	}
	// Status bar should reflect it.
	view := stripANSI(RenderStatusBar(updated.styles, updated.modelName, updated.tokens, updated.maxTokens, 80))
	if !strings.Contains(view, "1500/4096") {
		t.Errorf("expected '1500/4096' in status bar, got %q", view)
	}
}

func TestApp_CtrlCQuits(t *testing.T) {
	app := NewApp(DefaultStyles(), "m", 100, noopSubmit)
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected tea.Quit command, got nil")
	}
	// Run the command — tea.Quit returns tea.QuitMsg.
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestApp_ViewIncludesStatusBar(t *testing.T) {
	app := NewApp(DefaultStyles(), "qwen", 4096, noopSubmit)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app.tokens = 100

	view := stripANSI(app.View())
	if !strings.Contains(view, "qwen") {
		t.Errorf("expected model name in view, got: %q", view)
	}
	if !strings.Contains(view, "100/4096") {
		t.Errorf("expected token counter in view, got: %q", view)
	}
}

func TestApp_ViewSwitchesInputForSpinnerInGenerating(t *testing.T) {
	app := NewApp(DefaultStyles(), "m", 100, noopSubmit)
	app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Idle: text input visible (placeholder)
	idleView := stripANSI(app.View())
	if !strings.Contains(idleView, "Ask anything") {
		t.Errorf("expected placeholder in idle view, got: %q", idleView)
	}

	// Generating: spinner visible (verb), no placeholder
	app.state = StateGenerating
	app.spinner.SetVerb("Pondering...")
	genView := stripANSI(app.View())
	if !strings.Contains(genView, "Pondering...") {
		t.Errorf("expected spinner verb in generating view, got: %q", genView)
	}
}
