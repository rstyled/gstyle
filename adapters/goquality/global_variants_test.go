package goquality

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStableGlobalsAgreeAcrossLoadedTestVariants(t *testing.T) {
	for _, tc := range []struct {
		name       string
		testSource string
		wantGlobal bool
	}{
		{
			name:       "unsafe test package",
			testSource: "package repro\nimport _ \"unsafe\"\nfunc testHelper() {}\n",
			wantGlobal: true,
		},
		{
			name:       "test writes table",
			testSource: "package repro\nfunc testHelper() { clear(fixed) }\n",
			wantGlobal: true,
		},
		{
			name:       "test writes error sentinel",
			testSource: "package repro\nfunc testHelper() { frozen = nil }\n",
			wantGlobal: true,
		},
		{
			name:       "test reads table",
			testSource: "package repro\nfunc testHelper() int { return fixed[\"a\"] }\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := qualitySourceModule(t, []byte(`package repro
import "errors"
var frozen = errors.New("fixed")
var fixed = map[string]int{"a": 1}
func Read() int { if frozen != nil { return fixed["a"] }; return 0 }
`))
			if err := os.WriteFile(filepath.Join(root, "input_test.go"), []byte(tc.testSource), 0o600); err != nil {
				t.Fatal(err)
			}
			production, err := (Analyzer{Root: root}).Extract(context.Background(), []string{"input.go"})
			if err != nil {
				t.Fatal(err)
			}
			for _, effect := range findFunction(t, production, "Read").Effects {
				if effect.Kind.GlobalState() {
					t.Fatalf("unloaded tests change production proof: %+v", effect)
				}
			}
			variants, err := (Analyzer{Root: root, Tests: true}).Extract(context.Background(), []string{"input.go"})
			if err != nil {
				t.Fatal(err)
			}
			reads := 0
			for _, fact := range variants {
				if strings.HasSuffix(fact.Name, ".Read") {
					reads++
				}
			}
			if reads != 1 {
				t.Fatalf("Read emitted %d times across test variants", reads)
			}
			global := false
			for _, effect := range findFunction(t, variants, "Read").Effects {
				global = global || effect.Kind.GlobalState()
			}
			if global != tc.wantGlobal {
				t.Fatalf("loaded test variants: global=%v want=%v", global, tc.wantGlobal)
			}
		})
	}
}
