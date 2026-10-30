package cpp

import (
	"slices"
	"strings"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

func (p *projection) generalFacts(fact *patterns.MiningFacts, nodes []*vertex) {
	for _, node := range nodes {
		p.generalWrite(fact, node)
		p.generalConstruct(fact, node)
		if databaseCall(node) {
			fact.DbCalls = append(fact.DbCalls, patterns.DbCallHit{Line: max(1, node.number("LINE_NUMBER")), Call: node.text("CODE")})
		}
		if p.doc.enumSwitch(node) {
			fact.EnumDispatches = append(fact.EnumDispatches, patterns.EnumDispatchHit{Line: max(1, node.number("LINE_NUMBER")), NumCases: caseCount(node)})
		}
	}
	p.predicateFacts(fact, nodes)
}

func (p *projection) generalWrite(fact *patterns.MiningFacts, node *vertex) {
	record, target := p.recordWrite(node)
	if record == nil {
		return
	}
	name := record.node.text("FULL_NAME")
	fact.AggregateMods = append(fact.AggregateMods, patterns.AggregateModHit{TypeName: name})
	if identityField(memberName(target)) && p.method.text("NAME") != record.node.text("NAME") {
		fact.MutableIdentities = append(fact.MutableIdentities, patterns.MutableIdentityHit{Line: max(1, node.number("LINE_NUMBER")), Field: memberName(target), FuncName: p.method.text("FULL_NAME")})
	}
}

func (p *projection) recordWrite(node *vertex) (*cppRecord, *vertex) {
	if category(node) != pdg.Assignment {
		return nil, nil
	}
	target := argument(node, 1)
	if target == nil || !fieldOperation(target) {
		return nil, nil
	}
	record, _ := p.doc.accessedRecord(target)
	if !instanceMember(record, memberName(target)) {
		return nil, nil
	}
	return record, target
}

func (p *projection) generalConstruct(fact *patterns.MiningFacts, node *vertex) {
	if node.Label != "CALL" || !strings.Contains(node.text("CODE"), "{") {
		return
	}
	signature := ":" + node.text("SIGNATURE")
	owner := strings.TrimSuffix(node.text("METHOD_FULL_NAME"), signature)
	owner = strings.TrimSuffix(owner, "."+node.text("NAME"))
	record := p.doc.Records[owner]
	if record == nil || record.constructor || record.instanceFields() < 5 {
		return
	}
	fact.FactoryLits = append(fact.FactoryLits, patterns.FactoryHit{Line: max(1, node.number("LINE_NUMBER")), TypeName: record.node.text("FULL_NAME"), NumFields: record.instanceFields(), DeclFile: record.node.text("FILENAME"), HasLogic: methodLogic(p.method)})
}

func databaseCall(node *vertex) bool {
	if node.Label != "CALL" || !concreteName(node.text("METHOD_FULL_NAME")) {
		return false
	}
	return slices.Contains([]string{"sqlite3_exec", "sqlite3_prepare_v2", "sqlite3_step", "mysql_query", "mysql_real_query", "PQexec", "PQexecParams"}, node.text("NAME"))
}

func methodLogic(node *vertex) bool {
	for _, child := range node.Children {
		if child.Owner != node.Owner {
			continue
		}
		if child.Label == "CONTROL_STRUCTURE" || methodLogic(child) {
			return true
		}
	}
	return false
}

func caseCount(node *vertex) int {
	count := 0
	for _, child := range node.Children {
		if child.text("CONTROL_STRUCTURE_TYPE") == "SWITCH" {
			continue
		}
		if child.Label == "JUMP_TARGET" && strings.HasPrefix(child.text("CODE"), "case ") {
			count++
		}
		count += caseCount(child)
	}
	return count
}
