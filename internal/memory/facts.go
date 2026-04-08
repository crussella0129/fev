package memory

import (
	"fmt"
	"time"
)

// Fact is a grounded, verified claim about the codebase.
type Fact struct {
	ID         int64
	Claim      string
	SourceFile string
	SourceLine int
	CreatedAt  time.Time
	VerifiedAt time.Time
}

// InsertFact stores a new fact and returns its ID.
func (s *Store) InsertFact(claim, sourceFile string, sourceLine int) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO facts (claim, source_file, source_line) VALUES (?, ?, ?)`,
		claim, sourceFile, sourceLine,
	)
	if err != nil {
		return 0, fmt.Errorf("insert fact: %w", err)
	}
	return res.LastInsertId()
}

// FindFacts searches for facts whose claim matches query (LIKE %query%).
func (s *Store) FindFacts(query string, limit int) ([]Fact, error) {
	rows, err := s.db.Query(
		`SELECT id, claim, source_file, source_line, created_at, verified_at
		 FROM facts WHERE claim LIKE ? LIMIT ?`,
		"%"+query+"%", limit,
	)
	if err != nil {
		return nil, fmt.Errorf("find facts: %w", err)
	}
	defer rows.Close()
	return scanFacts(rows)
}

// FindFactsBySource returns all facts associated with a specific file.
func (s *Store) FindFactsBySource(sourceFile string) ([]Fact, error) {
	rows, err := s.db.Query(
		`SELECT id, claim, source_file, source_line, created_at, verified_at
		 FROM facts WHERE source_file = ?`,
		sourceFile,
	)
	if err != nil {
		return nil, fmt.Errorf("find facts by source: %w", err)
	}
	defer rows.Close()
	return scanFacts(rows)
}

// UpdateFactVerification marks a fact as re-verified at the current time.
func (s *Store) UpdateFactVerification(id int64) error {
	_, err := s.db.Exec(
		`UPDATE facts SET verified_at = CURRENT_TIMESTAMP WHERE id = ?`, id,
	)
	if err != nil {
		return fmt.Errorf("update fact verification: %w", err)
	}
	return nil
}

// DeleteFact removes a fact by ID.
func (s *Store) DeleteFact(id int64) error {
	_, err := s.db.Exec(`DELETE FROM facts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete fact: %w", err)
	}
	return nil
}

func scanFacts(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]Fact, error) {
	facts := []Fact{}
	for rows.Next() {
		var f Fact
		var createdStr, verifiedStr string
		if err := rows.Scan(&f.ID, &f.Claim, &f.SourceFile, &f.SourceLine, &createdStr, &verifiedStr); err != nil {
			return nil, fmt.Errorf("scan fact: %w", err)
		}
		f.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdStr)
		f.VerifiedAt, _ = time.Parse("2006-01-02 15:04:05", verifiedStr)
		facts = append(facts, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return facts, nil
}
