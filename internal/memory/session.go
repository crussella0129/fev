package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Session tracks structured notes for the current conversation.
type Session struct {
	ID        string
	Dir       string
	StartedAt time.Time

	Observations []string
	Decisions    []string
	Questions    []string
	Plan         []string
}

// NewSession creates a new session with a unique ID and directory structure.
func NewSession(baseDir string) (*Session, error) {
	id := sessionID()
	dir := filepath.Join(baseDir, id)

	if err := os.MkdirAll(filepath.Join(dir, "tool-results"), 0755); err != nil {
		return nil, fmt.Errorf("create session dir: %w", err)
	}

	return &Session{
		ID:        id,
		Dir:       dir,
		StartedAt: time.Now(),
	}, nil
}

// AddObservation records something the agent observed.
func (s *Session) AddObservation(obs string) {
	s.Observations = append(s.Observations, obs)
}

// AddDecision records a decision and its reasoning.
func (s *Session) AddDecision(dec string) {
	s.Decisions = append(s.Decisions, dec)
}

// AddQuestion records an unresolved question.
func (s *Session) AddQuestion(q string) {
	s.Questions = append(s.Questions, q)
}

// SetPlan replaces the current plan.
func (s *Session) SetPlan(items []string) {
	s.Plan = make([]string, len(items))
	copy(s.Plan, items)
}

// Save writes the session memory to session_memory.md.
func (s *Session) Save() error {
	var b strings.Builder

	writeSection := func(heading string, items []string) {
		fmt.Fprintf(&b, "## %s\n", heading)
		if len(items) == 0 {
			b.WriteString("_(none)_\n")
		} else {
			for _, item := range items {
				fmt.Fprintf(&b, "- %s\n", item)
			}
		}
		b.WriteString("\n")
	}

	writeSection("Observations", s.Observations)
	writeSection("Decisions", s.Decisions)
	writeSection("Questions", s.Questions)
	writeSection("Plan", s.Plan)

	path := filepath.Join(s.Dir, "session_memory.md")
	if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
		return fmt.Errorf("save session: %w", err)
	}
	return nil
}

// LoadSession reads session_memory.md from dir back into a Session struct.
func LoadSession(dir string) (*Session, error) {
	path := filepath.Join(dir, "session_memory.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load session: %w", err)
	}

	s := &Session{Dir: dir}
	s.ID = filepath.Base(dir)

	var current *[]string
	for _, line := range strings.Split(string(data), "\n") {
		switch strings.TrimSpace(line) {
		case "## Observations":
			current = &s.Observations
		case "## Decisions":
			current = &s.Decisions
		case "## Questions":
			current = &s.Questions
		case "## Plan":
			current = &s.Plan
		default:
			if current != nil && strings.HasPrefix(line, "- ") {
				*current = append(*current, strings.TrimPrefix(line, "- "))
			}
		}
	}
	return s, nil
}

// ToolResultPath returns a path for storing large tool output inside the session.
func (s *Session) ToolResultPath(toolName string) string {
	ts := time.Now().Format("150405")
	return filepath.Join(s.Dir, "tool-results", toolName+"_"+ts+".txt")
}

// sessionID generates a unique session ID based on timestamp + 4-char hex suffix.
func sessionID() string {
	now := time.Now()
	// Use nanoseconds for the suffix to ensure uniqueness within the same second.
	suffix := fmt.Sprintf("%04x", now.Nanosecond()&0xffff)
	return now.Format("20060102-150405") + "-" + suffix
}
