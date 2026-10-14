package quality

import (
	"fmt"
	"testing"
)

func TestAuditQualityBoundaries(t *testing.T) {
	cases := []struct {
		id       string
		makeFact func(bool) Function
	}{
		{"fn_length", func(over bool) Function {
			f := auditQualityFact()
			f.CodeLines = 200
			if over {
				f.CodeLines++
			}
			return f
		}},
		{"fn_params", func(over bool) Function {
			f := auditQualityFact()
			f.Params = 4
			if over {
				f.Params++
			}
			return f
		}},
		{"mutation_per_target", func(over bool) Function {
			f := auditQualityFact()
			n := 3
			if over {
				n++
			}
			for i := 0; i < n; i++ {
				f.Mutations = append(f.Mutations, Mutation{Root: "p", RootID: "1", Line: 1, Provenance: External})
			}
			return f
		}},
		{"mutated_targets", func(over bool) Function {
			f := auditQualityFact()
			n := 3
			if over {
				n++
			}
			for i := 0; i < n; i++ {
				f.Mutations = append(f.Mutations, Mutation{Root: fmt.Sprint("p", i), RootID: fmt.Sprint(i), Line: 1, Provenance: External})
			}
			return f
		}},
		{"side_effect_density", func(over bool) Function {
			f := auditQualityFact()
			f.Statements = 6
			if over {
				f.Statements--
			}
			f.Effects = []Effect{{Kind: IO, Line: 1}}
			return f
		}},
		{"aggregate", func(over bool) Function {
			f := auditQualityFact()
			f.DDD = []DDDViolation{{RuleID: "aggregate", Line: 1, Detail: "A"}}
			if over {
				f.DDD = append(f.DDD, DDDViolation{RuleID: "aggregate", Line: 1, Detail: "B"})
			}
			return f
		}},
		{"repository", func(over bool) Function {
			f := auditQualityFact()
			if over {
				f.DDD = []DDDViolation{{RuleID: "repository", Line: 1, Detail: "db.Query"}}
			}
			return f
		}},
		{"mutable_identity", func(over bool) Function {
			f := auditQualityFact()
			if over {
				f.DDD = []DDDViolation{{RuleID: "mutable_identity", Line: 1, Detail: "x.ID"}}
			}
			return f
		}},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			for _, over := range []bool{false, true} {
				report, err := Evaluate([]Function{tc.makeFact(over)}, DefaultConfig())
				if err != nil {
					t.Fatal(err)
				}
				found := false
				for _, d := range report.Diagnostics {
					if d.RuleID == tc.id {
						found = true
					}
				}
				if found != over {
					t.Fatalf("boundary over=%v found=%v", over, found)
				}
			}
		})
	}
}
func auditQualityFact() Function {
	return Function{Location: Location{Path: "p.go", Name: "audit.f", Line: 1, Column: 1}, Statements: 3}
}
