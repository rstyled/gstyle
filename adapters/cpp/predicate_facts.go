package cpp

import (
	"slices"
	"strings"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

func (p *projection) predicateFacts(fact *patterns.MiningFacts, nodes []*vertex) {
	for _, node := range nodes {
		if logical(node) == "" || node.Parent != nil && logical(node.Parent) != "" {
			continue
		}
		p.entityPredicate(fact, node)
		p.specificationPredicate(fact, node)
	}
}

func predicateNodes(node *vertex) []*vertex {
	var result []*vertex
	queue := []*vertex{node}
	for len(queue) > 0 {
		n := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		result = append(result, n)
		queue = append(queue, n.Children...)
	}
	return result
}

func (p *projection) specificationPredicate(fact *patterns.MiningFacts, node *vertex) {
	record, root := p.predicateRecord(node)
	if record == nil {
		return
	}
	fact.SpecRules = append(fact.SpecRules, patterns.SpecificationHit{Line: max(1, node.number("LINE_NUMBER")), RuleKey: predicateKey(node, root), CondText: node.text("CODE"), VarName: root.text("NAME"), TypeName: record.node.text("FULL_NAME")})
}

func (p *projection) predicateRecord(node *vertex) (*cppRecord, *vertex) {
	var record *cppRecord
	var root *vertex
	for _, part := range predicateNodes(node) {
		if category(part) == pdg.Call {
			return nil, nil
		}
		if !fieldOperation(part) {
			continue
		}
		r, declaration := p.doc.accessedRecord(part)
		if r == nil || !samePredicateRoot(root, declaration) {
			return nil, nil
		}
		record, root = r, declaration
	}
	return record, root
}

func samePredicateRoot(left, right *vertex) bool { return left == nil || left == right }

func predicateKey(node, root *vertex) string {
	if node.Label == "IDENTIFIER" && len(node.Out["REF"]) == 1 && node.Out["REF"][0] == root {
		return "$"
	}
	if len(node.Children) == 0 {
		return node.Label + ":" + node.text("CODE")
	}
	parts := []string{node.text("NAME")}
	for _, child := range node.Children {
		parts = append(parts, predicateKey(child, root))
	}
	return strings.Join(parts, "|")
}

func (p *projection) entityPredicate(fact *patterns.MiningFacts, node *vertex) {
	comparisons := equalFields(p.doc, node)
	if len(comparisons) < 2 {
		return
	}
	first := comparisons[0]
	fields := map[string]bool{}
	for _, comparison := range comparisons {
		if !sameEqualityOrigin(comparison, first) {
			return
		}
		fields[comparison.field] = true
	}
	if len(fields) < 2 {
		return
	}
	var names []string
	for name := range fields {
		names = append(names, name)
	}
	slices.Sort(names)
	fact.EntityIdentities = append(fact.EntityIdentities, patterns.EntityIdentityHit{Line: max(1, node.number("LINE_NUMBER")), TypeName: first.record.node.text("FULL_NAME"), IDField: first.record.identity, Fields: names, Left: first.left.text("NAME"), Right: first.right.text("NAME"), StructPath: first.record.node.text("FILENAME"), StructLine: max(1, first.record.node.number("LINE_NUMBER"))})
}

func sameEqualityOrigin(left, right fieldEquality) bool {
	return left.record == right.record && left.left == right.left && left.right == right.right
}

type fieldEquality struct {
	record      *cppRecord
	left, right *vertex
	field       string
}

func equalFields(doc *document, node *vertex) []fieldEquality {
	var result []fieldEquality
	for _, part := range predicateNodes(node) {
		if part.text("NAME") != "<operator>.equals" {
			continue
		}
		if comparison, ok := fieldComparison(doc, part); ok {
			result = append(result, comparison)
		}
	}
	return result
}

func fieldComparison(doc *document, node *vertex) (fieldEquality, bool) {
	left, right := argument(node, 1), argument(node, 2)
	if !fieldOperands(left, right) {
		return fieldEquality{}, false
	}
	lrecord, lroot := doc.accessedRecord(left)
	rrecord, rroot := doc.accessedRecord(right)
	if !sameRecordPair(lrecord, rrecord, lroot, rroot) {
		return fieldEquality{}, false
	}
	if !sameComparedField(lrecord, memberName(left), memberName(right)) {
		return fieldEquality{}, false
	}
	return fieldEquality{record: lrecord, left: lroot, right: rroot, field: memberName(left)}, true
}

func sameComparedField(record *cppRecord, left, right string) bool {
	return left == right && record.fields[left] != nil && !identityField(left) && !staticStorage(record.fields[left])
}

func fieldOperands(left, right *vertex) bool {
	return left != nil && right != nil && fieldOperation(left) && fieldOperation(right)
}
func sameRecordPair(left, right *cppRecord, lroot, rroot *vertex) bool {
	return left != nil && left == right && lroot != rroot
}
