package patterns

import (
	"reflect"
	"testing"
)

func TestLocalGuardFixesMatchFullMining(t *testing.T) {
	facts := []*FuncFacts{{ID: "p.work", Name: "work", Path: "work.go", Line: 2, EndLine: 10, GuardClauses: []GuardClauseHit{{Line: 3, BodyStmts: 2}}}}
	full := Run(facts, Options{})
	var expected []Candidate
	for _, candidate := range full.Candidates {
		if candidate.Kind == GuardClause {
			expected = append(expected, candidate)
		}
	}
	actual := RunFixCandidates(facts, Options{}, GuardClause)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("local fix candidates differ from full mining: %#v != %#v", actual, expected)
	}
}
