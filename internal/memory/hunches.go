package memory

import (
	"fmt"
	"time"
)

// Hunch is an unverified belief with a confidence score.
type Hunch struct {
	ID         int64
	Belief     string
	Confidence float64 // 0.0 to 1.0
	CreatedAt  time.Time
}

// InsertHunch stores a new hunch. Confidence is clamped to [0.0, 1.0].
func (s *Store) InsertHunch(belief string, confidence float64) (int64, error) {
	if confidence < 0.0 {
		confidence = 0.0
	}
	if confidence > 1.0 {
		confidence = 1.0
	}
	res, err := s.db.Exec(
		`INSERT INTO hunches (belief, confidence) VALUES (?, ?)`,
		belief, confidence,
	)
	if err != nil {
		return 0, fmt.Errorf("insert hunch: %w", err)
	}
	return res.LastInsertId()
}

// FindHunches searches for hunches whose belief matches query (LIKE %query%).
func (s *Store) FindHunches(query string, limit int) ([]Hunch, error) {
	rows, err := s.db.Query(
		`SELECT id, belief, confidence, created_at FROM hunches WHERE belief LIKE ? LIMIT ?`,
		"%"+query+"%", limit,
	)
	if err != nil {
		return nil, fmt.Errorf("find hunches: %w", err)
	}
	defer rows.Close()

	hunches := []Hunch{}
	for rows.Next() {
		var h Hunch
		var createdUnix int64
		if err := rows.Scan(&h.ID, &h.Belief, &h.Confidence, &createdUnix); err != nil {
			return nil, fmt.Errorf("scan hunch: %w", err)
		}
		h.CreatedAt = time.Unix(createdUnix, 0).UTC()
		hunches = append(hunches, h)
	}
	return hunches, rows.Err()
}

// DeleteHunch removes a hunch by ID (used when promoted to fact or disproven).
func (s *Store) DeleteHunch(id int64) error {
	_, err := s.db.Exec(`DELETE FROM hunches WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete hunch: %w", err)
	}
	return nil
}
