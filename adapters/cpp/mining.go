package cpp

import (
	"slices"
	"strings"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/quality"
)

func (p *projection) miningLabels(nodes []*vertex) {
	for _, node := range nodes {
		kind := miningKind(node)
		if kind == "" {
			continue
		}
		p.graph.Nodes[p.refs[node]-1].MatchLabel = p.pool.MatchLabel(p.miningLabel(node, kind))
	}
}

func (p *projection) miningLabel(node *vertex, kind patterns.NodeKind) pdg.MatchLabel {
	attrs := p.attrs[p.refs[node]-1]
	detail := miningDetail(node)
	if fieldOperation(node) {
		kind, detail = patterns.Field, memberName(node)
	}
	label := pdg.MatchLabel{Kind: p.pool.Text(string(kind)), TypeClass: p.pool.Text(cppTypeClass(p.expressionType(node))), Detail: p.pool.Text(detail)}
	if category(node) == pdg.Call {
		var callee pdg.Text
		if attrs.Callee.Status == pdg.Known {
			callee = attrs.Callee.Value
		}
		return pdg.MatchLabel{Kind: label.Kind, TypeClass: label.TypeClass, Detail: label.Detail, SignatureClass: p.pool.Text(callSignature(node)), Effects: patterns.EffectUnknown, Callee: callee}
	}
	if node.Label == "LITERAL" {
		return pdg.MatchLabel{Kind: p.pool.Text(string(patterns.Lit)), TypeClass: label.TypeClass, LiteralKind: p.pool.Text(cppTypeClass(node.text("TYPE_FULL_NAME"))), Detail: p.pool.Text(node.text("CODE"))}
	}
	return label
}

func (p *projection) miningFacts(nodes []*vertex, function Function) *patterns.MiningFacts {
	fact := &patterns.MiningFacts{
		ID: p.graph.Tables.Text(p.graph.Function.ID), Name: function.Score.Name, Path: function.Path,
		Line: function.Score.Line, EndLine: function.Score.EndLine, Canonical: p.graph,
		GuardClauses: function.source.guards, ConstantBoolReturns: function.source.booleans,
		AstTypes: map[string]int{}, EffectKinds: qualityEffectKinds(function.Quality),
		Mutates: slices.ContainsFunc(function.Quality.Mutations, func(mutation quality.Mutation) bool { return mutation.Provenance != quality.Local }),
	}
	var args []string
	for _, input := range p.inputs {
		args = append(args, cppTypeClass(input.text("TYPE_FULL_NAME")))
		if receiverParameter(input) {
			fact.SelfTy = input.text("TYPE_FULL_NAME")
		} else if scalarType(input.text("TYPE_FULL_NAME")) {
			fact.Params = append(fact.Params, patterns.ParamInfo{Name: input.text("NAME"), Type: input.text("TYPE_FULL_NAME")})
		}
	}
	fact.SigKey = "fn(" + strings.Join(args, ",") + ") -> " + cppTypeClass(methodResultType(p.method))
	for _, node := range nodes {
		fact.AstTypes[node.Label]++
	}
	p.generalFacts(fact, nodes)
	return fact
}

func cppTypeClass(name string) string {
	name = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(name), "const "))
	if !concreteName(name) {
		return "?"
	}
	if strings.HasSuffix(name, "*") || strings.HasSuffix(name, "&") {
		return name[len(name)-1:] + cppTypeClass(name[:len(name)-1])
	}
	if scalarType(name) || standardType(name) {
		return name
	}
	return "_"
}

func standardType(name string) bool {
	return strings.HasPrefix(name, "std.") || strings.HasPrefix(name, "std::")
}

func miningKind(node *vertex) patterns.NodeKind {
	if node.Label == "CONTROL_STRUCTURE" {
		return controlMiningKind(node.text("CONTROL_STRUCTURE_TYPE"))
	}
	return map[pdg.Category]patterns.NodeKind{pdg.FormalInput: patterns.Param, pdg.Declaration: patterns.Let, pdg.Read: patterns.Op, pdg.Assignment: patterns.Op, pdg.Computation: patterns.Op, pdg.Call: patterns.Call, pdg.Return: patterns.Return}[category(node)]
}

func miningDetail(node *vertex) string {
	if category(node) == pdg.Read {
		return "read"
	}
	comparisons := map[string]string{"<operator>.equals": "==", "<operator>.notEquals": "!=", "<operator>.lessThan": "<", "<operator>.lessEqualsThan": "<=", "<operator>.greaterThan": ">", "<operator>.greaterEqualsThan": ">="}
	if op := comparisons[node.text("NAME")]; op != "" {
		return "cmp:" + op
	}
	return node.text("NAME")
}

func controlMiningKind(kind string) patterns.NodeKind {
	switch kind {
	case "FOR", "WHILE", "DO":
		return patterns.Loop
	case "SWITCH":
		return patterns.Match
	case "CATCH":
		return patterns.Try
	default:
		return patterns.Branch
	}
}

func callSignature(node *vertex) string {
	var args []string
	for _, child := range node.Children {
		if child.number("ARGUMENT_INDEX") >= 0 {
			args = append(args, cppTypeClass(child.text("TYPE_FULL_NAME")))
		}
	}
	return "fn(" + strings.Join(args, ",") + ") -> " + cppTypeClass(node.text("TYPE_FULL_NAME"))
}

func qualityEffectKinds(function quality.Function) []string {
	var effects []string
	for _, effect := range function.Effects {
		effects = append(effects, string(effect.Kind))
	}
	if len(function.Calls) != 0 {
		effects = append(effects, string(quality.UnknownEffect))
	}
	return effects
}

func methodResultType(method *vertex) string {
	for _, node := range method.Children {
		if node.Label == "METHOD_RETURN" {
			return node.text("TYPE_FULL_NAME")
		}
	}
	return "ANY"
}
