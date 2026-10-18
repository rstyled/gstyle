// Package deferflow reads deferred registration and capture facts from the PDG.
// Registration is not invocation. Complete exit, panic and recovery flow remain
// separate obligations for a consumer that follows these sites.
package deferflow

import (
	"encoding/json"
	"fmt"
	"github.com/rstyled/gstyle/domain/pdg"
)

const Extension = "gstyle.go:deferred-call-v1"

// Fact describes required capture edges, not a second dependency graph.
type Fact struct {
	Invocation string `json:"invocation"`
	Slots      []Slot `json:"slots"`
	Spread     bool   `json:"spread"`
	Tuple      bool   `json:"tuple"`
}
type Slot struct {
	Kind     string       `json:"kind"`
	Position pdg.Position `json:"position"`
	Mode     string       `json:"mode"`
}
type Capture struct {
	Slot
	Source pdg.Ref
}
type Site struct {
	Registration, Invocation pdg.Ref
	Captures                 []Capture
	Spread, Tuple            bool
}

// Sites validates references and reads capture producers from canonical edges.
// Its returned slices are owned; the graph remains unchanged.
func Sites(graph *pdg.Graph) ([]Site, error) {
	if graph == nil {
		return nil, nil
	}
	if err := pdg.Validate(graph); err != nil {
		return nil, err
	}
	index := nodeIndex(graph)
	var sites []Site

	for i, node := range graph.Nodes {
		site, found, err := readSite(graph, pdg.Ref(i+1), node, index)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		sites = append(sites, site)
	}
	return sites, validateContext(graph, sites)
}
func nodeIndex(graph *pdg.Graph) map[string]pdg.Ref {
	index := make(map[string]pdg.Ref, len(graph.Nodes))
	for i, node := range graph.Nodes {
		index[graph.Tables.Text(node.ID)] = pdg.Ref(i + 1)
	}
	return index
}
func readSite(graph *pdg.Graph, ref pdg.Ref, node pdg.Node, index map[string]pdg.Ref) (Site, bool, error) {
	text, found, err := siteFactText(graph, node)
	if err != nil || !found {
		return Site{}, found, err
	}
	var fact *Fact
	if err := json.Unmarshal([]byte(text), &fact); err != nil {
		return Site{}, false, fmt.Errorf("decode deferred-call fact: %w", err)
	}
	if fact == nil {
		return Site{}, false, fmt.Errorf("null deferred-call fact")
	}
	site, err := resolveSite(graph, ref, *fact, index)
	return site, true, err
}
func siteFactText(graph *pdg.Graph, node pdg.Node) (string, bool, error) {
	text := ""
	found := false
	for _, extension := range graph.NodeAttributes(node).Extensions {
		if graph.Tables.Text(extension.Name) != Extension {
			continue
		}
		if found {
			return "", false, fmt.Errorf("duplicate deferred-call fact")
		}
		text, found = graph.Tables.Text(extension.Value), true
	}
	if !found && graph.Tables.Text(graph.NodeAttributes(node).Construct) == "defer-register" {
		return "", false, fmt.Errorf("deferred registration is missing its capture fact")
	}
	return text, found, nil
}
func resolveSite(graph *pdg.Graph, ref pdg.Ref, fact Fact, index map[string]pdg.Ref) (Site, error) {
	invocation := index[fact.Invocation]
	if !canonicalInvocation(graph, invocation) {
		return Site{}, fmt.Errorf("deferred invocation is not a canonical call")
	}
	if !canonicalRegistration(graph, ref) {
		return Site{}, fmt.Errorf("deferred registration has the wrong operation kind")
	}
	if fact.Slots == nil {
		return Site{}, fmt.Errorf("deferred capture inventory is missing")
	}
	captures, err := resolveCaptures(graph, ref, fact.Slots)
	if err != nil {
		return Site{}, err
	}
	return Site{Registration: ref, Invocation: invocation, Captures: captures, Spread: fact.Spread, Tuple: fact.Tuple}, nil
}
func canonicalInvocation(graph *pdg.Graph, ref pdg.Ref) bool {
	return ref != 0 && graph.Nodes[ref-1].Category == pdg.Call
}
func canonicalRegistration(graph *pdg.Graph, ref pdg.Ref) bool {
	node := graph.Nodes[ref-1]
	return node.Category == pdg.Other && graph.Tables.Text(graph.NodeAttributes(node).Construct) == "defer-register"
}
func uniqueInvocations(sites []Site) error {
	used := map[pdg.Ref]bool{}
	for _, site := range sites {
		if used[site.Invocation] {
			return fmt.Errorf("deferred invocation belongs to multiple registrations")
		}
		used[site.Invocation] = true
	}
	return nil
}
