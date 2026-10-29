package joern

import (
	"strconv"

	"github.com/rstyled/gstyle/domain/pdg"
)

func (p *projection) bindings(nodes []*vertex) {
	for _, node := range nodes {
		p.declaration(node)
	}
	for _, node := range nodes {
		p.readBinding(node)
	}
	for _, node := range nodes {
		p.bindingAccess(node)
	}
	for i := range p.graph.Function.Inputs {
		p.graph.Function.Inputs[i].Symbol = p.symbols[p.inputs[i]]
	}
}

func (p *projection) declaration(node *vertex) {
	role, exists := map[string]pdg.Role{"LOCAL": pdg.LocalRole, "METHOD_PARAMETER_IN": pdg.ParameterRole}[node.Label]
	if !exists {
		return
	}
	ref := pdg.Ref(len(p.graph.Symbols) + 1)
	p.symbols[node] = ref
	p.graph.Symbols = append(p.graph.Symbols, pdg.Symbol{ID: p.pool.Text("s" + strconv.FormatUint(node.ID, 10)), Name: p.pool.Known(node.text("NAME")), Type: p.pool.Known(node.text("TYPE_FULL_NAME")), Scope: p.graph.Function.ID, Role: role})
	attrs := &p.attrs[p.refs[node]-1]
	attrs.Symbols = []pdg.Ref{ref}
	if node.Label == "METHOD_PARAMETER_IN" {
		p.definition(node, ref)
	}
}

func (p *projection) readBinding(node *vertex) {
	if node.Label == "IDENTIFIER" {
		if symbol := p.referredSymbol(node); symbol != 0 {
			p.attrs[p.refs[node]-1].Reads = []pdg.Access{{Symbol: symbol}}
		}
	}
}

func (p *projection) bindingAccess(node *vertex) {
	if category(node) != pdg.Assignment || len(node.Children) == 0 {
		return
	}
	target := node.Children[0]
	if symbol := p.referredSymbol(target); symbol != 0 {
		p.definition(node, symbol)
		p.attrs[p.refs[node]-1].Writes = []pdg.Access{{Symbol: symbol}}
		if node.text("NAME") == "<operator>.assignment" {
			p.attrs[p.refs[target]-1].Reads = nil
		}
	}
}

func (p *projection) referredSymbol(node *vertex) pdg.Ref {
	if len(node.Out["REF"]) != 1 {
		return 0
	}
	return p.symbols[node.Out["REF"][0]]
}

func (p *projection) definition(node *vertex, symbol pdg.Ref) {
	ref := pdg.Ref(len(p.graph.Definitions) + 1)
	p.graph.Definitions = append(p.graph.Definitions, pdg.Definition{ID: p.pool.Text("d" + strconv.Itoa(int(ref))), Node: p.refs[node], Symbol: symbol})
	p.attrs[p.refs[node]-1].Defines = append(p.attrs[p.refs[node]-1].Defines, ref)
}
