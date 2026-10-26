package gopatterns

import (
	"slices"

	"github.com/rstyled/gstyle/domain/patterns"
)

// PatternFacts includes the analysis evidence consumed by the full report.
func (ex *Extraction) PatternFacts() []*patterns.FuncFacts { return ex.miningFacts(true) }

// FixFacts preserves the narrower evidence supplied to fix candidate selection.
// Source revision and transformation proofs remain separate from these facts.
func (ex *Extraction) FixFacts() []*patterns.FuncFacts { return ex.miningFacts(false) }

func (ex *Extraction) miningFacts(report bool) []*patterns.FuncFacts {
	var facts []*patterns.FuncFacts
	if report {
		facts = make([]*patterns.FuncFacts, 0, len(ex.Funcs))
	}
	for _, function := range ex.Funcs {
		facts = append(facts, function.miningFact(report))
	}
	return facts
}

func (function FuncPdg) miningFact(report bool) *patterns.FuncFacts {
	evidence := function.miningEvidence(report)
	return &patterns.FuncFacts{
		ID: function.Name, Name: function.Name, Path: function.Path,
		Line: function.Line, EndLine: function.EndLine, Pdg: &function.Pdg,
		SigKey: SigKeyOf(function), SelfTy: function.SelfTy,
		GuardClauses:     guardFacts(function.GuardClauses),
		EnumDispatches:   dispatchFacts(function.EnumDispatches),
		TypeSwitches:     typeSwitchFacts(function.TypeSwitches),
		EntityIdentities: evidence.EntityIdentities, MutableIdentities: evidence.MutableIdentities,
		AggregateMods: function.AggregateMods, DbCalls: function.DbCalls,
		FactoryLits: function.FactoryLits, SpecRules: function.SpecRules,
		NilErrHits: function.NilErrHits, ForceTypeAssertHits: function.ForceTypeAssertHits,
		TypedNilHits: function.TypedNilHits, SuppressedKinds: function.SuppressedKinds,
		ConstantBoolReturns: function.ConstantBoolReturns,
		ResourceLifecycles:  evidence.ResourceLifecycles, AstTypes: evidence.AstTypes,
		ErrorCheckSites: evidence.ErrorCheckSites, ReleasedFileLines: evidence.ReleasedFileLines,
		FileObligationSites: evidence.FileObligationSites, DeferredReleaseLines: evidence.DeferredReleaseLines,
		SuccessReturnOutputSites: evidence.SuccessReturnOutputSites, Params: evidence.Params,
	}
}

// Select evidence before constructing facts, rather than patching the result.
// The full report retains its established entity fields; fix planning also
// uses original struct declaration locations from the extraction.
func (function FuncPdg) miningEvidence(report bool) FuncPdg {
	if !report {
		return FuncPdg{EntityIdentities: function.EntityIdentities, MutableIdentities: function.MutableIdentities, Params: copiedHits(function.Params)}
	}
	return FuncPdg{
		ResourceLifecycles: function.ResourceLifecycles, AstTypes: function.AstTypes,
		ErrorCheckSites: function.ErrorCheckSites, ReleasedFileLines: function.ReleasedFileLines,
		FileObligationSites: function.FileObligationSites, DeferredReleaseLines: function.DeferredReleaseLines,
		SuccessReturnOutputSites: function.SuccessReturnOutputSites, Params: function.Params,
		EntityIdentities: reportEntityHits(function.EntityIdentities), MutableIdentities: copiedHits(function.MutableIdentities),
	}
}

// Preserve the former converters' nil result for both nil and empty input.
func copiedHits[T any](hits []T) []T {
	if len(hits) == 0 {
		return nil
	}
	return slices.Clone(hits)
}

func reportEntityHits(hits []patterns.EntityIdentityHit) []patterns.EntityIdentityHit {
	var result []patterns.EntityIdentityHit
	for _, hit := range hits {
		result = append(result, patterns.EntityIdentityHit{
			Line: hit.Line, TypeName: hit.TypeName, IDField: hit.IDField,
			Fields: hit.Fields, Left: hit.Left, Right: hit.Right,
		})
	}
	return result
}

// Keep the adapter's named hit types distinct while using domain field shapes.
func guardFacts(hits []GuardClauseHit) []patterns.GuardClauseHit {
	var result []patterns.GuardClauseHit
	for _, hit := range hits {
		result = append(result, patterns.GuardClauseHit(hit))
	}
	return result
}
func dispatchFacts(hits []EnumDispatchHit) []patterns.EnumDispatchHit {
	var result []patterns.EnumDispatchHit
	for _, hit := range hits {
		result = append(result, patterns.EnumDispatchHit(hit))
	}
	return result
}
func typeSwitchFacts(hits []TypeSwitchHit) []patterns.TypeSwitchHit {
	var result []patterns.TypeSwitchHit
	for _, hit := range hits {
		result = append(result, patterns.TypeSwitchHit(hit))
	}
	return result
}
