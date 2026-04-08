package memory

import (
	"fmt"
	"time"
)

// Correction records an explicit self-correction: "I believed X, but actually Y."
type Correction struct {
	ID         int64
	WrongClaim string
	RightClaim string
	CreatedAt  time.Time
}

// InsertCorrection records that a belief was wrong.
func (s *Store) InsertCorrection(wrongClaim, rightClaim string) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO corrections (wrong_claim, right_claim) VALUES (?, ?)`,
		wrongClaim, rightClaim,
	)
	if err != nil {
		return 0, fmt.Errorf("insert correction: %w", err)
	}
	return res.LastInsertId()
}

// RecentCorrections returns the N most recent corrections, newest first.
func (s *Store) RecentCorrections(limit int) ([]Correction, error) {
	rows, err := s.db.Query(
		`SELECT id, wrong_claim, right_claim, created_at FROM corrections ORDER BY id DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("recent corrections: %w", err)
	}
	defer rows.Close()

	corrections := []Correction{}
	for rows.Next() {
		var c Correction
		var createdUnix int64
		if err := rows.Scan(&c.ID, &c.WrongClaim, &c.RightClaim, &createdUnix); err != nil {
			return nil, fmt.Errorf("scan correction: %w", err)
		}
		c.CreatedAt = time.Unix(createdUnix, 0).UTC()
		corrections = append(corrections, c)
	}
	return corrections, rows.Err()
}
