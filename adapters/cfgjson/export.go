// Package cfgjson writes the paired normal-flow CFG without changing the PDG.
package cfgjson

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/rstyled/gstyle/domain/pdg"
)

type object = map[string]any
type exporter struct {
	g            *pdg.Graph
	source       []byte
	nodes, edges []object
}

func Write(w io.Writer, g *pdg.Graph, source []byte) error {
	if err := pdg.Validate(g); err != nil {
		return err
	}
	if g.NormalFlow == nil {
		return fmt.Errorf("paired CFG is missing")
	}
	if err := validatedSource(g, source); err != nil {
		return err
	}
	x := exporter{g: g, source: source, nodes: []object{}, edges: []object{}}
	if err := x.operations(); err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(x.document())
}

func validatedSource(g *pdg.Graph, source []byte) error {
	profile := g.Tables.Policies[g.Profile-1]
	if g.Tables.Text(profile.ID) != "gstyle.go.scalar-normal" || g.Tables.Text(profile.Version) != "0.2.0" {
		return fmt.Errorf("CFG requires the registered ordered scalar extraction profile")
	}
	identity := g.Sources[0].ContentIdentity
	if identity.Status != pdg.Known || g.Tables.Text(identity.Value) != fmt.Sprintf("%x", sha256.Sum256(source)) {
		return fmt.Errorf("CFG source bytes do not match compiler snapshot")
	}
	if g.Producer.LanguageVersion.Status != pdg.Known {
		return fmt.Errorf("CFG requires known compiler language evidence")
	}
	return nil
}

func (x *exporter) span(span pdg.Span) object {
	return object{"source_id": x.g.Tables.Text(x.g.Sources[span.Source-1].ID), "start": span.Start, "end": span.End}
}

func known(value string) object { return object{"status": "known", "value": value} }

func (x *exporter) document() object {
	profile := x.g.Tables.Policies[x.g.Profile-1]
	p := x.g.Producer
	source := x.g.Sources[0]
	x.nodes = append(x.nodes, object{"id": "cfg-entry", "kind": "entry", "origin": "Administrative normal-flow entry; no value semantics.", "operations": []object{}}, object{"id": "cfg-exit", "kind": "exit", "exit_kind": "normal", "origin": "Administrative normal termination; panic/unwinding omitted.", "operations": []object{}})
	x.transfer("cfg-entry", x.first(x.g.NormalFlow.Entry), "normal")
	return object{
		"schema_version": "0.2.0",
		"profile":        object{"name": x.g.Tables.Text(profile.ID), "version": x.g.Tables.Text(profile.Version), "granularity": "operation", "evaluation_order": object{"sequencing": "language_defined", "short_circuit": "explicit_branches", "unspecified_order": "reject"}, "exceptional_flow": "omitted", "exit_policy": "Explicit source returns and normal fallthrough terminate at one normal exit. No calls, loops, or divergence are admitted.", "unreachable_code": "preserve"},
		"producer":       object{"name": x.g.Tables.Text(p.Name), "version": x.g.Tables.Text(p.Version), "language": "go", "language_version": known(x.g.Tables.Text(p.LanguageVersion.Value)), "configuration": json.RawMessage(x.g.Tables.Text(p.Configuration))},
		"sources":        []object{{"id": x.g.Tables.Text(source.ID), "path": x.g.Tables.Text(source.Path), "content_identity": known(x.g.Tables.Text(source.ContentIdentity.Value))}},
		"function":       object{"id": x.g.Tables.Text(x.g.Function.ID), "name": known(x.g.Tables.Text(x.g.Function.Name.Value)), "source_span": x.span(x.g.Function.Span)},
		"entry_node_id":  "cfg-entry", "exit_node_ids": []string{"cfg-exit"}, "nodes": x.nodes, "edges": x.edges,
		"extraction_evidence": object{"status": "partial", "limitations": []string{"Panic/unwinding flow is omitted under the bounded normal-flow extraction policy."}},
	}
}

func (x *exporter) operations() error {
	for _, block := range x.g.NormalFlow.Blocks {
		for i, ref := range block.Operations {
			if err := x.operation(ref, block, i); err != nil {
				return err
			}
		}
	}
	return nil
}

func (x *exporter) operation(ref pdg.Ref, block pdg.FlowBlock, index int) error {
	node := x.g.Nodes[ref-1]
	span, original, err := x.operationSource(node)
	if err != nil {
		return err
	}
	kind := cfgOperationKind(node.Category)
	id := x.g.Tables.Text(node.ID)
	x.nodes = append(x.nodes, object{"id": "cfg-" + id, "kind": kind, "origin": "Typed Go source operation, independently checked against the scalar source inventory.", "operations": []object{{"id": id, "original_kind": original, "text": string(x.source[span.Start:span.End]), "source_span": x.span(span)}}})
	x.successors(ref, block, index)
	return nil
}

func (x *exporter) operationSource(node pdg.Node) (pdg.Span, string, error) {
	if node.Span == 0 {
		return pdg.Span{}, "", fmt.Errorf("CFG operation requires a source span")
	}
	span := x.g.Spans[node.Span-1]
	if uint64(span.End) > uint64(len(x.source)) {
		return pdg.Span{}, "", fmt.Errorf("CFG operation span exceeds source bytes")
	}
	original := x.g.Tables.Text(node.OriginalKind.Value)
	if !node.Synthetic && node.OriginalKind.Status != pdg.Known {
		return pdg.Span{}, "", fmt.Errorf("CFG operation requires a known frontend kind")
	}
	if node.Synthetic {
		original = "gstyle.go.scalar-normal:lowered-" + strconv.Itoa(int(node.Category))
	}

	return span, original, nil
}

func cfgOperationKind(category pdg.Category) string {
	switch category {
	case pdg.Control:
		return "condition"
	case pdg.Return:
		return "transfer"
	default:
		return "block"
	}
}

func (x *exporter) successors(ref pdg.Ref, block pdg.FlowBlock, index int) {
	from := "cfg-" + x.g.Tables.Text(x.g.Nodes[ref-1].ID)
	if index+1 < len(block.Operations) {
		x.transfer(from, []string{"cfg-" + x.g.Tables.Text(x.g.Nodes[block.Operations[index+1]-1].ID)}, "normal")
		return
	}
	if len(block.Successors) == 0 {
		x.termination(from, ref)
		return
	}
	for i, next := range block.Successors {
		kind := "normal"
		if len(block.Successors) == 2 {
			kind = []string{"true", "false"}[i]
		}
		x.transfer(from, x.first(next), kind)
	}
}

func (x *exporter) termination(from string, ref pdg.Ref) {
	kind := "normal"
	if x.g.Nodes[ref-1].Category == pdg.Return {
		kind = "return"
	}
	x.transfer(from, []string{"cfg-exit"}, kind)
}

func (x *exporter) first(start pdg.Ref) []string {
	queue := []pdg.Ref{start}
	seen := map[pdg.Ref]bool{}
	var ids []string
	for i := 0; i < len(queue); i++ {
		ref := queue[i]
		if seen[ref] {
			continue
		}
		seen[ref] = true
		block := x.g.NormalFlow.Blocks[ref-1]
		if len(block.Operations) > 0 {
			ids = append(ids, "cfg-"+x.g.Tables.Text(x.g.Nodes[block.Operations[0]-1].ID))
			continue
		}
		if len(block.Successors) == 0 {
			ids = append(ids, "cfg-exit")
			continue
		}
		queue = append(queue, block.Successors...)
	}
	return ids
}

func (x *exporter) transfer(from string, targets []string, kind string) {
	for _, target := range targets {
		x.edges = append(x.edges, object{"id": "cfge-" + strconv.Itoa(len(x.edges)+1), "source": from, "target": target, "kind": kind})
	}
}
