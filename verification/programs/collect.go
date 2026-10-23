package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rstyled/gstyle/adapters/gopatterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/runtimeguard"
)

type observation struct {
	Name    string             `json:"name"`
	Suite   string             `json:"suite"`
	Program program            `json:"program"`
	Source  string             `json:"source"`
	Range   runtimeguard.Value `json:"range"`
}

func corpus(selected []suite) ([]observation, string) {
	var cases []observation
	var source strings.Builder
	source.WriteString("package generated\nfunc observe(value int64) {}\n")
	for _, s := range selected {
		for _, p := range enumerate(s, s.Depth) {
			name := fmt.Sprintf("Case%d", len(cases))
			body := p.source(name)
			cases = append(cases, observation{Name: name, Suite: s.Name, Program: p, Source: body})
			source.WriteString(body)
		}
	}
	return cases, source.String()
}

func collect(ctx context.Context, cases []observation, source string, inputs []int64) ([]observation, error) {
	root, err := os.MkdirTemp("", "gstyle-program-proof-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(root)
	if err := writeModule(root, source); err != nil {
		return nil, err
	}
	if err := checkRuntime(ctx, root, cases, inputs); err != nil {
		return nil, err
	}

	extraction, err := gopatterns.Extract(ctx, root, []string{"input.go"})
	if err != nil {
		return nil, err
	}
	reports, err := observedRanges(extraction.Funcs, cases)
	if err != nil {
		return nil, err
	}
	return attachReports(cases, reports)
}

func attachReports(cases []observation, reports map[string]runtimeguard.Value) ([]observation, error) {
	for i := range cases {
		value, ok := reports[cases[i].Name]
		if !ok {
			return nil, fmt.Errorf("analyzer omitted observation for %s", cases[i].Name)
		}
		cases[i].Range = value
		delete(reports, cases[i].Name)
	}
	if len(reports) != 0 {
		return nil, fmt.Errorf("unexpected analyzer observations: %v", reports)
	}
	return cases, nil
}

func observedRanges(functions []gopatterns.FuncPdg, cases []observation) (map[string]runtimeguard.Value, error) {
	wanted := make(map[string]bool, len(cases))
	for _, c := range cases {
		wanted[c.Name] = true
	}
	out := make(map[string]runtimeguard.Value)
	for _, function := range functions {
		name := function.Name[strings.LastIndex(function.Name, ".")+1:]
		if !wanted[name] {
			continue
		}
		if _, exists := out[name]; exists {
			return nil, fmt.Errorf("duplicate function %s", name)
		}
		value, err := observedRange(&function.Pdg)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		out[name] = value
	}
	return out, nil
}

func observedRange(graph *pdg.Graph) (runtimeguard.Value, error) {
	report, err := runtimeguard.Analyze(graph)
	if err != nil {
		return runtimeguard.Value{}, err
	}
	var values []runtimeguard.Value
	for _, call := range report.Calls {
		node := graph.Nodes[call.Node-1]
		attrs := graph.NodeAttributes(node)
		if attrs.Callee.Status == pdg.Known && graph.Tables.Text(attrs.Callee.Value) == "example.com/generated.observe" {
			values = append(values, call.FirstArgument)
		}
	}
	if len(values) != 1 {
		return runtimeguard.Value{}, fmt.Errorf("expected one result observation, got %d; calls=%+v", len(values), callNames(graph, report))
	}
	return values[0], nil
}

func callNames(graph *pdg.Graph, report runtimeguard.Report) []string {
	var names []string
	for _, call := range report.Calls {
		names = append(names, graph.Tables.Text(graph.NodeAttributes(graph.Nodes[call.Node-1]).Callee.Value))
	}
	return names
}

func writeModule(root, source string) error {
	files := map[string]string{"go.mod": "module example.com/generated\n\ngo 1.26.0\n", "input.go": source}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0600); err != nil {
			return err
		}
	}
	return nil
}
