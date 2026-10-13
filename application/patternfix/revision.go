package patternfix

import (
	"context"

	"github.com/rstyled/gstyle/domain/patterns"
)

type candidateMiner func(context.Context, options) ([]patterns.Candidate, error)

// A revision owns candidates only until a write changes their source inputs.
// All phases retain the complete input scope for cross-file dependencies.
type fixRevision struct {
	mine       candidateMiner
	candidates []patterns.Candidate
	valid      bool
}

func (r *fixRevision) load(ctx context.Context, opts options) ([]patterns.Candidate, error) {
	if r.valid {
		return r.candidates, nil
	}
	candidates, err := r.mine(ctx, opts)
	if err != nil {
		return nil, err
	}
	r.candidates = candidates
	r.valid = true
	return candidates, nil
}

func (r *fixRevision) invalidate() {
	r.candidates = nil
	r.valid = false
}

// Separate dependent kinds so every subsequent edit sees its predecessor's
// committed source revision. Zero-write stages reuse the same extraction.
func dependencyPhases() [][]patterns.CandidateKind {
	var phases [][]patterns.CandidateKind
	for _, kind := range patterns.FixKindOrder {
		phases = append(phases, []patterns.CandidateKind{kind})
		if kind == patterns.DeclarationOrder {
			break
		}
	}
	return phases
}
