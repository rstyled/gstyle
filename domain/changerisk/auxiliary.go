package changerisk

import (
	"maps"
	"slices"
)

type QualityBaseline struct {
	Version   int                     `json:"version"`
	Generated string                  `json:"generated_at"`
	Commit    string                  `json:"commit"`
	Functions int                     `json:"functions"`
	Tolerance float64                 `json:"tolerance"`
	Rules     map[string]BaselineRule `json:"rules"`
}

type BaselineRule struct {
	Limit  int64   `json:"limit"`
	Excess int64   `json:"excess"`
	Rate   float64 `json:"per_1000_functions"`
}

func auxiliaryReasons(base, head Snapshot) []Reason {
	var reasons []Reason
	for _, path := range slices.Sorted(maps.Keys(base.PreservedDeclarations)) {
		if preservedDeclarations(base.PreservedDeclarations[path], head.PreservedDeclarations[path]) {
			continue
		}
		reasons = append(reasons, Reason{Rule: "analysis_unavailable", Level: Unknown, Path: path,
			Message: "existing covered declarations or test file changed or were removed"})
	}
	if !baselineTightens(base.QualityBaseline, head.QualityBaseline) {
		reasons = append(reasons, Reason{Rule: "analysis_unavailable", Level: Unknown, Path: "quality-baseline.json",
			Message: "quality baseline was added, removed, or changes its rules or permitted excess rates"})
	}
	return reasons
}

func preservedDeclarations(base, head map[string]string) bool {
	for name, before := range base {
		if after, exists := head[name]; !exists || before != after {
			return false
		}
	}
	return true
}

func baselineTightens(base, head *QualityBaseline) bool {
	if base == nil || head == nil {
		return base == head
	}
	return head.Version == base.Version && head.Tolerance <= base.Tolerance &&
		len(head.Rules) == len(base.Rules) && baselineRulesTighten(base.Rules, head.Rules)
}

func baselineRulesTighten(base, head map[string]BaselineRule) bool {
	for name, before := range base {
		after, exists := head[name]
		if !exists || after.Limit != before.Limit || after.Rate > before.Rate {
			return false
		}
	}
	return true
}
