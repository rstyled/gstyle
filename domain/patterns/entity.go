package patterns

import "fmt"

// Entity identity patterns: DDD Entities are defined by identity, not
// attributes (Evans). These find code that violates that principle.

// entityIdentityScoreMilli is the fixed score for attribute-based equality.
// It is advisory: identity equivalence requires a domain invariant.
const entityIdentityScoreMilli = 450

// missingIdentityScoreMilli is the fixed score for structs used as entities
// without an ID field. Adding identity requires a domain decision.
const missingIdentityScoreMilli = 400

// mutableIdentityScoreMilli is the fixed score for ID assignments outside
// constructors. Detection-only: there's no safe mechanical fix.
const mutableIdentityScoreMilli = 350

// entityIdentityCandidates proposes identity design changes for review.
// Neither an ID field nor attribute equality proves identity equivalence.
func entityIdentityCandidates(facts []*MiningFacts) []Candidate {
	var out []Candidate
	for _, f := range facts {
		for _, hit := range f.EntityIdentities {
			hit := hit
			f := f
			if hit.IDField == "" {
				out = append(out, noIDCandidate(f, hit))
				continue
			}
			out = append(out, rewriteCandidate(f, hit))
		}
	}
	return out
}

// noIDCandidate builds the candidate for attribute equality without an ID.
// Adding identity and choosing its semantics require a domain invariant.
func noIDCandidate(f *MiningFacts, hit EntityIdentityHit) Candidate {
	return Candidate{
		Kind:       EntityIdentity,
		ScoreMilli: entityIdentityScoreMilli,
		Breakdown: Breakdown{
			Support:       1,
			CoverageMilli: 1000,
		},
		Observation:      fmt.Sprintf("line %d compares %s by %d fields (%s) but %s has no identity field", hit.Line, hit.TypeName, len(hit.Fields), joinFields(hit.Fields), hit.TypeName),
		Inference:        "entities need identity before they can be compared by it",
		PossibleRefactor: fmt.Sprintf("add ID field to %s, then compare by identity", hit.TypeName),
		Sites: []Site{{
			Path:    f.Path,
			Line:    f.Line,
			EndLine: f.EndLine,
			ID:      f.ID,
			Name:    f.Name,
		}},
		FixSpec: nil,
	}
}

// rewriteCandidate builds the candidate for rewriting to identity comparison.
func rewriteCandidate(f *MiningFacts, hit EntityIdentityHit) Candidate {
	return Candidate{
		Kind:       EntityIdentity,
		ScoreMilli: entityIdentityScoreMilli,
		Breakdown: Breakdown{
			Support:       1,
			CoverageMilli: 1000,
		},
		Observation:      fmt.Sprintf("line %d compares %s by %d fields (%s) instead of by identity", hit.Line, hit.TypeName, len(hit.Fields), joinFields(hit.Fields)),
		Inference:        "if this type is an entity, decide whether identity defines equality",
		PossibleRefactor: fmt.Sprintf("replace with %s.%s == %s.%s", hit.Left, hit.IDField, hit.Right, hit.IDField),
		Sites: []Site{{
			Path:    f.Path,
			Line:    f.Line,
			EndLine: f.EndLine,
			ID:      f.ID,
			Name:    f.Name,
		}},
		FixSpec: nil,
	}
}

// missingIdentityCandidates proposes identity design changes for review.
func missingIdentityCandidates(hits []MissingIdentityHit) []Candidate {
	var out []Candidate
	for _, hit := range hits {
		hit := hit
		out = append(out, Candidate{
			Kind:       MissingIdentity,
			ScoreMilli: missingIdentityScoreMilli,
			Breakdown: Breakdown{
				Support:       hit.UseCount,
				CoverageMilli: 1000,
			},
			Observation:      fmt.Sprintf("type %s is compared by %d attributes but has no identity field", hit.TypeName, hit.UseCount),
			Inference:        "entities need a stable identity independent of their attributes",
			PossibleRefactor: fmt.Sprintf("add an ID field to %s and compare by identity", hit.TypeName),
			Sites: []Site{{
				Path:    hit.Path,
				Line:    hit.Line,
				EndLine: hit.EndLine,
				Name:    hit.TypeName,
			}},
			// No FixSpec: identity semantics require a domain invariant.
			FixSpec: nil,
		})
	}
	return out
}

// mutableIdentityCandidates builds detection-only candidates for ID mutation.
// These have no FixSpec: there's no safe mechanical fix for moving an
// assignment into a constructor.
func mutableIdentityCandidates(facts []*MiningFacts) []Candidate {
	var out []Candidate
	for _, f := range facts {
		for _, hit := range f.MutableIdentities {
			hit := hit
			f := f
			out = append(out, Candidate{
				Kind:       MutableIdentity,
				ScoreMilli: mutableIdentityScoreMilli,
				Breakdown: Breakdown{
					Support:       1,
					CoverageMilli: 1000,
				},
				Observation:      fmt.Sprintf("line %d assigns .%s outside a constructor in %s", hit.Line, hit.Field, hit.FuncName),
				Inference:        "entity identity should be immutable after creation",
				PossibleRefactor: "move the ID assignment into the constructor or factory function",
				Sites: []Site{{
					Path:    f.Path,
					Line:    f.Line,
					EndLine: f.EndLine,
					ID:      f.ID,
					Name:    f.Name,
				}},
				// No FixSpec: detection-only. There is no safe mechanical
				// transform for relocating an assignment into a constructor.
				FixSpec: nil,
			})
		}
	}
	return out
}

// joinFields formats field names for display.
func joinFields(fields []string) string {
	if len(fields) == 0 {
		return ""
	}
	out := fields[0]
	for _, f := range fields[1:] {
		out += ", " + f
	}
	return out
}
