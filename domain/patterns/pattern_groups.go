package patterns

import (
	"encoding/json"
	"slices"
)

type PatternGroup string

const (
	GeneralPatterns PatternGroup = "general"
	CPPPatterns     PatternGroup = "cpp"
	GoPatterns      PatternGroup = "go"
)

func (kind CandidateKind) Group() PatternGroup {
	if kind == CPPEmptyCatch {
		return CPPPatterns
	}
	for _, builder := range goSingleFunctionBuilders {
		if kind == builder.kind {
			return GoPatterns
		}
	}
	if slices.Contains(FixKindOrder, kind) {
		return GeneralPatterns
	}
	return ""
}

func (candidate Candidate) MarshalJSON() ([]byte, error) {
	type record Candidate
	return json.Marshal(struct {
		Group PatternGroup `json:"group,omitempty"`
		record
	}{Group: candidate.Kind.Group(), record: record(candidate)})
}
