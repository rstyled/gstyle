package patternfix

import (
	"bytes"
	"context"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func TestNoWriteCycleExtractsOnceForAllDependentKinds(t *testing.T) {
	loads := 0
	state := &phasedState{revision: fixRevision{mine: func(_ context.Context, opts options) ([]patterns.Candidate, error) {
		loads++
		if len(opts.paths) != 2 {
			t.Fatal("cross-file input scope narrowed")
		}
		return nil, nil
	}}}
	var output bytes.Buffer
	for range 2 {
		fixed, err := runPhasedCycle(context.Background(), options{kind: "all", paths: []string{"first", "second"}}, &output, state)
		if err != nil || fixed != 0 {
			t.Fatalf("empty cycle: %d, %v", fixed, err)
		}
	}
	if loads != 1 {
		t.Fatalf("unchanged revision extracted %d times", loads)
	}
}

func TestRevisionRefreshesCandidatesAfterSourceWrite(t *testing.T) {
	loads := 0
	revision := fixRevision{mine: func(context.Context, options) ([]patterns.Candidate, error) {
		loads++
		return []patterns.Candidate{{Kind: patterns.ValueObject, ScoreMilli: uint32(loads)}}, nil
	}}
	first, err := revision.load(context.Background(), options{})
	if err != nil {
		t.Fatal(err)
	}
	revision.invalidate()
	second, err := revision.load(context.Background(), options{})
	if err != nil {
		t.Fatal(err)
	}
	if first[0].ScoreMilli == second[0].ScoreMilli || loads != 2 {
		t.Fatal("old candidates survive changed source revision")
	}
}

func TestDependentKindsNeverShareAPhaseAndOrderingRunsLast(t *testing.T) {
	previous := -1
	for _, phase := range fixPhases {
		if len(phase) != 1 {
			t.Fatal("dependent kinds share stale source evidence")
		}
		rank := patterns.FixKindRank(phase[0])
		if rank <= previous {
			t.Fatal("phase dependency order differs from domain order")
		}
		previous = rank
	}
	if fixPhases[len(fixPhases)-1][0] != patterns.DeclarationOrder {
		t.Fatal("ordering does not see final declarations")
	}
}

func TestDefaultInputScopeRunsPhases(t *testing.T) {
	loads := 0
	state := &phasedState{revision: fixRevision{mine: func(context.Context, options) ([]patterns.Candidate, error) { loads++; return nil, nil }}}
	if _, err := runPhasedCycle(context.Background(), options{kind: "all"}, &bytes.Buffer{}, state); err != nil {
		t.Fatal(err)
	}
	if loads != 1 {
		t.Fatal("default current-directory scope is skipped")
	}
}
