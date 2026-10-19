package goquality

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestCallDensityUsesTypedSourceOperations(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
 func edits(p *int){*p=1;*p=2;*p=3}
 func single(p *int){tmp:=0;edits(p);_=tmp}
 func separate(p *int){edits(p);edits(p);edits(p)}
 func makeFn() func(){return func(){}}
 func chained(){makeFn()()}
 `))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	calls := findFunction(t, facts, "separate").Calls
	if len(calls) != 3 || calls[0].Site == 0 || calls[0].Site == calls[1].Site || calls[1].Site == calls[2].Site {
		t.Fatalf("distinct call identities: %+v", calls)
	}
	chain := findFunction(t, facts, "chained").Calls
	if len(chain) != 2 || chain[0].Site == 0 || chain[0].Site == chain[1].Site {
		t.Fatalf("chained invocations share an identity: %+v", chain)
	}
	data, err := json.Marshal(facts)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &facts); err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name     string
		density  int64
		evidence int
	}{
		{"edits", 1000, 3}, {"single", 333, 3}, {"separate", 1000, 9},
	} {
		f := findFunction(t, facts, tc.name)
		for _, result := range report.Functions {
			if result.Name == f.Name && (result.DensityMilli != tc.density || len(result.Effects) != tc.evidence) {
				t.Fatalf("%s: density=%d evidence=%d", tc.name, result.DensityMilli, len(result.Effects))
			}
		}
	}
}
