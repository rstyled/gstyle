package callflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestProgramBoundaries(t *testing.T) {
	var missing *Program
	if err := missing.Add(nil); err == nil {
		t.Fatal("nil program accepts registration")
	}
	if _, err := missing.ExitEffects(nil); err == nil {
		t.Fatal("nil program accepts a solve")
	}
	var program Program
	if err := program.Add(nil); err == nil {
		t.Fatal("nil graph is registered")
	}
	if err := program.Add(&pdg.Graph{}); err == nil {
		t.Fatal("invalid graph is registered")
	}
	summaries, err := program.ExitEffects(nil)
	if err != nil || len(summaries) != 0 {
		t.Fatalf("empty program: %v %v", summaries, err)
	}
}
