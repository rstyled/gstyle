// Command detector exports the actual non-nil solver witness for the lens fixture.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/rstyled/gstyle/adapters/gopatterns"
	"github.com/rstyled/gstyle/domain/interfacenil"
	"github.com/rstyled/gstyle/domain/pdg"
	"os"
)

// gstyle-allow(side_effect_density): CLI boundary performs typed extraction and writes the solver witness to stdout.
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// gstyle-allow(side_effect_density): Explicit evidence export loads Go source and writes JSON; extraction effects propagate here.
func run() error {
	path := "./adapters/gopatterns/testdata/interface-nil"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	ext, err := gopatterns.Extract(context.Background(), "", []string{path})
	if err != nil {
		return err
	}
	graphs := make([]pdg.Graph, len(ext.Funcs))
	for i := range ext.Funcs {
		graphs[i] = ext.Funcs[i].Pdg
	}
	proofs := interfacenil.Explain(graphs)
	return json.NewEncoder(os.Stdout).Encode(struct {
		Functions []string             `json:"functions"`
		Proofs    []interfacenil.Proof `json:"proofs"`
	}{Functions: functionIDs(graphs), Proofs: proofs})
}

func functionIDs(graphs []pdg.Graph) []string {
	ids := make([]string, len(graphs))
	for i, g := range graphs {
		ids[i] = g.Tables.Text(g.Function.ID)
	}
	return ids
}
