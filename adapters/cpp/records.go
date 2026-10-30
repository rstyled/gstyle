package cpp

import (
	"slices"
	"strings"
)

type cppRecord struct {
	node        *vertex
	fields      map[string]*vertex
	identity    string
	methods     int
	constructor bool
}

func (r *cppRecord) recordMethod(method *vertex) {
	if method.flag("IS_EXTERNAL") {
		return
	}
	if method.text("NAME") == r.node.text("NAME") {
		r.constructor = true
		return
	}
	r.methods++
}

func (record *cppRecord) instanceFields() int {
	count := 0
	for _, field := range record.fields {
		if !staticStorage(field) {
			count++
		}
	}
	return count
}

func (d *document) indexRecords() {
	d.Records = map[string]*cppRecord{}
	d.Enums = map[string]bool{}
	ambiguous := map[string]bool{}
	for _, node := range d.Nodes {
		if node.Label == "TYPE_DECL" && strings.HasPrefix(strings.TrimSpace(node.text("CODE")), "enum ") {
			d.Enums[node.text("FULL_NAME")] = true
		}
		if !recordDefinition(node) {
			continue
		}
		d.indexRecord(node, ambiguous)
	}
	for name := range ambiguous {
		delete(d.Records, name)
	}
}

func (d *document) indexRecord(node *vertex, ambiguous map[string]bool) {
	name := node.text("FULL_NAME")
	if previous := d.Records[name]; previous != nil && previous.node.text("CODE") != node.text("CODE") {
		ambiguous[name] = true
	}
	d.Records[name] = recordDefinitionFacts(node)
}

func (d *document) staticMember(node *vertex) *vertex {
	if node == nil || !fieldOperation(node) {
		return nil
	}
	record, _ := d.accessedRecord(node)
	if record == nil || record.fields[memberName(node)] == nil {
		return nil
	}
	field := record.fields[memberName(node)]
	if !staticStorage(field) {
		return nil
	}
	return field
}

func (d *document) enumSwitch(node *vertex) bool {
	if node.text("CONTROL_STRUCTURE_TYPE") != "SWITCH" || len(node.Out["CONDITION"]) != 1 {
		return false
	}
	condition := node.Out["CONDITION"][0]
	if d.Enums[recordType(condition.text("TYPE_FULL_NAME"))] {
		return true
	}
	if len(condition.Out["REF"]) != 1 {
		return false
	}
	return d.Enums[recordType(condition.Out["REF"][0].text("TYPE_FULL_NAME"))]
}

func (d *document) accessedRecord(node *vertex) (*cppRecord, *vertex) {
	root := argument(node, 1)
	if root == nil || root.Label != "IDENTIFIER" || len(root.Out["REF"]) != 1 {
		return nil, nil
	}
	declaration := root.Out["REF"][0]
	if node.text("NAME") == "<operator>.indirectFieldAccess" && !strings.Contains(declaration.text("TYPE_FULL_NAME"), "*") {
		return nil, nil
	}
	return d.Records[recordType(declaration.text("TYPE_FULL_NAME"))], declaration
}

func recordType(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimSpace(strings.TrimPrefix(name, "const "))
	return strings.TrimSpace(strings.TrimRight(name, "*& "))
}

func instanceMember(record *cppRecord, name string) bool {
	return record != nil && record.fields[name] != nil && !staticStorage(record.fields[name])
}

func recordDefinition(node *vertex) bool {
	if node.Label != "TYPE_DECL" || node.flag("IS_EXTERNAL") || standardType(node.text("FULL_NAME")) {
		return false
	}
	code := strings.TrimSpace(node.text("CODE"))
	return strings.HasPrefix(code, "struct ") || strings.HasPrefix(code, "class ")
}

func recordDefinitionFacts(node *vertex) *cppRecord {
	record := &cppRecord{node: node, fields: map[string]*vertex{}}
	for _, child := range node.Children {
		switch child.Label {
		case "MEMBER":
			name := child.text("NAME")
			record.fields[name] = child
			if identityField(name) && !staticStorage(child) {
				record.identity = name
			}
		case "METHOD":
			record.recordMethod(child)
		}
	}
	return record
}

func staticStorage(node *vertex) bool {
	header, _, _ := strings.Cut(declarationHeader(node), "=")
	if slices.Contains(strings.Fields(header), "static") || slices.Contains(strings.Fields(header), "thread_local") {
		return true
	}
	return slices.ContainsFunc(node.Children, func(child *vertex) bool { return child.text("MODIFIER_TYPE") == "STATIC" })
}

func declarationHeader(node *vertex) string {
	if node.Parent == nil || node.Parent.Label != "TYPE_DECL" {
		return node.text("CODE")
	}
	code := node.Parent.text("CODE")
	offset := node.number("OFFSET") - node.Parent.number("OFFSET")
	if offset < 0 || offset > len(code) || !strings.HasPrefix(code[offset:], node.text("NAME")) {
		return node.text("CODE")
	}
	start := strings.LastIndexAny(code[:offset], ";{}") + 1
	return code[start:offset] + node.text("CODE")
}

func identityField(name string) bool {
	name = strings.ToLower(strings.TrimPrefix(name, "m_"))
	return name == "id" || name == "uuid" || name == "guid"
}

func memberName(node *vertex) string {
	if node == nil {
		return ""
	}
	field := argument(node, 2)
	if field == nil {
		return ""
	}
	return field.text("CANONICAL_NAME")
}

func fieldOperation(node *vertex) bool {
	name := node.text("NAME")
	return name == "<operator>.fieldAccess" || name == "<operator>.indirectFieldAccess"
}
