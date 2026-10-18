package gopatterns

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"go/token"
)

type parameterizationAttempt struct {
	spec    *patterns.FixSpec
	reasons []string
}

type parameterizationVerifier struct {
	frames   map[string]callflow.FrameEffect
	fset     *token.FileSet
	graphs   map[string]*pdg.Graph
	files    map[string]*parameterizationSource
	names    map[string][]string
	claimed  map[string]bool
	attempts map[[2]string]parameterizationAttempt
}

func (v *parameterizationVerifier) candidates(candidate patterns.Candidate) ([]patterns.Candidate, error) {
	if candidate.Kind != patterns.Parameterize {
		return []patterns.Candidate{candidate}, nil
	}
	if len(candidate.Sites) == 2 {
		pair, err := v.pair(candidate)
		return []patterns.Candidate{pair}, err
	}
	return v.group(candidate)
}

func (v *parameterizationVerifier) group(candidate patterns.Candidate) ([]patterns.Candidate, error) {
	advisory := unverifiedParameterization(candidate, "group similarity does not prove a shared helper; function pairs are verified independently")
	advisory.FixSpec = nil
	result := []patterns.Candidate{advisory}
	for i, first := range candidate.Sites {
		for _, second := range candidate.Sites[i+1:] {
			if !v.availablePair(first, second) {
				continue
			}
			pair := candidate
			pair.Sites = []patterns.Site{first, second}
			verified, err := v.pair(pair)
			if err != nil {
				return nil, err
			}
			if verified.FixSpec != nil {
				result = append(result, verified)
			}
		}
	}
	return result, nil
}

func (v *parameterizationVerifier) availablePair(first, second patterns.Site) bool {
	return functionPackage(first.ID) == functionPackage(second.ID) && !v.claimed[first.ID] && !v.claimed[second.ID]
}

func (v *parameterizationVerifier) pair(candidate patterns.Candidate) (patterns.Candidate, error) {
	first, second := candidate.Sites[0], candidate.Sites[1]
	if v.claimed[first.ID] || v.claimed[second.ID] {
		candidate.FixSpec = nil
		return unverifiedParameterization(candidate, "overlapping extraction is deferred to the next source revision"), nil
	}
	key := parameterizationPairKey(first.ID, second.ID)
	attempt, ok := v.attempts[key]
	if !ok {
		verified, err := v.verifyParameterization(candidate)
		if err != nil {
			return candidate, err
		}
		attempt = parameterizationAttempt{spec: verified.FixSpec, reasons: verified.CounterEvidence}
		v.attempts[key] = attempt
	}
	candidate.FixSpec = attempt.spec
	candidate.CounterEvidence = attempt.reasons
	if attempt.spec != nil {
		v.claimed[first.ID], v.claimed[second.ID] = true, true
		packageName := functionPackage(first.ID)
		v.names[packageName] = append(append([]string(nil), v.names[packageName]...), attempt.spec.Params["helper_name"])
		candidate.Observation = fmt.Sprintf("%s and %s share a complete body under verified callable substitutions", attempt.spec.Params["pair_a"], attempt.spec.Params["pair_b"])
		candidate.Breakdown.Support = 2
		candidate.Inference = "complete source bodies match under callable substitutions with compatible PDG types and dependencies"
		candidate.PossibleRefactor = "extract the verified pair into a shared helper with the differing callables as parameters"
	}
	return candidate, nil
}

func parameterizationPairKey(first, second string) [2]string {
	if second < first {
		return [2]string{second, first}
	}
	return [2]string{first, second}
}
