package cpp

import (
	"strconv"

	"github.com/rstyled/gstyle/domain/pdg"
)

func (p *projection) bindings(nodes []*vertex) {
	for _, node := range nodes {
		p.declaration(node)
	}
	p.externalBindings(nodes)
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
	role, exists := declarationRole(node)
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

func declarationRole(node *vertex) (pdg.Role, bool) {
	if node.Label == "LOCAL" && staticStorage(node) {
		return pdg.GlobalRole, true
	}
	if node.Label == "LOCAL" && node.text("CLOSURE_BINDING_ID") != "" {
		return pdg.CaptureRole, true
	}
	role, exists := map[string]pdg.Role{"LOCAL": pdg.LocalRole, "METHOD_PARAMETER_IN": pdg.ParameterRole}[node.Label]
	return role, exists
}

func (p *projection) readBinding(node *vertex) {
	if node.Label != "IDENTIFIER" {
		return
	}
	access := pdg.Access{Symbol: p.readSymbol(node)}
	if access.Symbol == 0 {
		access.UnresolvedReason = p.pool.Text("CPG identifier does not have one imported declaration binding")
	}
	p.attrs[p.refs[node]-1].Reads = []pdg.Access{access}
}

func (p *projection) bindingAccess(node *vertex) {
	if category(node) != pdg.Assignment || len(node.Children) == 0 {
		return
	}
	target := node.Children[0]
	if target.Label == "IDENTIFIER" && node.text("NAME") == "<operator>.assignment" {
		p.attrs[p.refs[target]-1].Reads = nil
	}
	if symbol := p.referredSymbol(target); symbol != 0 {
		p.definition(node, symbol)
		p.attrs[p.refs[node]-1].Writes = []pdg.Access{{Symbol: symbol}}
		return
	}
	p.attrs[p.refs[node]-1].Writes = []pdg.Access{{UnresolvedReason: p.pool.Text("CPG write requires unresolved indirect storage or a missing declaration binding")}}
}

func (p *projection) referredSymbol(node *vertex) pdg.Ref {
	if len(node.Out["REF"]) == 1 {
		return p.symbols[node.Out["REF"][0]]
	}
	return 0
}

func (p *projection) definition(node *vertex, symbol pdg.Ref) {
	ref := pdg.Ref(len(p.graph.Definitions) + 1)
	p.graph.Definitions = append(p.graph.Definitions, pdg.Definition{ID: p.pool.Text("d" + strconv.Itoa(int(ref))), Node: p.refs[node], Symbol: symbol})
	p.attrs[p.refs[node]-1].Defines = append(p.attrs[p.refs[node]-1].Defines, ref)
}

func (p *projection) reachingIdentity(edge *pdg.Edge, variable string) {
	if p.pool.Tables.Text(edge.Subkind) != "REACHING_DEF" {
		return
	}
	for _, ref := range p.attrs[edge.Source-1].Defines {
		definition := p.graph.Definitions[ref-1]
		symbol := p.graph.Symbols[definition.Symbol-1]
		if p.pool.Tables.Text(symbol.Name.Value) != variable {
			continue
		}
		edge.Symbol, edge.Definition = definition.Symbol, ref
		return
	}
}
