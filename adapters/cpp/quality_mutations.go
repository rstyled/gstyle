package cpp

import (
	"strconv"
	"strings"

	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/quality"
)

func localInitializer(node *vertex) bool {
	if node.text("NAME") != "<operator>.assignment" {
		return false
	}
	declaration := assignedLocal(node)
	return declaration != nil && declaration.Parent == node.Parent &&
		declaration.number("ORDER") < node.number("ORDER") && sameDeclarator(node, declaration)
}

func assignedLocal(node *vertex) *vertex {
	target := argument(node, 1)
	if target == nil || target.Label != "IDENTIFIER" || len(target.Out["REF"]) != 1 {
		return nil
	}
	declaration := target.Out["REF"][0]
	role, ok := declarationRole(declaration)
	if !ok || role != pdg.LocalRole {
		return nil
	}
	return declaration
}

func sameDeclarator(node, declaration *vertex) bool {
	start, end := node.number("OFFSET"), node.number("OFFSET_END")
	return start >= 0 && end > start && declaration.number("OFFSET") == start && declaration.number("OFFSET_END") == end
}

func (p *projection) qualityMutation(node *vertex) quality.Mutation {
	target := argument(node, 1)
	mutation := quality.Mutation{Site: int(p.refs[node]), Root: "indirect C++ storage", RootID: strconv.FormatUint(node.ID, 10), Line: max(1, node.number("LINE_NUMBER")), Provenance: quality.Unknown}
	if field := p.doc.staticMember(target); field != nil {
		mutation.Root, mutation.RootID, mutation.FieldPath, mutation.Provenance = field.Parent.text("FULL_NAME"), strconv.FormatUint(field.ID, 10), field.text("NAME"), quality.External
		return mutation
	}
	root := storageRoot(target)
	if root == nil {
		return mutation
	}
	symbol := p.referredSymbol(root)
	if symbol == 0 {
		return mutation
	}
	binding := p.graph.Symbols[symbol-1]
	mutation.Root, mutation.RootID = p.pool.Tables.Text(binding.Name.Value), p.pool.Tables.Text(binding.ID)
	mutation.FieldPath = memberName(target)
	mutation.Provenance = mutationProvenance(binding, target == root, p.pool.Tables.Text(binding.Type.Value))
	if mutation.Provenance == quality.External && len(root.Out["REF"]) == 1 {
		mutation.Parameter = p.parameterIndex(root.Out["REF"][0])
	}
	return mutation
}

func storageRoot(node *vertex) *vertex {
	for node != nil {
		if node.Label == "IDENTIFIER" {
			return node
		}
		switch node.text("NAME") {
		case "<operator>.indirection", "<operator>.fieldAccess", "<operator>.indirectFieldAccess", "<operator>.indexAccess", "<operator>.indirectIndexAccess", "<operator>.addressOf":
			node = argument(node, 1)
		default:
			return nil
		}
	}
	return nil
}

func mutationProvenance(binding pdg.Symbol, direct bool, typ string) quality.Provenance {
	switch binding.Role {
	case pdg.GlobalRole, pdg.CaptureRole:
		return quality.External
	}
	if direct && localValueType(typ) {
		return quality.Local
	}
	if binding.Role == pdg.ParameterRole && strings.ContainsAny(typ, "*&") {
		return quality.External
	}
	return quality.Unknown
}

func localValueType(typ string) bool {
	return scalarType(typ) || strings.Contains(typ, "*") && !strings.Contains(typ, "&")
}
