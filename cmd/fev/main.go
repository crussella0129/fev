// cmd/fev/main.go
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/crussella0129/fev/internal/agent"
	"github.com/crussella0129/fev/internal/config"
	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/ctxwin"
	"github.com/crussella0129/fev/internal/llm"
	"github.com/crussella0129/fev/internal/tools"
	"github.com/spf13/cobra"
)

var version = "0.1.0"

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

	// Load project config
	projectCfg, _ := config.LoadProjectConfig(ws.Root())
	systemPrompt := buildSystemPrompt(projectCfg, reg)
	a.SetSystemPrompt(systemPrompt)

	// Banner
	fmt.Printf("fev v%s — model: %s @ %s\n", version, cfg.Model.ModelName, cfg.Model.BaseURL)
	fmt.Printf("workspace: %s\n", ws.Root())
	fmt.Println("Type /help for commands, /exit to quit.")
	fmt.Println()

	// If task provided as args, run it
	if len(args) > 0 {
		task := strings.Join(args, " ")
		return runTask(a, task)
	}

	// Interactive REPL
	return repl(a)
}

func repl(a *agent.Agent) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Slash commands
		switch {
		case input == "/exit" || input == "/quit":
			fmt.Println("Goodbye.")
			return nil
		case input == "/help":
			printHelp()
			continue
		case input == "/reset":
			a.Reset()
			fmt.Println("Conversation reset.")
			continue
		}

		resp, err := a.Run(ctx, input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}
		fmt.Println(resp)
		fmt.Println()
	}
	return scanner.Err()
}

func runTask(a *agent.Agent, task string) error {
	resp, err := a.Run(context.Background(), task)
	if err != nil {
		return err
	}
	fmt.Println(resp)
	return nil
}

func printHelp() {
	fmt.Println("Commands:")
	fmt.Println("  /exit, /quit    Exit Fev")
	fmt.Println("  /reset          Clear conversation history")
	fmt.Println("  /help           Show this help")
}

func buildSystemPrompt(projectConfig string, reg *tools.Registry) string {
	var b strings.Builder
	b.WriteString("You are Fev, a local-first coding assistant. You help users navigate and modify their codebase using structured tools.\n\n")
	b.WriteString("Available tools: " + strings.Join(reg.List(), ", ") + "\n\n")
	b.WriteString("Always use tools when you need to read, write, or search files. Do not guess file contents.\n")
	b.WriteString("When using the bash tool, always include a description of what the command does.\n")

	if projectConfig != "" {
		b.WriteString("\n--- Project Configuration ---\n")
		b.WriteString(projectConfig)
	}

	return b.String()
}
