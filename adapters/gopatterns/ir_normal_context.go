package gopatterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/cfg"
	"slices"
)

const ScalarNormalProfileID = "gstyle.go.scalar-normal"
const ScalarNormalProfileVersion = "0.1.0"

// The matching reference feature counts bindings read on reachable normal flow,
// independently of Go reference-bearing types or the number of read occurrences.
func (b *irBuilder) normalFlowContext(flow *cfg.CFG, operations [][]pdg.Ref, dom *executionDominators) {
	b.normalBindingRoles()
	context := &pdg.NormalFlow{Entry: 1, Blocks: make([]pdg.FlowBlock, len(flow.Blocks))}
	for i, block := range flow.Blocks {
		context.Blocks[i].Operations = operations[i]
		for _, next := range block.Succs {
			context.Blocks[i].Successors = append(context.Blocks[i].Successors, pdg.Ref(next.Index+1))
		}
	}
	reads := map[pdg.Ref]bool{}
	for _, index := range dom.order {
		context.Blocks[index].Reachable = true
		for _, node := range operations[index] {
			b.normalReadBindings(node, reads)
		}
	}
	context.ReadBindings = uint64(len(reads))
	b.normalMiningLabels()
	b.graph.NormalFlow = context
	b.graph.Profile = b.pool.Policy(ScalarNormalProfileID, ScalarNormalProfileVersion)
	b.graph.Function.ReferenceCount = pdg.CountFact{Status: pdg.Known, Value: context.ReadBindings}
	b.normalReturnTransfers(flow, dom)
}

// Unreachable source operations remain available for navigation, but an
// unexecuted return cannot transfer a value to a live function output entity.
func (b *irBuilder) normalReturnTransfers(flow *cfg.CFG, dom *executionDominators) {
	reachable := map[pdg.Ref]bool{}
	for _, index := range dom.order {
		for _, operation := range b.normalOperations(flow.Blocks[index]) {
			reachable[operation] = true
		}
	}

	b.graph.Edges = slices.DeleteFunc(b.graph.Edges, func(edge pdg.Edge) bool {
		return b.pool.Tables.Text(edge.Subkind) == "return-value" && !reachable[edge.Source]
	})
}

func (b *irBuilder) normalReadBindings(node pdg.Ref, seen map[pdg.Ref]bool) {
	if b.graph.Nodes[node-1].Category != pdg.Read {
		return
	}
	for _, read := range b.attributes[b.graph.Nodes[node-1].Attributes-1].Reads {
		role := b.graph.Symbols[read.Symbol-1].Role
		if role == pdg.ParameterRole || role == pdg.LocalRole {
			seen[read.Symbol] = true
		}
	}
}

// A Go receiver is a parameter binding; a named result is a local binding.
// Interface metadata retains receiver/return roles independently of storage.
func (b *irBuilder) normalBindingRoles() {
	for i, symbol := range b.graph.Symbols {
		switch symbol.Role {
		case pdg.OutputRole:
			b.graph.Symbols[i].Role = pdg.LocalRole
		case pdg.ReceiverRole:
			b.graph.Symbols[i].Role = pdg.ParameterRole
		}
	}
}
