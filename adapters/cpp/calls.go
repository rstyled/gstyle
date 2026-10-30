package cpp

import (
	"slices"
	"strconv"
	"strings"

	"github.com/rstyled/gstyle/domain/pdg"
)

type callTarget struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	External bool   `json:"external"`
}

func (p *projection) callFacts(nodes []*vertex) {
	for _, node := range nodes {
		if category(node) != pdg.Call {
			continue
		}
		p.callFact(node)
	}
}

func (p *projection) callFact(node *vertex) {
	attrs := &p.attrs[p.refs[node]-1]
	targets := p.doc.callTargets(node)
	attrs.CallForm = importedCallForm(node)
	attrs.Callee = p.callCallee(node, targets)
	attrs.Extensions = append(attrs.Extensions, p.jsonExtension("gstyle.cpp:call-targets", targets))
	if attrs.Callee.Status == pdg.Unknown {
		p.diagnostic(node, "cpp.unresolved-call", "calls", p.pool.Tables.Text(attrs.Callee.Value))
	}
}

func (p *projection) callCallee(node *vertex, targets []callTarget) pdg.Fact {
	if importedCallForm(node) == pdg.DirectCall {
		return p.staticCallee(node, targets)
	}
	if target := p.doc.indirectTarget(node); target != nil && sameTarget(targets, target.text("FULL_NAME")) {
		return p.pool.Known(target.text("FULL_NAME"))
	}
	return p.pool.Missing(pdg.Unknown, "indirect call has no proven unique runtime target; recorded candidates are retained")
}

func (p *projection) staticCallee(node *vertex, targets []callTarget) pdg.Fact {
	names, complete := concreteTargets(targets)
	if !complete {
		return p.pool.Missing(pdg.Unknown, "recorded call targets include unresolved names or signatures")
	}
	if len(names) == 1 {
		return p.pool.Known(names[0])
	}
	if len(names) > 1 {
		return p.pool.Missing(pdg.Unknown, "static call has multiple distinct recorded targets")
	}
	if name := node.text("METHOD_FULL_NAME"); concreteName(name) {
		return p.pool.Known(name)
	}
	return p.pool.Missing(pdg.Unknown, "producer does not resolve the call name or signature; original properties and targets are retained")
}

func (p *projection) diagnostic(node *vertex, code, capability, message string) {
	p.graph.Diagnostics = append(p.graph.Diagnostics, pdg.Diagnostic{Code: p.pool.Text(code), Capability: p.pool.Text(capability), Message: p.pool.Text(message), Node: p.refs[node], Severity: pdg.Warning})
}

func (d *document) callTargets(node *vertex) []callTarget {
	methods := d.targetMethods(node)
	seen := map[uint64]bool{}
	var result []callTarget
	for _, method := range methods {
		if method.Label != "METHOD" || seen[method.ID] {
			continue
		}
		seen[method.ID] = true
		result = append(result, callTarget{ID: strconv.FormatUint(method.ID, 10), Name: method.text("FULL_NAME"), External: method.flag("IS_EXTERNAL")})
	}
	return result
}

func (d *document) targetMethods(node *vertex) []*vertex {
	methods := append(slices.Clone(node.Out["CALL"]), node.Out["REF"]...)
	if len(methods) != 0 {
		return methods
	}
	if importedCallForm(node) == pdg.DirectCall {
		return d.Callables[node.text("METHOD_FULL_NAME")]
	}
	if target := d.indirectTarget(node); target != nil {
		return []*vertex{target}
	}
	return nil
}

func (d *document) lambdaTarget(node *vertex) *vertex {
	receivers := node.Out["RECEIVER"]
	if importedCallForm(node) != pdg.IndirectCall || len(receivers) != 1 {
		return nil
	}
	methods := d.Callables[receivers[0].text("TYPE_FULL_NAME")]
	if len(methods) != 1 {
		return nil
	}
	method := methods[0]
	if !strings.HasPrefix(method.text("NAME"), "<lambda>") || !sameCallSignature(node, method) {
		return nil
	}
	return method
}

func macroCall(node *vertex) bool {
	return node.Label == "CALL" && node.text("DISPATCH_TYPE") == "INLINED"
}

func runtimeCall(node *vertex) bool {
	return node.Label == "CALL" && (node.text("DISPATCH_TYPE") == "DYNAMIC_DISPATCH" || node.text("NAME") == "<operator>.pointerCall")
}

func operatorCall(node *vertex) bool {
	return node.Label == "CALL" && strings.HasPrefix(node.text("NAME"), "<operator>.")
}

func concreteName(name string) bool {
	return name != "" && name != "ANY" && !strings.Contains(name, "<unresolved") && !strings.Contains(name, "<unknown>") && !strings.HasPrefix(name, "<operator>()")
}

func importedCallForm(node *vertex) pdg.CallForm {
	if node.text("NAME") == "<operator>.pointerCall" {
		return pdg.IndirectCall
	}
	switch node.text("DISPATCH_TYPE") {
	case "STATIC_DISPATCH":
		return pdg.DirectCall
	case "DYNAMIC_DISPATCH":
		return pdg.IndirectCall
	default:
		return pdg.UnknownCall
	}
}

func sameTarget(targets []callTarget, name string) bool {
	for _, target := range targets {
		if target.Name != name {
			return false
		}
	}
	return true
}

func concreteTargets(targets []callTarget) ([]string, bool) {
	var names []string
	for _, target := range targets {
		if !concreteName(target.Name) {
			return nil, false
		}
		if !slices.Contains(names, target.Name) {
			names = append(names, target.Name)
		}
	}
	return names, true
}

func sameCallSignature(call, method *vertex) bool {
	signature := call.text("SIGNATURE")
	return concreteName(signature) && signature == method.text("SIGNATURE")
}
