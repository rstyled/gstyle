package patterns

var goPatternBuilders = orderedPatternBuilders(generalSingleFunctionBuilders, goSingleFunctionBuilders)

var goGeneralAnalyses = []CandidateKind{BarrierSliceKind, ThinSliceKind, ChopKind}

func RunGo(facts []*FuncFacts, options Options) PatternsReport {
	views := make([]*MiningFacts, 0, len(facts))
	labels := sharedBaseLabels(facts)
	for _, f := range facts {
		if f == nil {
			continue
		}
		view := miningFacts(f)
		labels.attach(view.Pdg)
		views = append(views, view)
	}
	options.semantic = goSemanticResults(mergeFacts(views), options.Callables)
	options.Callables = nil
	for _, view := range views {
		view.Canonical = nil
	}
	return RunGoMining(views, options)
}

func RunGoMining(facts []*MiningFacts, options Options) PatternsReport {
	return runMining(facts, options, goPatternBuilders)
}

var goSingleFunctionBuilders = []singleFunctionBuilder{
	{OffsetFallback, func(_ []*MiningFacts, options Options) []Candidate { return options.OffsetFallbacks }},
	{DeclarationOrder, func(_ []*MiningFacts, options Options) []Candidate { return options.DeclarationOrders }},
	{NilErr, func(prepared []*MiningFacts, options Options) []Candidate { return nilErrCandidates(prepared) }},
	{ForceTypeAssert, func(prepared []*MiningFacts, options Options) []Candidate { return forceTypeAssertCandidates(prepared) }},
	{TypedNil, func(prepared []*MiningFacts, options Options) []Candidate { return typedNilCandidates(prepared) }},
	{NilGuard, func(_ []*MiningFacts, options Options) []Candidate { return options.NilGuards }},
	{NegativeIndex, func(_ []*MiningFacts, options Options) []Candidate { return options.NegativeIndices }},
	{LiteralIndex, func(_ []*MiningFacts, options Options) []Candidate { return options.LiteralIndices }},
	{TaintFlowKind, func(prepared []*MiningFacts, options Options) []Candidate {
		if options.semantic != nil {
			return options.semantic.candidates(TaintFlowKind)
		}
		flows := FindTaintFlows(prepared)
		return TaintFlowCandidates(flows, prepared)
	}},
	{Obligation, func(prepared []*MiningFacts, options Options) []Candidate {
		violations := FindObligationViolations(prepared)
		return ObligationCandidates(violations, prepared)
	}},
	{TypeSwitch, func(prepared []*MiningFacts, _ Options) []Candidate { return typeSwitchCandidates(prepared) }},
}
