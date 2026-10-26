package changerisk

import (
	"maps"
	"slices"
)

type PrivateDeclaration struct{ Path, Name, Signature string }

type TestFile struct {
	Package      string
	Imports      map[string]bool
	Declarations map[string]string
}

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

func privateDeclarationReasons(base, head Snapshot) []Reason {
	var reasons []Reason
	for _, id := range slices.Sorted(maps.Keys(base.PrivateDeclarations)) {
		before := base.PrivateDeclarations[id]
		if after, exists := head.PrivateDeclarations[id]; exists && before == after {
			continue
		}
		reasons = append(reasons, Reason{Rule: "analysis_unavailable", Level: Unknown, Path: before.Path, Function: before.Name,
			Message: "existing private function declaration changed or was removed"})
	}
	return reasons
}

func auxiliaryReasons(base, head Snapshot) []Reason {
	var reasons []Reason
	for _, path := range slices.Sorted(maps.Keys(base.TestFiles)) {
		if after, exists := head.TestFiles[path]; exists && additiveTestFile(base.TestFiles[path], after) {
			continue
		}
		reasons = append(reasons, Reason{Rule: "analysis_unavailable", Level: Unknown, Path: path,
			Message: "test declarations or imports changed or were removed; only additions are covered"})
	}
	if !baselineTightens(base.QualityBaseline, head.QualityBaseline) {
		reasons = append(reasons, Reason{Rule: "analysis_unavailable", Level: Unknown, Path: "quality-baseline.json",
			Message: "quality baseline was added, removed, or changes its rules or permitted excess rates"})
	}
	return reasons
}

func additiveTestFile(base, head TestFile) bool {
	if base.Package != head.Package {
		return false
	}
	if !testImportsRetained(base.Imports, head.Imports) {
		return false
	}
	for name, declaration := range base.Declarations {
		if current, exists := head.Declarations[name]; !exists || current != declaration {
			return false
		}
	}
	return true
}

func testImportsRetained(base, head map[string]bool) bool {
	for imported := range base {
		if !head[imported] {
			return false
		}
	}
	return true
}

func baselineTightens(base, head *QualityBaseline) bool {
	if base == nil && head == nil {
		return true
	}
	if base == nil || head == nil {
		return false
	}
	if !baselinePolicyMatches(base, head) {
		return false
	}
	return baselineRulesTighten(base.Rules, head.Rules)
}

func baselinePolicyMatches(base, head *QualityBaseline) bool {
	return head.Version == base.Version && head.Tolerance <= base.Tolerance && len(head.Rules) == len(base.Rules)
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
