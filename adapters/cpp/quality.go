package cpp

import (
	"slices"
	"strings"

	"github.com/rstyled/gstyle/domain"
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/quality"
)

func DefaultQualityConfig() quality.Config {
	config := quality.DefaultConfig()
	config.Prefixes = cppEffectPrefixes()
	return config
}

func qualityName(name string) string { return "cpp::" + name }

func (p *projection) qualityFunction(nodes []*vertex, score domain.Function) quality.Function {
	fact := quality.Function{Location: quality.Location{Path: p.path(), Line: score.Line, Column: score.Column, Name: qualityName(score.Name)}, Source: score.Source, CodeLines: codeLines(score.Source)}
	for _, input := range p.inputs {
		if receiverParameter(input) {
			fact.HasSelf = true
			continue
		}
		fact.ParamList = append(fact.ParamList, quality.ParamFacts{Name: input.text("NAME"), Type: input.text("TYPE_FULL_NAME"), Uses: referenceUses(p.doc.Referrers[input], p.method)})
	}
	fact.Params = len(fact.ParamList)
	if !methodBody(p.method) {
		fact.Effects = append(fact.Effects, quality.Effect{Kind: quality.UnknownEffect, Line: score.Line, Detail: "C++ declaration has no body"})
	}
	for _, node := range nodes {
		p.qualityNode(&fact, node)
	}
	return fact
}

func methodBody(method *vertex) bool {
	return slices.ContainsFunc(method.Children, func(node *vertex) bool {
		return node.Label == "BLOCK" && node.number("LINE_NUMBER") > 0 && node.text("CODE") != "<empty>"
	})
}

func referenceUses(nodes []*vertex, method *vertex) int {
	count := 0
	for _, node := range nodes {
		if node.Label == "IDENTIFIER" && node.Owner == method {
			count++
		}
	}
	return count
}

func receiverParameter(node *vertex) bool {
	return node.text("NAME") == "this" && node.number("INDEX") == 0
}

func (p *projection) qualityNode(fact *quality.Function, node *vertex) {
	if node.Label == "BLOCK" {
		fact.Statements += statementCount(node)
	}
	if category(node) == pdg.Call {
		fact.Calls = append(fact.Calls, p.qualityCall(node))
	}
	if category(node) == pdg.Assignment && !localInitializer(node) {
		fact.Mutations = append(fact.Mutations, p.qualityMutation(node))
	}
	if node.text("CONTROL_STRUCTURE_TYPE") == "THROW" {
		fact.Effects = append(fact.Effects, quality.Effect{Site: int(p.refs[node]), Kind: quality.Panic, Line: max(1, node.number("LINE_NUMBER")), Detail: "C++ throw"})
	}
	p.qualityBoundary(fact, node)
}

func (p *projection) qualityCall(node *vertex) quality.Call {
	attrs := p.attrs[p.refs[node]-1]
	name := node.text("CODE")
	if attrs.Callee.Status == pdg.Known {
		name = p.pool.Tables.Text(attrs.Callee.Value)
	} else if modeled := p.modeledCtype(node); modeled != "" {
		name = modeled + ":int(int)"
	}
	call := quality.Call{Site: int(p.refs[node]), Callee: qualityName(name), Line: max(1, node.number("LINE_NUMBER")), Dynamic: attrs.Callee.Status != pdg.Known}
	p.qualityArguments(&call, node)
	return call
}

func (p *projection) qualityBoundary(fact *quality.Function, node *vertex) {
	p.qualityFieldEffect(fact, node)
	attrs := p.attrs[p.refs[node]-1]
	for _, read := range attrs.Reads {
		if read.Symbol == 0 {
			fact.Effects = append(fact.Effects, quality.Effect{Site: int(p.refs[node]), Kind: quality.UnknownEffect, Line: max(1, node.number("LINE_NUMBER")), Detail: "unbound C++ read"})
		}
		if read.Symbol != 0 && p.graph.Symbols[read.Symbol-1].Role == pdg.GlobalRole {
			fact.Effects = append(fact.Effects, quality.Effect{Site: int(p.refs[node]), Kind: quality.GlobalRead, Line: max(1, node.number("LINE_NUMBER")), Detail: p.pool.Tables.Text(p.graph.Symbols[read.Symbol-1].Name.Value)})
		}
	}
	if !p.uncertainOperation(node) {
		return
	}
	fact.Effects = append(fact.Effects, quality.Effect{Site: int(p.refs[node]), Kind: quality.UnknownEffect, Line: max(1, node.number("LINE_NUMBER")), Detail: uncertainOperationDetail(node)})
}

func (p *projection) uncertainOperation(node *vertex) bool {
	if p.typeQualifierLocal(node) {
		return false
	}
	if opaqueObject(node) || p.opaqueOperator(node) {
		return true
	}
	for _, declaration := range node.Out["REF"] {
		if strings.Contains(declaration.text("CODE"), "volatile") {
			return true
		}
	}
	return false
}

func opaqueObject(node *vertex) bool {
	switch node.Label {
	case "UNKNOWN":
		return true
	case "RETURN":
		return node.Owner != nil && objectValueType(methodResultType(node.Owner))
	case "LOCAL", "METHOD_PARAMETER_IN":
		return !strings.HasPrefix(node.text("CODE"), "<unknown>") && objectValueType(node.text("TYPE_FULL_NAME"))
	default:
		return false
	}
}

func uncertainOperationDetail(node *vertex) string {
	switch node.Label {
	case "LOCAL", "METHOD_PARAMETER_IN":
		return "unresolved C++ object lifetime: " + node.text("NAME")
	case "RETURN":
		return "unresolved C++ return copy or move"
	case "UNKNOWN":
		return "unsupported C++ source: " + node.text("CODE")
	default:
		return "unresolved C++ operation: " + node.text("CODE")
	}
}

func objectValueType(typ string) bool {
	if strings.ContainsAny(typ, "*&") {
		return false
	}
	base, _, _ := strings.Cut(typ, "[")
	return !scalarType(base)
}

func (p *projection) expressionType(node *vertex) string {
	if typ, exists := p.types[node]; exists {
		return typ
	}
	typ := p.inferredExpressionType(node)
	p.types[node] = typ
	return typ
}

func (p *projection) inferredExpressionType(node *vertex) string {
	if typ := node.text("TYPE_FULL_NAME"); concreteName(typ) {
		return typ
	}
	if typ := p.fieldResultType(node); concreteName(typ) {
		return typ
	}
	if len(node.Out["REF"]) == 1 {
		if typ := node.Out["REF"][0].text("TYPE_FULL_NAME"); concreteName(typ) {
			return typ
		}
	}
	return p.operationResultType(node)
}

func (p *projection) opaqueOperator(node *vertex) bool {
	if !valueOperator(node) || localInitializer(node) {
		return false
	}
	for _, child := range node.Children {
		if nonvalueOperand(child) {
			continue
		}
		if !operationValueType(p.expressionType(child)) {
			return true
		}
	}
	return false
}

func valueOperator(node *vertex) bool {
	return operatorCall(node) && category(node) != pdg.Call && node.text("NAME") != "<operator>.fieldAccess"
}
func nonvalueOperand(node *vertex) bool {
	return node.Label == "FIELD_IDENTIFIER" || node.Label == "TYPE_REF"
}
func operationValueType(typ string) bool {
	return typ == "std::size_t" || localValueType(strings.TrimRight(typ, "& ")) || strings.Contains(typ, "[")
}

func scalarType(name string) bool {
	name = strings.TrimSpace(name)
	if strings.ContainsAny(name, "*&[") {
		return false
	}
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return false
	}
	return !slices.ContainsFunc(parts, func(part string) bool {
		return !slices.Contains(strings.Fields("const volatile signed unsigned short long bool char wchar_t char8_t char16_t char32_t int float double void _Bool"), part)
	})
}

func (e *Extraction) QualityFacts() []quality.Function {
	facts := make([]quality.Function, len(e.Functions))
	counts := map[string]int{}
	for i, function := range e.Functions {
		facts[i] = function.Quality
		facts[i].DDD = qualityDDD(function.mining, facts[i].Line)
		if function.body {
			counts[facts[i].Name]++
		}
	}
	for i := range facts {
		facts[i].Calls = sourceQualityCalls(facts[i].Calls, counts)
	}
	catalog := e.qualityHelpers(facts, counts)
	for i := range facts {
		facts[i].Helpers = quality.ReachableHelpers(facts[i].Name, facts[i].Calls, func(name string) (quality.Helper, bool) { helper, ok := catalog[name]; return helper, ok })
	}
	return facts
}

func (e *Extraction) qualityHelpers(facts []quality.Function, counts map[string]int) map[string]quality.Helper {
	catalog := map[string]quality.Helper{}
	for i, f := range facts {
		if e.Functions[i].body && counts[f.Name] == 1 {
			catalog[f.Name] = quality.Helper{Name: f.Name, Params: f.Params, HasSelf: f.HasSelf, Mutations: f.Mutations, Calls: f.Calls, Effects: f.Effects}
		}
	}
	return catalog
}

func sourceQualityCalls(calls []quality.Call, counts map[string]int) []quality.Call {
	result := slices.Clone(calls)
	for i := range result {
		result[i].Local = !result[i].Dynamic && counts[result[i].Callee] == 1
	}
	return result
}
