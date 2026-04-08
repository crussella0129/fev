package memory

import (
	"fmt"
	"os"
)

// VerifyStatus describes the result of checking a fact against the filesystem.
type VerifyStatus int

const (
	VerifyNotFound VerifyStatus = iota // no matching fact in memory
	VerifyValid                        // fact exists and source file is unchanged
	VerifyStale                        // fact exists but source file was modified after verification
)

// VerificationResult is returned by Verifier.Verify.
type VerificationResult struct {
	Status    VerifyStatus
	Fact      *Fact
	Stale     bool
	Corrected bool // true if a correction was recorded and the fact deleted
}

// Verifier checks facts against the filesystem.
type Verifier struct {
	store *Store
}

// NewVerifier creates a Verifier backed by the given store.
func NewVerifier(store *Store) *Verifier {
	return &Verifier{store: store}
}

// Verify checks if a claim about a file is still true.
//
//  1. Search for facts matching the claim.
//  2. If none found → VerifyNotFound.
//  3. Stat the source file to get its modification time.
//  4. If mtime > fact.VerifiedAt → VerifyStale (the source may have changed).
//  5. Otherwise update verified_at → VerifyValid.
func (v *Verifier) Verify(claim, sourceFile string) (*VerificationResult, error) {
	facts, err := v.store.FindFacts(claim, 1)
	if err != nil {
		return nil, fmt.Errorf("verify: find facts: %w", err)
	}
	if len(facts) == 0 {
		return &VerificationResult{Status: VerifyNotFound}, nil
	}

	f := facts[0]

	info, err := os.Stat(sourceFile)
	if err != nil {
		// If the file is gone, the fact is definitely stale.
		return &VerificationResult{Status: VerifyStale, Fact: &f, Stale: true}, nil
	}

	if info.ModTime().After(f.VerifiedAt) {
		return &VerificationResult{Status: VerifyStale, Fact: &f, Stale: true}, nil
	}

	// File hasn't changed since we last verified — refresh the timestamp.
	if err := v.store.UpdateFactVerification(f.ID); err != nil {
		return nil, fmt.Errorf("verify: update verification: %w", err)
	}
	return &VerificationResult{Status: VerifyValid, Fact: &f}, nil
}
