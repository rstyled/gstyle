package cpp

import "github.com/rstyled/gstyle/domain/pdg"

const referencePolicy = "Distinct REF-bound local and parameter bindings with recorded read uses in this method, including unreachable source uses; repeated reads count once and shadowed bindings separately. Captures and global/member storage are excluded. This counts imported bindings, not C++ ownership or aliases."

func (p *projection) referenceCount(nodes []*vertex) {
	bindings := map[pdg.Ref]bool{}
	complete := true
	for _, node := range nodes {
		for _, read := range p.attrs[p.refs[node]-1].Reads {
			complete = p.referenceRead(node, read, bindings) && complete
		}
	}
	p.extension("gstyle.cpp:reference-count-policy", referencePolicy)
	if !complete {
		p.graph.Function.ReferenceCount = pdg.CountFact{Status: pdg.Unknown, Reason: p.pool.Text("unresolved identifier declarations prevent a complete binding count; see cpp.unresolved-binding diagnostics")}
		return
	}
	p.graph.Function.ReferenceCount = pdg.CountFact{Status: pdg.Known, Value: uint64(len(bindings))}
}

func (p *projection) referenceRead(node *vertex, read pdg.Access, bindings map[pdg.Ref]bool) bool {
	if read.Symbol == 0 {
		p.diagnostic(node, "cpp.unresolved-binding", "reference_counting", p.pool.Tables.Text(read.UnresolvedReason))
		return false
	}
	role := p.graph.Symbols[read.Symbol-1].Role
	if role == pdg.LocalRole || role == pdg.ParameterRole {
		bindings[read.Symbol] = true
	}
	return true
}
