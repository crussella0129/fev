// cmd/fev/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/crussella0129/fev/internal/agent"
	"github.com/crussella0129/fev/internal/config"
	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/ctxwin"
	"github.com/crussella0129/fev/internal/llm"
	"github.com/crussella0129/fev/internal/memory"
	"github.com/crussella0129/fev/internal/tools"
	"github.com/crussella0129/fev/internal/ui"
	"github.com/spf13/cobra"
)

var version = "0.3.0"

var (
	cfgPath       string
	workspaceRoot string
	verbose       bool
)

var rootCmd = &cobra.Command{
	Use:   "fev [task]",
	Short: "Fev — local-first agentic CLI",
	Long:  "Fev is a personal assistant for navigating the digital space. Model-agnostic, single binary, local-first.",
	RunE:  runInteractive,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print Fev version",
	Run:   func(cmd *cobra.Command, args []string) { fmt.Printf("fev v%s\n", version) },
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create FEV.md template in current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		template := `# FEV.md

## Project
Describe your project here.

## Build & Test
- Build: ` + "`command`" + `
- Test: ` + "`command`" + `

## Conventions
- List conventions here

## Do Not
- List restrictions here
`
		if _, err := os.Stat("FEV.md"); err == nil {
			return fmt.Errorf("FEV.md already exists")
		}
		return os.WriteFile("FEV.md", []byte(template), 0644)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "", "Config file path")
	rootCmd.PersistentFlags().StringVar(&workspaceRoot, "workspace", ".", "Workspace root directory")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.AddCommand(versionCmd, initCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runInteractive(cmd *cobra.Command, args []string) error {
	// Load config
	configPath := cfgPath
	if configPath == "" {
		dir, err := config.ConfigDir()
		if err != nil {
			return err
		}
		configPath = dir + "/config.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Setup workspace
	ws, err := core.NewWorkspace(workspaceRoot)
	if err != nil {
		return fmt.Errorf("workspace: %w", err)
	}

	// Create LLM client
	client := llm.NewClient(cfg.Model.BaseURL, cfg.Model.ModelName)

	// Create tool registry
	reg := tools.NewRegistry()
	reg.Register(tools.NewReadFileTool(ws))
	reg.Register(tools.NewWriteFileTool(ws))
	reg.Register(tools.NewEditFileTool(ws))
	reg.Register(tools.NewBashTool(ws))
	reg.Register(tools.NewGrepTool(ws))
	reg.Register(tools.NewGlobTool(ws))
	reg.Register(tools.NewListDirTool(ws))
	reg.Register(tools.NewGitTool(ws))

	// Context manager
	ctxLen := cfg.Model.ContextLength
	if ctxLen <= 0 {
		ctxLen = 8192
	}
	ctxMgr := ctxwin.NewManager(ctxLen, 2048)

	// Create agent
	a := agent.New(client, reg, ctxMgr, cfg)

	// Initialize persistent memory
	fevDir := fevHomeDir()
	store, session := initMemory(fevDir)
	if store != nil {
		a.SetMemory(store, session)
		defer store.Close()
	}

	// Load project config and latest review into system prompt
	projectCfg, _ := config.LoadProjectConfig(ws.Root())
	latestReview, _ := memory.LoadLatestReview(filepath.Join(fevDir, "sessions"))
	systemPrompt := buildSystemPrompt(projectCfg, reg, latestReview)
	a.SetSystemPrompt(systemPrompt)

	// Banner
	fmt.Printf("fev v%s — model: %s @ %s\n", version, cfg.Model.ModelName, cfg.Model.BaseURL)
	fmt.Printf("workspace: %s\n", ws.Root())
	fmt.Println("Type /help for commands, /exit to quit.")
	fmt.Println()

	// If task provided as args, run it (no REPL, no review)
	if len(args) > 0 {
		task := strings.Join(args, " ")
		return runTask(a, task)
	}

	// Interactive UI
	return runUI(a, client, store, session, cfg.Model.ModelName, ctxLen)
}

// fevHomeDir returns ~/.fev, creating it if necessary.
func fevHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	dir := filepath.Join(home, ".fev")
	_ = os.MkdirAll(filepath.Join(dir, "sessions"), 0755)
	return dir
}

// initMemory opens the persistent store and creates a new session.
// Returns nil, nil on failure so the caller can proceed without memory.
func initMemory(fevDir string) (*memory.Store, *memory.Session) {
	store, err := memory.NewStore(filepath.Join(fevDir, "memory.db"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: memory unavailable: %v\n", err)
		return nil, nil
	}
	session, err := memory.NewSession(filepath.Join(fevDir, "sessions"))
	if err != nil {
		store.Close()
		fmt.Fprintf(os.Stderr, "warning: session unavailable: %v\n", err)
		return nil, nil
	}
	return store, session
}

// runUI launches the bubbletea-based terminal UI. Slash commands are
// intercepted in submit() so they never reach the agent loop. /exit sets
// quitting=true and returns tea.Quit; the deferred review runs after the
// program exits, keeping its output in the main terminal scrollback.
func runUI(a *agent.Agent, client memory.ReviewClient, store *memory.Store, session *memory.Session, modelName string, maxTokens int) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var quitting bool

	submit := func(input string) tea.Cmd {
		// Slash commands handled here so the App stays generic — no agent
		// or memory deps inside internal/ui.
		switch {
		case input == "/exit" || input == "/quit":
			quitting = true
			return tea.Quit

		case input == "/help":
			return func() tea.Msg {
				return ui.AgentResponseMsg{Content: helpText()}
			}

		case input == "/reset":
			a.Reset()
			return func() tea.Msg {
				return ui.AgentResponseMsg{Content: "_Conversation reset._"}
			}
		}

		// Normal turn: run agent.Run synchronously inside the cmd.
		// Bubbletea executes cmds in their own goroutines, so the UI
		// keeps animating the spinner while we wait.
		return func() tea.Msg {
			resp, err := a.Run(ctx, input)
			return ui.AgentResponseMsg{Content: resp, Err: err}
		}
	}

	app := ui.NewApp(ui.DefaultStyles(), modelName, maxTokens, submit)

	prog := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		return err
	}

	if quitting {
		runExitReview(ctx, client, store, session)
		fmt.Println("Goodbye.")
	}
	return nil
}

// helpText is the body shown when the user types /help in the UI. Returned
// as markdown so glamour can render it.
func helpText() string {
	return "**Commands**\n\n" +
		"- `/exit`, `/quit` — Exit Fev (generates session review)\n" +
		"- `/reset` — Clear conversation history\n" +
		"- `/help` — Show this help\n"
}

// runExitReview generates and prints a post-session review on /exit.
func runExitReview(ctx context.Context, client memory.ReviewClient, store *memory.Store, session *memory.Session) {
	if client == nil || store == nil || session == nil {
		return
	}
	fmt.Print("Generating session review... ")
	summary, err := memory.Review(ctx, client, session, store)
	if err != nil {
		fmt.Printf("(skipped: %v)\n", err)
		return
	}
	fmt.Printf("\nSession summary: %s\n", summary)
}

func runTask(a *agent.Agent, task string) error {
	resp, err := a.Run(context.Background(), task)
	if err != nil {
		return err
	}
	fmt.Println(resp)
	return nil
}

func buildSystemPrompt(projectConfig string, reg *tools.Registry, latestReview string) string {
	var b strings.Builder
	b.WriteString("You are Fev, a local-first coding assistant. You help users navigate and modify their codebase using structured tools.\n\n")
	b.WriteString("Available tools: " + strings.Join(reg.List(), ", ") + "\n\n")
	b.WriteString("Always use tools when you need to read, write, or search files. Do not guess file contents.\n")
	b.WriteString("When using the bash tool, always include a description of what the command does.\n")

	if projectConfig != "" {
		b.WriteString("\n--- Project Configuration ---\n")
		b.WriteString(projectConfig)
	}

	if latestReview != "" {
		b.WriteString("\n--- Previous Session ---\n")
		b.WriteString(latestReview)
	}

	return b.String()
}
