package cpp

import "github.com/rstyled/gstyle/domain/pdg"

func (p *projection) fieldResultType(node *vertex) string {
	if !fieldOperation(node) {
		return "ANY"
	}
	record, _ := p.doc.accessedRecord(node)
	if record == nil || record.fields[memberName(node)] == nil {
		return "ANY"
	}
	return record.fields[memberName(node)].text("TYPE_FULL_NAME")
}

func (p *projection) operationResultType(node *vertex) string {
	if category(node) == pdg.Call {
		return p.callResultType(node)
	}
	if !operatorCall(node) {
		return "ANY"
	}
	switch node.text("NAME") {
	case "<operator>.sizeOf":
		return "std::size_t"
	case "<operator>.equals", "<operator>.notEquals", "<operator>.lessThan", "<operator>.lessEqualsThan", "<operator>.greaterThan", "<operator>.greaterEqualsThan", "<operator>.logicalAnd", "<operator>.logicalOr", "<operator>.logicalNot":
		return p.predicateResultType(node)
	default:
		return "ANY"
	}
}

func (p *projection) predicateResultType(node *vertex) string {
	if len(node.Children) > 0 && !p.opaqueOperator(node) {
		return "bool"
	}
	return "ANY"
}

func (p *projection) callResultType(node *vertex) string {
	attrs := p.attrs[p.refs[node]-1]
	if attrs.Callee.Status != pdg.Known {
		if p.modeledCtype(node) != "" {
			return "int"
		}
		return "ANY"
	}
	targets := p.doc.targetMethods(node)
	if len(targets) != 1 {
		return "ANY"
	}
	return methodResultType(targets[0])
}
