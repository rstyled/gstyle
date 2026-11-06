package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// The boundary keeps frontend node IDs intact. rstyle owns validation and derivation.
type cpgSpan struct {
	File  string `json:"file"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}
type cpgFile struct {
	ID      string `json:"id"`
	Path    string `json:"path"`
	Digest  string `json:"digest"`
	Content string `json:"content"`
}
type cpgFunction struct {
	ID         string  `json:"id"`
	Definition string  `json:"definition"`
	Span       cpgSpan `json:"span"`
	Signature  struct {
		Span cpgSpan `json:"span"`
	} `json:"signature"`
}
type cpgDocument struct {
	Profile struct {
		ID      string `json:"id"`
		Version string `json:"version"`
	} `json:"profile"`
	Layout   string          `json:"layout"`
	Producer json.RawMessage `json:"producer"`
	Coverage json.RawMessage `json:"coverage"`
	Unit     struct {
		Files       []cpgFile         `json:"files"`
		Functions   []json.RawMessage `json:"functions"`
		Definitions []struct {
			ID       string `json:"id"`
			Identity string `json:"stable_identity"`
		} `json:"definitions"`
	} `json:"unit"`
}
type rstyleReport struct {
	Check *struct {
		Diagnostics []json.RawMessage `json:"diagnostics"`
	} `json:"check"`
	Patterns *struct {
		Candidates []json.RawMessage `json:"candidates"`
	} `json:"patterns"`
	Flow *struct {
		Functions []rstyleFlow `json:"functions"`
	} `json:"flow"`
}
type rstyleFlow struct {
	Function string          `json:"function"`
	Path     string          `json:"path"`
	Flow     json.RawMessage `json:"flow"`
}
type rstyleGraph struct {
	Format      string            `json:"format"`
	Function    json.RawMessage   `json:"function"`
	Files       []cpgFile         `json:"files"`
	Producer    json.RawMessage   `json:"producer"`
	Coverage    json.RawMessage   `json:"coverage"`
	Flow        json.RawMessage   `json:"flow"`
	Diagnostics []json.RawMessage `json:"diagnostics"`
	Patterns    []json.RawMessage `json:"patterns"`
}
type rstyleStore map[string]rstyleGraph

func (r rstyleReport) validate() error {
	if r.Check == nil || r.Patterns == nil || r.Flow == nil {
		return fmt.Errorf("rstyle report requires check, patterns and flow analyses")
	}
	return nil
}

// Function IDs are unit-local: path is part of every lookup.
func (r rstyleReport) flow(path, id string) (json.RawMessage, error) {
	for _, row := range r.Flow.Functions {
		if row.Path == path && row.Function == id {
			return row.Flow, nil
		}
	}
	return nil, fmt.Errorf("rstyle has no flow record for %s in %s", id, path)
}
func (r rstyleReport) diagnostics(path string, line int) ([]json.RawMessage, error) {
	out := []json.RawMessage{}
	for _, raw := range r.Check.Diagnostics {
		var d struct {
			Path string `json:"path"`
			Line int    `json:"line"`
		}
		if err := json.Unmarshal(raw, &d); err != nil {
			return nil, err
		}
		if d.Path == path && d.Line == line {
			out = append(out, raw)
		}
	}
	return out, nil
}
func (r rstyleReport) patterns(identity string) ([]json.RawMessage, error) {
	out := []json.RawMessage{}
	for _, raw := range r.Patterns.Candidates {
		var c struct {
			Sites []struct {
				ID string `json:"id"`
			} `json:"sites"`
		}
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, err
		}
		if candidateHas(c.Sites, identity) {
			out = append(out, raw)
		}
	}
	return out, nil
}
func candidateHas(sites []struct {
	ID string `json:"id"`
}, identity string) bool {
	for _, s := range sites {
		if s.ID == identity {
			return true
		}
	}
	return false
}
func (d cpgDocument) file(id string) (cpgFile, error) {
	for _, f := range d.Unit.Files {
		if f.ID == id {
			return f, nil
		}
	}
	return cpgFile{}, fmt.Errorf("CPG source %q is missing", id)
}
func (d cpgDocument) identity(id string) (string, error) {
	for _, def := range d.Unit.Definitions {
		if def.ID == id {
			return def.Identity, nil
		}
	}
	return "", fmt.Errorf("CPG definition %q is missing", id)
}
func (d cpgDocument) graph(raw json.RawMessage, report rstyleReport) (cpgFunction, rstyleGraph, error) {
	var f cpgFunction
	if err := json.Unmarshal(raw, &f); err != nil {
		return f, rstyleGraph{}, err
	}
	file, err := d.file(f.Span.File)
	if err != nil {
		return f, rstyleGraph{}, err
	}
	if err := validateCPGSpan(f.Signature.Span, file); err != nil {
		return f, rstyleGraph{}, err
	}
	identity, err := d.identity(f.Definition)
	if err != nil {
		return f, rstyleGraph{}, err
	}
	graph := rstyleGraph{Format: "rstyle.cpg-view/1", Function: raw, Files: []cpgFile{file}, Producer: d.Producer, Coverage: d.Coverage}
	graph, err = report.withFindings(graph, f, identity)
	return f, graph, err
}
func (r rstyleReport) withFindings(graph rstyleGraph, f cpgFunction, identity string) (rstyleGraph, error) {
	file := graph.Files[0]
	flow, err := r.flow(file.Path, f.ID)
	if err != nil {
		return rstyleGraph{}, err
	}
	line := strings.Count(file.Content[:f.Signature.Span.Start], "\n") + 1
	diagnostics, err := r.diagnostics(file.Path, line)
	if err != nil {
		return rstyleGraph{}, err
	}
	patterns, err := r.patterns(identity)
	if err != nil {
		return rstyleGraph{}, err
	}
	graph.Flow, graph.Diagnostics, graph.Patterns = flow, diagnostics, patterns
	return graph, nil
}

func validateCPGSpan(span cpgSpan, file cpgFile) error {
	if span.File != file.ID || span.Start < 0 || span.End < span.Start || span.End > len(file.Content) {
		return fmt.Errorf("CPG span is outside source %s", file.Path)
	}
	return nil
}
func sourceKey(path string, start, end int) string {
	return fmt.Sprintf("%s:%d:%d", filepath.ToSlash(path), start, end)
}

// gstyle-allow(side_effect_density): Import boundary decodes recorded JSON and stores validated snapshots; the map is the intended output.
func (store rstyleStore) add(doc cpgDocument, report rstyleReport) error {
	if doc.Profile.ID != "compat.cpg" || doc.Profile.Version != "0.1.0" || doc.Layout != "bundled" {
		return fmt.Errorf("requires bundled compat.cpg/0.1.0 output")
	}
	for _, raw := range doc.Unit.Functions {
		if err := store.addFunction(doc, raw, report); err != nil {
			return err
		}
	}
	return nil
}
func (store rstyleStore) addFunction(doc cpgDocument, raw json.RawMessage, report rstyleReport) error {
	f, graph, err := doc.graph(raw, report)
	if err != nil {
		return err
	}
	file, err := doc.file(f.Span.File)
	if err != nil {
		return err
	}
	if err := validateCPGSpan(f.Span, file); err != nil {
		return err
	}
	key := sourceKey(file.Path, f.Span.Start, f.Span.End)
	if _, exists := store[key]; exists {
		return fmt.Errorf("ambiguous CPG function at %s", key)
	}
	store[key] = graph
	return nil
}
func (store rstyleStore) document(s repoSymbol) ([]byte, error) {
	data, ok := store[sourceKey(s.File, s.Start, s.End)]
	if !ok {
		return nil, fmt.Errorf("rstyle has no matching function snapshot for %s; its source may be outside frontend coverage", s.ID)
	}
	if err := verifyCPGSymbol(data, s); err != nil {
		return nil, err
	}
	return json.Marshal([]rstyleGraph{data})
}
func verifyCPGSymbol(g rstyleGraph, s repoSymbol) error {
	var f cpgFunction
	if err := json.Unmarshal(g.Function, &f); err != nil {
		return err
	}
	for _, file := range g.Files {
		if file.ID == f.Span.File && file.Digest == "sha256:"+s.Digest {
			return nil
		}
	}
	return fmt.Errorf("rstyle source differs from indexed source; restart the viewer")
}
