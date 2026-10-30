package cpp

import (
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/quality"
)

func qualityDDD(fact *patterns.MiningFacts, line int) []quality.DDDViolation {
	if fact == nil {
		return nil
	}
	var result []quality.DDDViolation
	for _, hit := range fact.AggregateMods {
		result = append(result, quality.DDDViolation{RuleID: "aggregate", Line: line, Detail: hit.TypeName})
	}
	for _, hit := range fact.MutableIdentities {
		result = append(result, quality.DDDViolation{RuleID: "mutable_identity", Line: hit.Line, Detail: "assigns ." + hit.Field + " outside a constructor"})
	}
	for _, hit := range fact.DbCalls {
		result = append(result, quality.DDDViolation{RuleID: "repository", Line: hit.Line, Detail: "direct db call " + hit.Call})
	}
	return result
}
