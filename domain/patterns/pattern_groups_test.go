package patterns

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestPatternGroupsInJSON(t *testing.T) {
	for _, kind := range FixKindOrder {
		if kind.Group() == "" {
			t.Errorf("unclassified pattern %s", kind)
		}
	}
	for kind, want := range map[CandidateKind]PatternGroup{ThinSliceKind: GeneralPatterns, CCGraphClone: GeneralPatterns, GuardClause: GeneralPatterns, TypedNil: GoPatterns, TaintFlowKind: GoPatterns, Obligation: GoPatterns, CPPEmptyCatch: CPPPatterns} {
		data, err := json.Marshal(Candidate{Kind: kind})
		if err != nil {
			t.Fatal(err)
		}
		var got struct {
			Group PatternGroup `json:"group"`
			Kind  CandidateKind
		}
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatal(err)
		}
		if got.Group != want || got.Kind != kind {
			t.Errorf("%s: %s", kind, data)
		}
	}
	if CandidateKind("unknown").Group() != "" {
		t.Fatal("unknown rule is classified as supported")
	}
}

func TestGeneralAndGoBuildersStaySeparate(t *testing.T) {
	facts := []*MiningFacts{{ID: "method", Path: "fixture.go", GuardClauses: []GuardClauseHit{{Line: 1, BodyStmts: 3}}, NilErrHits: []NilErrHit{{Line: 5}}}}
	general := runMining(facts, Options{}, orderedPatternBuilders(generalSingleFunctionBuilders))
	if len(general.Candidates) != 1 || general.Candidates[0].Kind != GuardClause {
		t.Fatalf("general registry runs foreign rules: %+v", general.Candidates)
	}
	goReport := RunGoMining(facts, Options{})
	if len(goReport.Candidates) != 2 {
		t.Fatalf("Go registry loses shared or Go rules: %+v", goReport.Candidates)
	}
}

func TestCPPProfileDoesNotRunGoModelsOrFixes(t *testing.T) {
	graph := semanticFlowFixture(t)
	for _, name := range []string{"cpg_method_projection", "cpg_data_relations", "cpg_control_relations", "cpg_matching_labels"} {
		graph.Tables.Strings = append(graph.Tables.Strings, name, "fixture import contract")
		graph.Tables.Evidence = append(graph.Tables.Evidence, pdg.Capability{Status: pdg.Supported, Description: pdg.Text(len(graph.Tables.Strings))})
		graph.Capabilities = append(graph.Capabilities, pdg.NamedCapability{Name: pdg.Text(len(graph.Tables.Strings) - 1), Evidence: pdg.Ref(len(graph.Tables.Evidence))})
	}
	fact := &MiningFacts{ID: "cpp:method", Canonical: graph, GuardClauses: []GuardClauseHit{{Line: 1, BodyStmts: 3}}, NilErrHits: []NilErrHit{{Line: 5}}}
	if len(FindTaintFlows([]*MiningFacts{fact})) == 0 {
		t.Fatal("fixture must exercise the Go taint models")
	}
	report := RunCPP([]*MiningFacts{fact, nil}, CPPOptions{Reason: "fixture scope"})
	if len(report.Candidates) != 1 || report.Candidates[0].Kind != GuardClause || report.Candidates[0].FixSpec != nil {
		t.Fatalf("C++ profile runs Go models or fixes: %+v", report.Candidates)
	}
	if len(report.AnalysisCoverage) != len(cppGeneralAnalyses)+len(cppSourceAnalyses)+len(cppMiningAnalyses) {
		t.Fatalf("unexpected C++ analyses: %+v", report.AnalysisCoverage)
	}
	for _, coverage := range report.AnalysisCoverage {
		if coverage.Complete != 1 || coverage.Partial != 0 || coverage.Unavailable != 1 || coverage.Reason != "fixture scope" || coverage.Scope != ImportedGraphScope {
			t.Fatalf("wrong C++ coverage: %+v", coverage)
		}
	}
	if !HasIncompleteAnalysis(report.AnalysisCoverage) {
		t.Fatal("imported-graph coverage is mistaken for C++ source-semantic certification")
	}
}

func TestFrontendGraphProfilesUseKnownGeneralAnalyses(t *testing.T) {
	for _, enabled := range [][]CandidateKind{goGeneralAnalyses, cppGeneralAnalyses} {
		for _, kind := range enabled {
			if generalGraphBuilders[kind] == nil || kind.Group() != GeneralPatterns {
				t.Errorf("invalid shared analysis %s", kind)
			}
		}
	}
}

func TestCPPQueriesRequireVerifiedProjectionAndOverlays(t *testing.T) {
	graph := semanticFlowFixture(t)
	fact := &MiningFacts{ID: "cpp:method", Canonical: graph, GuardClauses: []GuardClauseHit{{Line: 1, BodyStmts: 3}}}
	report := RunCPP([]*MiningFacts{fact}, CPPOptions{})
	if len(report.Candidates) != 0 {
		t.Fatal("uncertified import produces C++ findings")
	}
	for _, coverage := range report.AnalysisCoverage {
		if coverage.Unavailable != 1 || coverage.Complete != 0 {
			t.Fatalf("uncertified import claims coverage: %+v", coverage)
		}
	}
	graph.Tables.Strings = append(graph.Tables.Strings, "cpg_method_projection", "fixture projection certificate")
	graph.Tables.Evidence = append(graph.Tables.Evidence, pdg.Capability{Status: pdg.Supported, Description: pdg.Text(len(graph.Tables.Strings))})
	graph.Capabilities = append(graph.Capabilities, pdg.NamedCapability{Name: pdg.Text(len(graph.Tables.Strings) - 1), Evidence: pdg.Ref(len(graph.Tables.Evidence))})
	report = RunCPP([]*MiningFacts{fact}, CPPOptions{})
	for _, coverage := range report.AnalysisCoverage {
		if slices.Contains(cppGeneralAnalyses, coverage.Kind) || slices.Contains(cppMiningAnalyses, coverage.Kind) {
			if coverage.Unavailable != 1 || coverage.Complete != 0 {
				t.Fatalf("missing overlay claims graph-query coverage: %+v", coverage)
			}
		} else if coverage.Complete != 1 {
			t.Fatalf("AST query requires unrelated dependency overlays: %+v", coverage)
		}
	}
	if len(report.Candidates) != 1 || report.Candidates[0].Kind != GuardClause {
		t.Fatalf("AST query does not run independently: %+v", report.Candidates)
	}
}
