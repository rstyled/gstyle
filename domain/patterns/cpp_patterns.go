package patterns

import (
	"fmt"
	"slices"

	"github.com/rstyled/gstyle/domain/pdg"
)

const CPPEmptyCatch CandidateKind = "cpp_empty_catch"

type CPPEmptyCatchHit struct {
	FunctionID, NodeID string
	Line               int
}

type CPPOptions struct {
	Reason       string
	EmptyCatches []CPPEmptyCatchHit
	Params       Params
	WlCache      *WlCache
	Source       Options
	Measurements MeasurementLimits
}

var cppGeneralAnalyses = []CandidateKind{ThinSliceKind, ChopKind, DependenceCluster, BarrierSliceKind}
var cppGeneralSourceBuilders = cppSourceBuilders()
var cppSourceAnalyses = cppSourceKinds()
var cppMiningAnalyses = []CandidateKind{TraitMethod, CapabilitySet, GenericFn, Parameterize, CCGraphClone, InconsistentClone, MinedRule, AlattinRule}

func cppSourceBuilders() []singleFunctionBuilder {
	result := orderedPatternBuilders(generalSingleFunctionBuilders)
	for i := range result {
		if result[i].kind == Factory {
			result[i].build = func(facts []*MiningFacts, _ Options) []Candidate { return factoryCandidatesWithLimit(facts, -1) }
		}
	}
	return result
}

func cppSourceKinds() []CandidateKind {
	var result []CandidateKind
	for _, builder := range cppGeneralSourceBuilders {
		result = append(result, builder.kind)
	}
	return append(result, CPPEmptyCatch)
}

func RunCPP(facts []*MiningFacts, options CPPOptions) PatternsReport {
	prepared := cppFacts(facts)
	mined := Mined{}
	groups, matching := cppMatchingCandidates(prepared, options)
	mined.Candidates = append(mined.Candidates, matching...)
	mined.Candidates = append(mined.Candidates, cppSourceCandidates(prepared, options.Source)...)
	mined.Candidates = append(mined.Candidates, cppEmptyCatchCandidates(prepared, options.EmptyCatches)...)
	sortMined(&mined)
	return PatternsReport{SignatureGroups: groups, Candidates: mined.Candidates, Measurements: cppGraphMeasurements(prepared, defaultMeasurementLimits(options.Measurements)), AnalysisCoverage: cppCoverage(facts, options.Reason)}
}

func cppMatchingCandidates(facts []*MiningFacts, options CPPOptions) ([]SigGroup, []Candidate) {
	var inputs []*MiningFacts
	for _, fact := range facts {
		if fact.SigKey == "" || !supportedCapability(graphCapability(fact.Canonical, "cpg_matching_labels")) {
			continue
		}
		input := *fact
		input.Name = fact.Canonical.Tables.Text(fact.Canonical.Function.Name.Value)
		input.Pdg = MiningView(fact.Canonical)
		inputs = append(inputs, &input)
	}
	groups := MineGroups(inputs)
	mined := MineCached(inputs, groups, defaultParams(options.Params), options.WlCache)
	mined.Candidates = append(mined.Candidates, cppCloneCandidates(inputs)...)
	mined.Candidates = append(mined.Candidates, cppCallRules(inputs)...)
	for i := range mined.Candidates {
		mined.Candidates[i] = cppMiningCandidate(mined.Candidates[i])
	}
	return groups, mined.Candidates
}

func cppCloneCandidates(inputs []*MiningFacts) []Candidate {
	clones := defaultCloneGroups(inputs)
	pdgs := map[string]*MiningGraph{}
	for _, fact := range inputs {
		pdgs[fact.ID] = fact.Pdg
	}
	result := CCGraphCloneCandidates(clones, inputs)
	divergent := FindInconsistentClones(clones, pdgs)
	return append(result, InconsistentCloneCandidates(divergent, inputs)...)
}

func cppMiningCandidate(candidate Candidate) Candidate {
	candidate.FixSpec = nil
	candidate.ScoreMilli = min(candidate.ScoreMilli, 400)
	candidate.Analysis = &AnalysisEvidence{Scope: ImportedGraphScope, Status: "complete", Conformance: graphConformanceName(pdg.NotEvaluated)}
	candidate.Inference = "the recorded graphs share structural features; behavioral equivalence is not established"
	candidate.CounterEvidence = append(slices.Clone(candidate.CounterEvidence), "C++ graph similarity is a review signal; opaque operations and source equivalence require review")
	return candidate
}

func cppFacts(facts []*MiningFacts) []*MiningFacts {
	var result []*MiningFacts
	for _, f := range facts {
		if f == nil || f.Canonical == nil {
			continue
		}
		if !supportedCapability(graphCapability(f.Canonical, "cpg_method_projection")) {
			continue
		}
		result = append(result, f)
	}
	return result
}

func cppGraphMeasurements(facts []*MiningFacts, limits MeasurementLimits) []DependencyMeasurement {
	var result []DependencyMeasurement
	for _, original := range facts {
		f := semanticFact(original)
		result = append(result, cppFunctionMeasurements(f, limits)...)
	}
	return result
}

func cppFunctionMeasurements(f *MiningFacts, limits MeasurementLimits) []DependencyMeasurement {
	var result []DependencyMeasurement
	for _, kind := range cppGeneralAnalyses {
		if cppAnalysisStatus(f, kind) != "complete" {
			continue
		}
		measurements := dependencyMeasurementsWithLimits(f, kind, limits)
		for i := range measurements {
			measurements[i].Analysis = cppQueryEvidence(f, kind, measurements[i].SeedNode)
		}
		result = append(result, measurements...)
	}
	return result
}

func cppSourceCandidates(facts []*MiningFacts, options Options) []Candidate {
	var mined Mined
	addSingleFunctionCandidates(&mined, facts, options, cppGeneralSourceBuilders)
	byID := buildFactMap(facts)
	for i, candidate := range mined.Candidates {
		mined.Candidates[i] = cppSourceCandidate(candidate, byID[candidate.Sites[0].ID])
	}
	return mined.Candidates
}

func cppSourceCandidate(candidate Candidate, fact *MiningFacts) Candidate {
	sites := slices.Clone(candidate.Sites)
	site := sites[0]
	if candidate.FixSpec != nil {
		site.Line, site.EndLine = candidate.FixSpec.Line, candidate.FixSpec.EndLine
		sites[0] = site
	}
	if candidate.Kind == GuardClause {
		return Candidate{Kind: GuardClause, ScoreMilli: candidate.ScoreMilli, Breakdown: candidate.Breakdown,
			Analysis: cppQueryEvidence(fact, GuardClause), Sites: []Site{site},
			Observation:      fmt.Sprintf("the else branch at line %d returns while the main branch remains nested", site.Line),
			Inference:        "an early-return layout is an alternative to inspect",
			PossibleRefactor: "consider an early return while preserving condition conversion, local scopes, and destructor timing",
			CounterEvidence:  []string{"C++ scopes, overloaded operators, and object lifetimes require manual review; no automatic fix"}}
	}
	return Candidate{Kind: candidate.Kind, ScoreMilli: cppSourceScore(candidate), Breakdown: candidate.Breakdown,
		Analysis: cppQueryEvidence(fact, candidate.Kind), Sites: sites, Definitions: candidate.Definitions,
		Observation: candidate.Observation, Inference: candidate.Inference, PossibleRefactor: cppSourceRefactor(candidate), CounterEvidence: candidate.CounterEvidence}
}

func cppSourceScore(candidate Candidate) uint32 {
	if candidate.Kind == ConstantBoolReturn {
		return candidate.ScoreMilli
	}
	return min(candidate.ScoreMilli, 400)
}

func cppSourceRefactor(candidate Candidate) string {
	switch candidate.Kind {
	case PrimitiveObsession:
		return "consider a named C++ type for the repeated domain value"
	case ValueObject:
		return "consider a C++ value type for parameters that travel together"
	case Factory:
		return "consider a named C++ factory for repeated object construction"
	case Specification:
		return "consider a named C++ predicate for the repeated business rule"
	default:
		return "review the C++ domain boundary and object lifetimes before changing the design"
	}
}

func cppEmptyCatchCandidates(facts []*MiningFacts, hits []CPPEmptyCatchHit) []Candidate {
	byID := buildFactMap(facts)
	var result []Candidate
	for _, hit := range hits {
		f := byID[hit.FunctionID]
		if f == nil {
			continue
		}
		candidate := Candidate{Kind: CPPEmptyCatch, ScoreMilli: 600, Analysis: cppQueryEvidence(f, CPPEmptyCatch, hit.NodeID),
			Observation:      "this C++ catch handler has an uncommented empty body",
			Inference:        "the handler discards a caught exception without an explicit handling action",
			PossibleRefactor: "review the exception policy; document intentional suppression, handle the failure, or rethrow as appropriate",
			CounterEvidence:  []string{"suppression can be intentional, including during cleanup; the finding does not prove a bug"},
			Sites:            []Site{{Path: f.Path, Line: hit.Line, ID: f.ID, Name: f.Name}}}
		result = append(result, candidate)
	}
	return result
}

func cppQueryEvidence(f *MiningFacts, kind CandidateKind, nodes ...string) *AnalysisEvidence {
	if f == nil {
		return &AnalysisEvidence{Scope: ImportedGraphScope, Status: "complete", Conformance: "not_evaluated"}
	}
	return &AnalysisEvidence{FunctionID: f.ID, Scope: ImportedGraphScope, Conformance: graphConformanceName(f.Canonical.Conformance.Status), Status: cppAnalysisStatus(f, kind), NodeIDs: nonemptyStrings(nodes), Missing: capabilityGaps(f.Canonical, cppRequirements(kind))}
}

func cppCoverage(facts []*MiningFacts, reason string) []AnalysisCoverage {
	var result []AnalysisCoverage
	for _, kind := range slices.Concat(cppGeneralAnalyses, cppSourceAnalyses, cppMiningAnalyses) {
		counts := map[string]int{}
		for _, f := range facts {
			counts[cppAnalysisStatus(f, kind)]++
		}
		result = append(result, AnalysisCoverage{Scope: ImportedGraphScope, Kind: kind, Output: cppAnalysisOutput(kind), Complete: counts["complete"], Partial: counts["partial"], Unknown: counts["not_evaluated"], Unavailable: counts["unavailable"], Reason: reason})
	}
	return result
}

func cppAnalysisOutput(kind CandidateKind) string {
	if slices.Contains(cppGeneralAnalyses, kind) {
		return "measurement"
	}
	return "candidate"
}

func cppAnalysisStatus(f *MiningFacts, kind CandidateKind) string {
	if f == nil || f.Canonical == nil {
		return "unavailable"
	}
	if len(capabilityGaps(f.Canonical, cppRequirements(kind))) != 0 {
		return "unavailable"
	}
	return "complete"
}

func cppRequirements(kind CandidateKind) []string {
	required := []string{"cpg_method_projection"}
	if slices.Contains(cppMiningAnalyses, kind) {
		required = append(required, "cpg_matching_labels")
	}
	if slices.Contains(cppGeneralAnalyses, kind) {
		required = append(required, "cpg_data_relations")
	}
	if kind == ChopKind || kind == DependenceCluster || kind == BarrierSliceKind {
		required = append(required, "cpg_control_relations")
	}
	return required
}
