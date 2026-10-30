package cpp

import (
	"slices"
	"strings"

	"github.com/rstyled/gstyle/domain/quality"
)

func (p *projection) typeQualifierLocal(node *vertex) bool {
	if node.Label != "LOCAL" || node.text("NAME") == "" || node.text("CODE") != node.text("NAME") {
		return false
	}
	uses := p.doc.Referrers[node]
	return len(uses) > 0 && !slices.ContainsFunc(uses, func(use *vertex) bool { return !typeQualifierUse(use) })
}

func typeQualifierUse(node *vertex) bool {
	parent := node.Parent
	if parent == nil || !fieldOperation(parent) || argument(parent, 1) != node {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(strings.TrimPrefix(parent.text("CODE"), node.text("CODE"))), "::")
}

func (p *projection) qualityFieldEffect(fact *quality.Function, node *vertex) {
	if !fieldOperation(node) {
		return
	}
	record, _ := p.doc.accessedRecord(node)
	if record == nil || record.fields[memberName(node)] == nil {
		fact.Effects = append(fact.Effects, quality.Effect{Site: int(p.refs[node]), Line: max(1, node.number("LINE_NUMBER")), Kind: quality.UnknownEffect, Detail: "unresolved C++ member access: " + node.text("CODE")})
		return
	}
	field := record.fields[memberName(node)]
	if staticStorage(field) {
		fact.Effects = append(fact.Effects, quality.Effect{Site: int(p.refs[node]), Line: max(1, node.number("LINE_NUMBER")), Kind: quality.GlobalRead, Detail: record.node.text("FULL_NAME") + "." + memberName(node)})
	}
	if strings.Contains(field.text("CODE")+" "+field.text("TYPE_FULL_NAME"), "volatile") {
		fact.Effects = append(fact.Effects, quality.Effect{Site: int(p.refs[node]), Line: max(1, node.number("LINE_NUMBER")), Kind: quality.UnknownEffect, Detail: "volatile C++ member access"})
	}
}
