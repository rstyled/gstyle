package taintflow

import (
	"errors"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestAnalysisWorkBoundaryRemainsExhausted(t *testing.T) {
	work := &analysisWork{remaining: 3}
	if !work.spend(3) || !work.available() || work.spend(1) || work.available() {
		t.Fatal("work limit does not enforce its inclusive boundary")
	}
	if work.spend(0) || work.available() {
		t.Fatal("an exhausted analysis resumes")
	}
}

func TestAnalyzeWorkLimitReturnsUnknownInsteadOfEmptySuccess(t *testing.T) {
	q := projectionQuery(t, "caller", 0)
	q.graph.NormalFlow = &pdg.NormalFlow{Entry: 1, Blocks: []pdg.FlowBlock{
		{Operations: []pdg.Ref{1}, Successors: []pdg.Ref{2, 3}, Reachable: true},
		{Operations: []pdg.Ref{2}, Successors: []pdg.Ref{4}, Reachable: true},
		{Successors: []pdg.Ref{4}, Reachable: true},
		{Operations: []pdg.Ref{3}, Reachable: true},
	}}
	graphs := map[string]*pdg.Graph{"caller": q.graph}
	models := Models{
		Source: func(*pdg.Graph, pdg.Ref) bool { return true },
		Sinks: func(*pdg.Graph, pdg.Ref) []Sink {
			return []Sink{{Kind: "test", Positions: []pdg.Position{1}}}
		},
	}
	if _, err := Analyze(graphs, models); err != nil {
		t.Fatalf("default analysis rejects the fixture: %v", err)
	}
	findings, err := analyze(graphs, models, 0)
	if !errors.Is(err, ErrWorkLimit) || findings != nil || !strings.Contains(err.Error(), "caller") {
		t.Fatalf("limited analysis reports findings=%v err=%v", findings, err)
	}
	if _, err := Analyze(graphs, models); err != nil {
		t.Fatalf("an exhausted analysis poisons the next run: %v", err)
	}
}

func TestStateClonesShareAnalysisWorkWithoutSharingReadCaches(t *testing.T) {
	store := emptyState()
	store.work = &analysisWork{remaining: 1}
	copy := cloneState(store)
	copy.joinValues(value{}, value{})
	store.joinValues(value{}, value{})
	if store.work.available() || copy.work.available() {
		t.Fatal("cloning a state resets the analysis work limit")
	}
}
