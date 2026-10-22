package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/rstyled/gstyle/domain/quality"
)

type assuranceRow struct {
	Name, Family, Variant string
	Seed                  int
	Holdout, Complete     bool
	Writes                []assuranceWitness
	Classified, Uncertain int
	Missing               []assuranceWitness
	Effects               []quality.Effect
}

type assuranceAssessment struct {
	SchemaVersion int `json:"schema_version"`
	BinaryHash    string
	SourceHash    string
	TraceHash     string
	Executions    int
	Classified    int
	Uncertain     int
	Missing       int
	Rows          []assuranceRow
}

// This bounded test runs in normal CI. Explicit external binaries and artifact
// directories are optional, so the exact experiment also runs against overlays.
func TestMutationAssurance(t *testing.T) {
	cases, families := assuranceCorpus(t)
	source := assuranceProgram(cases)
	if selected := os.Getenv("GSTYLE_ASSURANCE_CASE"); selected != "" {
		found := false
		for _, c := range cases {
			if c.Name == selected {
				c.Base = c.Name
				cases = []specimen{c}
				source = assuranceReducedProgram(t, c)
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("unknown assurance case %q", selected)
		}
	}
	assuranceAssess(t, cases, families, source)
}

func assuranceAssess(t *testing.T, cases []specimen, families map[string]assuranceFamily, source []byte) {
	t.Helper()
	if err := equivalentSource(source, cases); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	check := runner{root: root, binary: os.Getenv("GSTYLE_ASSURANCE_BINARY"), timeout: time.Minute}
	if err := check.write(source, cases); err != nil {
		t.Fatal(err)
	}
	assuranceWrite(t, root, "trace.go", []byte(assuranceTrace))
	assuranceWrite(t, filepath.Join(root, "oracle"), "main.go", assuranceOracle(cases))
	plain := assuranceExecute(t, root)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	tracedSource, err := assuranceInstrument(ctx, root, source)
	if err != nil {
		t.Fatal(err)
	}
	assuranceWrite(t, root, "input.go", tracedSource)
	traced := assuranceExecute(t, root)
	assuranceWrite(t, root, "input.go", source)
	if err := os.Remove(filepath.Join(root, "trace.go")); err != nil {
		t.Fatal(err)
	}
	facts, err := check.extract(ctx)
	if err != nil {
		t.Fatal(err)
	}
	report, err := check.evaluate(ctx, facts)
	if err != nil {
		t.Fatal(err)
	}
	results := map[string]quality.FunctionResult{}
	for _, result := range report.Functions {
		results[shortName(result.Name)] = result
	}
	proofs := map[string]evidence{}
	for _, fact := range facts {
		name := shortName(fact.Name)
		proofs[name] = signature(fact, results[name])
	}
	for _, c := range cases {
		if !reflect.DeepEqual(proofs[c.Name], proofs[c.Base]) {
			t.Fatalf("equivalent syntax changes evidence: %s", c.Name)
		}
	}
	assessment := assuranceAssessment{SchemaVersion: 1, SourceHash: assuranceHash(source), TraceHash: assuranceHash(tracedSource)}
	if check.binary != "" {
		data, err := os.ReadFile(check.binary)
		if err != nil {
			t.Fatal(err)
		}
		assessment.BinaryHash = assuranceHash(data)
	}
	for _, c := range cases {
		result, ok := results[c.Name]
		if !ok {
			t.Fatalf("analyzer omitted %s", c.Name)
		}
		for _, seed := range []int{-1, 0, 7} {
			key := fmt.Sprintf("%s/%d", c.Name, seed)
			original, present := plain[key]
			observation, tracedPresent := traced[key]
			if !present || !tracedPresent {
				t.Fatalf("runtime omitted %s", key)
			}
			assuranceCheckObservation(t, key, original, observation, families[c.Name], seed)
			row := assuranceCompare(c, families[c.Name], seed, observation, result)
			assessment.Rows = append(assessment.Rows, row)
			assessment.Executions++
			assessment.Classified += row.Classified
			assessment.Uncertain += row.Uncertain
			assessment.Missing += len(row.Missing)
		}
	}
	assuranceSave(t, assessment, source, tracedSource, plain, traced, facts, report, cases)
	t.Logf("%d executions: %d classified writes; %d covered by uncertainty; %d missing", assessment.Executions, assessment.Classified, assessment.Uncertain, assessment.Missing)
	if assessment.Missing != 0 {
		for _, row := range assessment.Rows {
			if len(row.Missing) != 0 {
				t.Fatalf("caller write missing: %s/%d (%s): %+v; effects %+v", row.Name, row.Seed, row.Family, row.Missing, row.Effects)
			}
		}
	}
}

func assuranceCompare(c specimen, family assuranceFamily, seed int, observed assuranceObservation, result quality.FunctionResult) assuranceRow {
	row := assuranceRow{Name: c.Name, Family: family.name, Variant: c.Variant, Seed: seed, Holdout: family.holdout, Complete: result.Complete, Writes: observed.Writes, Effects: result.Effects}
	for _, witness := range observed.Writes {
		switch assuranceCoverage(witness, result.Effects) {
		case quality.MutationEffect:
			row.Classified++
		case quality.UnknownEffect:
			row.Uncertain++
		default:
			row.Missing = append(row.Missing, witness)
		}
	}
	return row
}

func assuranceCoverage(witness assuranceWitness, effects []quality.Effect) quality.Kind {
	covered := quality.None
	for _, effect := range effects {
		if effect.Line != witness.CaseLine {
			continue
		}
		if effect.Kind == quality.MutationEffect {
			return effect.Kind
		}
		if effect.Kind == quality.UnknownEffect {
			covered = effect.Kind
		}
	}
	return covered
}

func assuranceHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func assuranceSave(t *testing.T, assessment assuranceAssessment, source, tracedSource []byte, plain, traced map[string]assuranceObservation, facts []quality.Function, report quality.Report, cases []specimen) {
	t.Helper()
	directory := os.Getenv("GSTYLE_ASSURANCE_OUTPUT")
	if directory == "" {
		return
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	assuranceWrite(t, directory, "input.go.txt", source)
	assuranceWrite(t, directory, "traced.go.txt", tracedSource)
	assuranceWrite(t, directory, "observer.go.txt", []byte(assuranceTrace))
	assuranceWrite(t, directory, "oracle.go.txt", assuranceOracle(cases))
	objects := map[string]any{"assessment.json": assessment, "runtime-original.json": plain, "runtime-traced.json": traced, "facts.json": facts, "report.json": report}
	for name, object := range objects {
		data, err := json.MarshalIndent(object, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		assuranceWrite(t, directory, name, data)
	}
	for _, row := range assessment.Rows {
		if len(row.Missing) == 0 {
			continue
		}
		for _, c := range cases {
			if c.Name == row.Name {
				// Preserve the first witness as a standalone seed. A reduction must
				// re-run this oracle; legacy boolean checks cannot validate it.
				seed := assuranceProgram([]specimen{c})
				if strings.HasPrefix(c.Family, "external-") {
					seed = source
				}
				assuranceWrite(t, directory, "counterexample-seed.go.txt", seed)
				return
			}
		}
	}
}

func TestMutationAssuranceRejectsUnrelatedEvidence(t *testing.T) {
	witness := assuranceWitness{Target: "Count", Source: 8, CaseLine: 8}
	for _, effects := range [][]quality.Effect{
		{{Kind: quality.IO, Line: 8}},
		{{Kind: quality.MutationEffect, Line: 9}},
		{{Kind: quality.UnknownEffect, Line: 9}},
	} {
		if got := assuranceCoverage(witness, effects); got != quality.None {
			t.Fatalf("unrelated evidence covers a write: %s", got)
		}
	}
	for _, kind := range []quality.Kind{quality.MutationEffect, quality.UnknownEffect} {
		if got := assuranceCoverage(witness, []quality.Effect{{Kind: kind, Line: 8}}); got != kind {
			t.Fatalf("matching evidence lost: %s", got)
		}
	}
}

func TestMutationAssuranceRejectsEffectfulPlaces(t *testing.T) {
	root := t.TempDir()
	source := []byte("package repro\nfunc index() int { return 0 }; func Change(values []int) { values[index()]++ }\n")
	assuranceWrite(t, root, "go.mod", []byte("module example.com/quality-hunt\ngo 1.25.0\n"))
	assuranceWrite(t, root, "input.go", source)
	_, err := assuranceInstrument(context.Background(), root, source)
	if err == nil || !strings.Contains(err.Error(), "unsupported integer write") {
		t.Fatalf("effectful place accepted: %v", err)
	}
}

func TestFrozenPolicyReportIsNotReevaluated(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the independent CLI fixture uses a POSIX shell")
	}
	root := t.TempDir()
	binary := filepath.Join(root, "frozen-policy")
	// Current policy gives this body no effects. The frozen fixture gives IO.
	data := []byte("#!/bin/sh\nprintf '%s\\n' '{\"schema_version\":1,\"functions\":[{\"name\":\"p.Local\",\"complete\":true,\"effects\":[{\"kind\":\"io\",\"detail\":\"frozen policy\"}]}]}'\nexit 1\n")
	assuranceWrite(t, root, "frozen-policy", data)
	if err := os.Chmod(binary, 0700); err != nil {
		t.Fatal(err)
	}
	facts := []quality.Function{{Location: quality.Location{Name: "p.Local"}, Statements: 1, CodeLines: 1}}
	check := runner{root: root, binary: binary}
	report, err := check.evaluate(context.Background(), facts)
	if err != nil || len(report.Functions) != 1 || len(report.Functions[0].Effects) != 1 || report.Functions[0].Effects[0].Kind != quality.IO {
		t.Fatalf("frozen policy was replaced or finding exit rejected: %+v, %v", report, err)
	}
	assuranceWrite(t, root, "frozen-policy", []byte("#!/bin/sh\nprintf '%s\\n' '{}'\nexit 2\n"))
	if _, err := check.evaluate(context.Background(), facts); err == nil {
		t.Fatal("analysis failure was accepted as a report")
	}
	assuranceWrite(t, root, "frozen-policy", []byte("#!/bin/sh\nprintf '%s\\n' '{}'\nexit 0\n"))
	if _, err := check.evaluate(context.Background(), facts); err == nil {
		t.Fatal("an unversioned report was accepted")
	}
}
