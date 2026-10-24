package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	output := flag.String("output", "dist/verification/programs", "directory for collected reports and Lean certificate")
	control := flag.Bool("controls", false, "collect the small grammar used by verification negative controls")
	flag.Parse()
	if err := run(context.Background(), *output, *control); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, output string, control bool) error {
	if err := os.MkdirAll(output, 0700); err != nil {
		return err
	}
	selected := suites()
	if control {
		selected = []suite{{Name: "controls", Leaves: []expression{{Input: true}, {Value: -1}, {}, {Value: 1}}}}
	}
	cases, source := corpus(selected)
	fmt.Printf("Collecting %d programs from %d exhaustive grammars\n", len(cases), len(selected))
	observed, err := collect(ctx, cases, source, representativeInputs(selected))
	if err != nil {
		return err
	}
	if err := writeArtifacts(output, source, selected, observed); err != nil {
		return err
	}

	fmt.Printf("Collected %d reports; Lean certificate: %s\n", len(observed), filepath.Join(output, "Certificate.lean"))
	return nil
}

func writeArtifacts(output, source string, selected []suite, observed []observation) error {
	digest := sha256.Sum256([]byte(source))
	manifest := struct {
		SourceSHA256 string        `json:"source_sha256"`
		Suites       []suite       `json:"suites"`
		Cases        []observation `json:"cases"`
	}{hex.EncodeToString(digest[:]), selected, observed}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	files := map[string][]byte{"input.go.txt": []byte(source), "observations.json": data, "Certificate.lean": []byte(certificate(observed, selected, source))}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(output, name), body, 0600); err != nil {
			return err
		}
	}
	return nil
}
