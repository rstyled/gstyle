package gopatterns

import (
	"context"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

// TestMinerSmoke runs the full patterns pipeline on the shapes fixture:
// extractor -> FuncFacts -> report.Run -> candidates.
func TestMinerSmoke(t *testing.T) {
	ctx := context.Background()
	ex, err := Extract(ctx, "testdata/shapes", []string{"."})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	pdgs := ex.Funcs
	if len(pdgs) == 0 {
		t.Fatal("no PDGs extracted")
	}
	t.Logf("extracted %d PDGs", len(pdgs))

	// Convert to FuncFacts with SigKey derived from Param TyClasses.
	var facts []*patterns.FuncFacts
	for _, fp := range pdgs {
		var params []string
		for _, n := range patterns.MiningView(&fp.Pdg).Nodes {
			if n.Kind == patterns.Param {
				params = append(params, n.TyClass)
			}
		}
		facts = append(facts, &patterns.FuncFacts{
			ID:     "test/shapes." + fp.Name,
			Name:   fp.Name,
			Path:   fp.Path,
			Line:   fp.Line,
			Pdg:    &fp.Pdg,
			SigKey: "fn(" + strings.Join(params, ",") + ")",
		})
	}

	report := patterns.Run(facts, patterns.Options{})
	text := patterns.Text(&report)
	t.Logf("report:\n%s", text)
	if len(report.Candidates) == 0 {
		t.Fatal("expected candidates from parallel pairs, got none")
	}
	// The retry pair has parallel operation/control graphs. The validate/insert
	// examples differ in parameter types and call arity; lexical similarity alone
	// is no longer sufficient evidence that they must form a candidate.
	found := false
	for _, c := range report.Candidates {
		fetch, load := false, false
		for _, s := range c.Sites {
			fetch = fetch || strings.HasSuffix(s.Name, ".FetchWithRetry")
			load = load || strings.HasSuffix(s.Name, ".LoadWithRetry")
		}
		found = found || (fetch && load)
		t.Logf("candidate: %s score=%d sites=%d", c.Kind, c.ScoreMilli, len(c.Sites))
	}
	if !found {
		t.Fatal("expected candidate pairing FetchWithRetry and LoadWithRetry")
	}
}
