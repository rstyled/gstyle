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

type collectOptions struct {
	output, source       string
	controls, structural bool
}

type collection struct {
	SourceSHA256 string        `json:"source_sha256"`
	Suites       []suite       `json:"suites"`
	Cases        []observation `json:"cases"`
}

func main() {
	var opts collectOptions
	flag.StringVar(&opts.output, "output", "dist/verification/programs", "directory for collected reports and Lean certificate")
	flag.BoolVar(&opts.controls, "controls", false, "collect the small grammar used by verification negative controls")
	flag.StringVar(&opts.source, "source", "", "canonical branch source to certify instead of enumerating fixed grammars")
	flag.BoolVar(&opts.structural, "structural", false, "use the arbitrary-depth structural range checker")
	flag.Parse()
	if err := run(context.Background(), opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, opts collectOptions) error {
	if err := os.MkdirAll(opts.output, 0700); err != nil {
		return err
	}
	selected, cases, source, inputs, err := selectCorpus(opts.source, opts.controls)
	if err != nil {
		return err
	}
	fmt.Printf("Collecting %d programs\n", len(cases))
	observed, err := collect(ctx, cases, source, inputs)
	if err != nil {
		return err
	}
	if err := writeArtifacts(opts.output, source, collection{Suites: selected, Cases: observed}, opts.structural || opts.source != ""); err != nil {
		return err
	}

	fmt.Printf("Collected %d reports; Lean certificate: %s\n", len(observed), filepath.Join(opts.output, "Certificate.lean"))
	return nil
}

func selectCorpus(input string, control bool) ([]suite, []observation, string, []int64, error) {
	if input != "" {
		if control {
			return nil, nil, "", nil, fmt.Errorf("--source and --controls cannot be combined")
		}
		cases, source, inputs, err := suppliedCorpus(input)
		return nil, cases, source, inputs, err
	}
	selected := suites()
	if control {
		selected = []suite{{Name: "controls", Leaves: []expression{{Input: true}, {Value: -1}, {}, {Value: 1}}}}
	}
	cases, source := corpus(selected)
	return selected, cases, source, representativeInputs(selected), nil
}

func writeArtifacts(output, source string, collected collection, structural bool) error {
	digest := sha256.Sum256([]byte(source))
	collected.SourceSHA256 = hex.EncodeToString(digest[:])
	data, err := json.MarshalIndent(collected, "", "  ")
	if err != nil {
		return err
	}
	proof := certificate
	if structural {
		proof = structuralCertificate
	}
	files := map[string][]byte{"input.go.txt": []byte(source), "observations.json": data, "Certificate.lean": []byte(proof(collected.Cases, collected.Suites, source))}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(output, name), body, 0600); err != nil {
			return err
		}
	}
	return nil
}
