package gopatterns

import (
	"slices"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

const frameGraphFixture = `package example
import "runtime"
func First()int{return left()}
func Second()int{return right()}
func left()int{return middle()}
func right()int{return middle()}
func middle()int{target:=plain;return target()}
func plain()int{return 1}
func observe()int{_,_,line,_:=runtime.Caller(0);return line}
`

func namedFrameGraph(t *testing.T, extraction *Extraction, name string) *pdg.Graph {
	t.Helper()
	for i := range extraction.Funcs {
		if extraction.Funcs[i].Name == "example."+name {
			return &extraction.Funcs[i].Pdg
		}
	}
	t.Fatalf("missing frame graph: %s", name)
	return nil
}
func TestFrameEvidenceFollowsCanonicalCallValues(t *testing.T) {
	for _, change := range []string{"function identity", "missing identity", "invalid identity", "duplicate identity", "target edge", "conflicting target", "reaching definition", "source revision"} {
		t.Run(change, func(t *testing.T) {
			extraction, report := proofExtraction(t, frameGraphFixture)
			baseline := report
			baseline.Candidates = slices.Clone(report.Candidates)
			if err := VerifyParameterizations(&baseline, extraction); err != nil {
				t.Fatal(err)
			}
			if baseline.Candidates[0].FixSpec == nil {
				t.Fatalf("baseline has no fix: %+v", baseline.Candidates[0])
			}
			graph := namedFrameGraph(t, extraction, "middle")
			switch change {
			case "function identity", "missing identity", "invalid identity", "duplicate identity":
				changed := false
				for i, extensions := range graph.AttributeTables.Extensions {
					for j, extension := range extensions {
						if graph.Tables.Text(extension.Name) != "gstyle.go:function-value" {
							continue
						}
						switch change {
						case "function identity":
							graph.Tables.Strings = append(graph.Tables.Strings, `"example.observe"`)
							graph.AttributeTables.Extensions[i][j].Value = pdg.Text(len(graph.Tables.Strings))
						case "missing identity":
							graph.AttributeTables.Extensions[i] = slices.DeleteFunc(extensions, func(value pdg.Extension) bool { return value.Name == extension.Name })
						case "invalid identity":
							graph.Tables.Strings = append(graph.Tables.Strings, "null")
							graph.AttributeTables.Extensions[i][j].Value = pdg.Text(len(graph.Tables.Strings))
						case "duplicate identity":
							graph.AttributeTables.Extensions[i] = append(slices.Clone(extensions), extension)
						}
						changed = true
						break
					}
					if changed {
						break
					}
				}
				if !changed {
					t.Fatal("function-value fact missing")
				}
			case "target edge":
				graph.Edges = slices.DeleteFunc(graph.Edges, func(edge pdg.Edge) bool { return graph.Tables.Text(edge.Subkind) == "call-target" })
			case "conflicting target":
				found := false
				for _, edge := range graph.Edges {
					if graph.Tables.Text(edge.Subkind) != "call-target" {
						continue
					}
					graph.Tables.Strings = append(graph.Tables.Strings, "conflicting-target")
					edge.ID = pdg.Text(len(graph.Tables.Strings))
					edge.Source = 1
					graph.Edges = append(graph.Edges, edge)
					found = true
					break
				}
				if !found {
					t.Fatal("call-target edge missing")
				}
			case "reaching definition":
				graph.Edges = slices.DeleteFunc(graph.Edges, func(edge pdg.Edge) bool { return graph.Tables.Text(edge.Subkind) == "reaching-definition" })
			case "source revision":
				graph.Tables.Strings = append(graph.Tables.Strings, "stale-transitive-graph")
				graph.Sources[0].ContentIdentity.Value = pdg.Text(len(graph.Tables.Strings))
			}
			if err := pdg.Validate(graph); err != nil {
				t.Fatalf("mutation is not structurally valid: %v", err)
			}
			if err := VerifyParameterizations(&report, extraction); err != nil {
				t.Fatal(err)
			}
			candidate := report.Candidates[0]
			reason := strings.Join(candidate.CounterEvidence, " ")
			want := "unknown"
			if change == "function identity" {
				want = "stack-inspection"
			}
			if candidate.FixSpec != nil || !strings.Contains(reason, want) {
				t.Fatalf("graph evidence bypassed: %+v", candidate)
			}
		})
	}
}
func TestFrameEffectsResolveStaticMethodExpressionsOnly(t *testing.T) {
	source := `package example
type box struct{}
func(box) plain(x int)int{return x}
func First(x int)int{return left(x)}
func Second(x int)int{return right(x)}
func left(x int)int{return x}
func right(x int)int{return x+1}
func static(b box)int{target:=box.plain;return target(b,1)}
func bound(b box)int{target:=b.plain;return target(1)}
`
	extraction, _ := proofExtraction(t, source)
	effects := extractedFrameEffects(t, extraction)
	if effects["example.static"] != (callflow.FrameEffect{}) {
		t.Fatalf("static method=%+v", effects["example.static"])
	}
	if !effects["example.bound"].Unknown {
		t.Fatal("bound method implementation guessed")
	}
	graph := namedFrameGraph(t, extraction, "static")
	if _, err := callflow.FrameEffects(map[string]*pdg.Graph{"wrong.identity": graph}); err == nil {
		t.Fatal("graph attached to wrong function identity")
	}
	invalid := *graph
	invalid.Nodes = slices.Clone(graph.Nodes)
	invalid.Nodes[0].Span = pdg.Ref(len(graph.Spans) + 1)
	if _, err := callflow.FrameEffects(map[string]*pdg.Graph{"example.static": &invalid}); err == nil {
		t.Fatal("invalid transitive graph accepted")
	}
}
