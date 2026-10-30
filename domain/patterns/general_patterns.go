package patterns

import "slices"

func orderedPatternBuilders(groups ...[]singleFunctionBuilder) []singleFunctionBuilder {
	var result []singleFunctionBuilder
	for _, group := range groups {
		result = append(result, group...)
	}
	slices.SortFunc(result, func(a, b singleFunctionBuilder) int { return FixKindRank(a.kind) - FixKindRank(b.kind) })
	return result
}

var generalGraphBuilders = map[CandidateKind]func(*MiningFacts) []Candidate{
	ThinSliceKind: func(f *MiningFacts) []Candidate { return ThinSliceCandidates(thinSlicesForFunc(f), []*MiningFacts{f}) },
	ChopKind:      func(f *MiningFacts) []Candidate { return ChopCandidates(chopsForFunc(f), []*MiningFacts{f}) },
	DependenceCluster: func(f *MiningFacts) []Candidate {
		return DepClusterCandidates(FindDepClusters([]*MiningFacts{f}), []*MiningFacts{f})
	},
	BarrierSliceKind: func(f *MiningFacts) []Candidate {
		return BarrierSliceCandidates(barrierSlicesForFunc(f), []*MiningFacts{f})
	},
}

func selectedGeneralBuilders(kinds ...CandidateKind) []singleFunctionBuilder {
	var result []singleFunctionBuilder
	for _, builder := range generalSingleFunctionBuilders {
		if slices.Contains(kinds, builder.kind) {
			result = append(result, builder)
		}
	}
	return orderedPatternBuilders(result)
}

func generalFunctionCandidates(f *MiningFacts, enabled []CandidateKind) []Candidate {
	var result []Candidate
	for _, kind := range enabled {
		result = append(result, generalGraphBuilders[kind](f)...)
	}
	return result
}

var generalSingleFunctionBuilders = []singleFunctionBuilder{
	{GuardClause, func(prepared []*MiningFacts, options Options) []Candidate { return guardCandidates(prepared) }},
	{PrimitiveObsession, func(prepared []*MiningFacts, options Options) []Candidate {
		return primitiveObsessionCandidates(prepared)
	}},
	{ValueObject, func(prepared []*MiningFacts, options Options) []Candidate { return valueObjectCandidates(prepared) }},
	{Factory, func(prepared []*MiningFacts, options Options) []Candidate { return factoryCandidates(prepared) }},
	{Specification, func(prepared []*MiningFacts, options Options) []Candidate { return specificationCandidates(prepared) }},
	{MissingIdentity, func(prepared []*MiningFacts, options Options) []Candidate {
		return missingIdentityCandidates(options.MissingIdentities)
	}},
	{EntityIdentity, func(prepared []*MiningFacts, options Options) []Candidate { return entityIdentityCandidates(prepared) }},
	{DomainService, func(prepared []*MiningFacts, options Options) []Candidate {
		return domainServiceCandidates(options.DomainServices)
	}},
	{AnemicModel, func(prepared []*MiningFacts, options Options) []Candidate {
		return anemicModelCandidates(options.AnemicModels, options.DomainServices)
	}},
	{MutableIdentity, func(prepared []*MiningFacts, options Options) []Candidate { return mutableIdentityCandidates(prepared) }},
	{Aggregate, func(prepared []*MiningFacts, options Options) []Candidate { return aggregateCandidates(prepared) }},
	{Repository, func(prepared []*MiningFacts, options Options) []Candidate { return repositoryCandidates(prepared) }},
	{ConstantBoolReturn, func(prepared []*MiningFacts, _ Options) []Candidate { return constantBoolReturnCandidates(prepared) }},
	{EnumDispatch, func(prepared []*MiningFacts, _ Options) []Candidate { return enumDispatchCandidates(prepared) }},
}
