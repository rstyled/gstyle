package cpp

import (
	"slices"
	"strconv"
	"strings"

	"github.com/rstyled/gstyle/domain/patterns"
)

func (doc *document) patternOptions(groups map[*vertex][]*vertex) patterns.Options {
	var result patterns.Options
	uses := map[*cppRecord][]string{}
	ids := map[string]string{}
	for _, method := range doc.Methods {
		ids[method.text("FULL_NAME")] = "cpp:" + strconv.FormatUint(method.ID, 10)
		records := doc.methodRecords(groups[method])
		var names []string
		for record := range records {
			uses[record] = append(uses[record], method.text("FULL_NAME"))
			names = append(names, record.node.text("FULL_NAME"))
		}
		if domainService(method, names) {
			slices.Sort(names)
			result.DomainServices = append(result.DomainServices, patterns.DomainServiceHit{FuncName: method.text("FULL_NAME"), Path: method.text("FILENAME"), Line: max(1, method.number("LINE_NUMBER")), EndLine: max(1, method.number("LINE_NUMBER_END")), Types: names})
		}
	}
	for _, record := range doc.orderedRecords() {
		functions := uses[record]
		if record.methods != 0 {
			continue
		}
		slices.Sort(functions)
		result.AnemicModels = append(result.AnemicModels, patterns.AnemicModelHit{TypeName: record.node.text("FULL_NAME"), Path: record.node.text("FILENAME"), Line: max(1, record.node.number("LINE_NUMBER")), EndLine: recordEnd(record), Funcs: functions, FuncIDs: ids})
	}
	result.MissingIdentities = doc.missingIdentities(groups)
	return result
}

func domainService(method *vertex, names []string) bool {
	return len(names) >= 2 && freeFunction(method)
}

func freeFunction(method *vertex) bool {
	return !slices.ContainsFunc(method.Children, receiverParameter)
}

func (doc *document) orderedRecords() []*cppRecord {
	var result []*cppRecord
	for _, node := range doc.Nodes {
		if record := doc.Records[node.text("FULL_NAME")]; record != nil && record.node == node {
			result = append(result, record)
		}
	}
	return result
}

func (doc *document) missingIdentities(groups map[*vertex][]*vertex) []patterns.MissingIdentityHit {
	counts := map[*cppRecord]int{}
	for _, method := range doc.Methods {
		for _, record := range doc.comparedRecords(groups[method]) {
			counts[record]++
		}
	}
	var result []patterns.MissingIdentityHit
	for _, record := range doc.orderedRecords() {
		if record.identity == "" && counts[record] >= 2 {
			result = append(result, patterns.MissingIdentityHit{TypeName: record.node.text("FULL_NAME"), Path: record.node.text("FILENAME"), Line: max(1, record.node.number("LINE_NUMBER")), EndLine: recordEnd(record), UseCount: counts[record]})
		}
	}
	return result
}

func (doc *document) comparedRecords(nodes []*vertex) []*cppRecord {
	var result []*cppRecord
	for _, node := range nodes {
		if node.text("NAME") != "<operator>.equals" {
			continue
		}
		if comparison, ok := fieldComparison(doc, node); ok {
			result = append(result, comparison.record)
		}
	}
	return result
}

func recordEnd(record *cppRecord) int {
	return max(1, record.node.number("LINE_NUMBER")) + strings.Count(record.node.text("CODE"), "\n")
}

func (doc *document) methodRecords(nodes []*vertex) map[*cppRecord]bool {
	result := map[*cppRecord]bool{}
	for _, node := range nodes {
		if !fieldOperation(node) {
			continue
		}
		record, _ := doc.accessedRecord(node)
		if record != nil {
			result[record] = true
		}
	}
	return result
}
