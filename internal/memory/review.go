package memory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/crussella0129/fev/internal/core"
	"github.com/crussella0129/fev/internal/llm"
)

// ReviewClient is any LLM client that can generate a summary.
// Structurally compatible with agent.LLMClient.
type ReviewClient interface {
	Generate(ctx context.Context, messages []core.Message, tools []llm.ToolSchema) (*core.Message, error)
}

// Review generates a post-session summary and writes it to the session directory.
// It collects unresolved questions and recent corrections, asks the LLM for a
// one-sentence summary, and saves it to session.Dir/review.md.
func Review(ctx context.Context, client ReviewClient, session *Session, store *Store) (string, error) {
	var b strings.Builder

	b.WriteString("Summarize this session in one sentence. Include what was accomplished and what remains unresolved.\n\n")

	if len(session.Questions) > 0 {
		b.WriteString("Unresolved questions:\n")
		for _, q := range session.Questions {
			fmt.Fprintf(&b, "- %s\n", q)
		}
		b.WriteString("\n")
	}

	corrections, err := store.RecentCorrections(10)
	if err == nil && len(corrections) > 0 {
		b.WriteString("Corrections made this session:\n")
		for _, c := range corrections {
			fmt.Fprintf(&b, "- Was: %s → Now: %s\n", c.WrongClaim, c.RightClaim)
		}
		b.WriteString("\n")
	}

	if len(session.Observations) > 0 {
		b.WriteString("Key observations:\n")
		for _, o := range session.Observations {
			fmt.Fprintf(&b, "- %s\n", o)
		}
	}

	resp, err := client.Generate(ctx, []core.Message{core.NewUserMessage(b.String())}, nil)
	if err != nil {
		return "", fmt.Errorf("review generate: %w", err)
	}
	summary := resp.Content

	reviewPath := filepath.Join(session.Dir, "review.md")
	content := fmt.Sprintf("# Session Review\n\n%s\n", summary)
	if err := os.WriteFile(reviewPath, []byte(content), 0644); err != nil {
		return summary, fmt.Errorf("write review: %w", err)
	}

	return summary, nil
}

// LoadLatestReview finds the most recent session's review.md and returns its content.
// Returns empty string (no error) if no sessions exist yet.
func LoadLatestReview(baseDir string) (string, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read sessions dir: %w", err)
	}

	// Collect directory names (session IDs are timestamp-prefixed, so name sort = time sort).
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	if len(dirs) == 0 {
		return "", nil
	}

	sort.Strings(dirs)
	latest := dirs[len(dirs)-1]

	reviewPath := filepath.Join(baseDir, latest, "review.md")
	data, err := os.ReadFile(reviewPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // session exists but no review yet
		}
		return "", fmt.Errorf("read review: %w", err)
	}
	return string(data), nil
}
