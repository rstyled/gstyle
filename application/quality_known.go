package application

import (
	"fmt"
	"maps"
	"slices"

	"github.com/rstyled/gstyle/domain"
	"github.com/rstyled/gstyle/domain/quality"
)

func knownEffects(effects []quality.Effect) []quality.Effect {
	return slices.DeleteFunc(slices.Clone(effects), func(effect quality.Effect) bool { return effect.Kind == quality.UnknownEffect })
}

func knownMutations(mutations []quality.Mutation) []quality.Mutation {
	return slices.DeleteFunc(slices.Clone(mutations), func(mutation quality.Mutation) bool {
		return mutation.Provenance != quality.Local && mutation.Provenance != quality.External
	})
}

func knownQuality(q *domain.FunctionQuality) domain.FunctionQuality {
	mutations := knownMutations(q.MutationEvents)
	result := q.FunctionResult
	result.Effects = knownEffects(q.Effects)
	if len(mutations) != len(q.MutationEvents) {
		result.Mutations = len(mutations)
		result.MutatedTargets, _ = knownTargetCount(int64(q.MutatedTargets), q.MutationEvents, mutations)
	}
	return domain.FunctionQuality{FunctionResult: result, Diagnostics: knownDiagnostics(q), MutationEvents: mutations}
}

func knownDiagnostics(q *domain.FunctionQuality) []quality.Diagnostic {
	var result []quality.Diagnostic
	for _, diagnostic := range q.Diagnostics {
		visible, keep := knownDiagnostic(diagnostic, q.MutationEvents)
		if !keep {
			continue
		}
		visible.Effects = knownEffects(visible.Effects)
		visible.Mutations = knownMutations(visible.Mutations)
		result = append(result, visible)
	}
	return result
}

func knownDiagnostic(diagnostic quality.Diagnostic, mutations []quality.Mutation) (quality.Diagnostic, bool) {
	switch diagnostic.RuleID {
	case "side_effect_density":
		return knownDensityDiagnostic(diagnostic)
	case "mutation_per_target":
		known := knownMutations(diagnostic.Mutations)
		if len(known) != len(diagnostic.Mutations) {
			return knownCountDiagnostic(diagnostic, int64(len(known)))
		}
	case "mutated_targets":
		return knownTargetsDiagnostic(diagnostic, mutations)
	}
	return diagnostic, true
}

func knownDensityDiagnostic(diagnostic quality.Diagnostic) (quality.Diagnostic, bool) {
	unknown := diagnostic.KindWeights[string(quality.UnknownEffect)]
	if unknown == 0 && !slices.ContainsFunc(diagnostic.Effects, func(effect quality.Effect) bool { return effect.Kind == quality.UnknownEffect }) {
		return diagnostic, true
	}
	if diagnostic.Statements <= 0 || diagnostic.KindWeights == nil {
		return diagnostic, false
	}
	diagnostic.Weight -= unknown
	diagnostic.KindWeights = maps.Clone(diagnostic.KindWeights)
	delete(diagnostic.KindWeights, string(quality.UnknownEffect))
	return knownCountDiagnostic(diagnostic, 1000*diagnostic.Weight/diagnostic.Statements)
}

func knownTargetsDiagnostic(diagnostic quality.Diagnostic, mutations []quality.Mutation) (quality.Diagnostic, bool) {
	known := knownMutations(mutations)
	if len(known) == len(mutations) {
		return diagnostic, true
	}
	count, supported := knownTargetCount(diagnostic.Actual, mutations, known)
	if !supported {
		return diagnostic, false
	}
	return knownCountDiagnostic(diagnostic, int64(count))
}

func knownCountDiagnostic(diagnostic quality.Diagnostic, actual int64) (quality.Diagnostic, bool) {
	diagnostic.Actual = actual
	diagnostic.Message = fmt.Sprintf("%s: %d exceeds %d", diagnostic.RuleID, actual, diagnostic.Limit)
	return diagnostic, actual > diagnostic.Limit
}

func knownTargetCount(actual int64, all, known []quality.Mutation) (int, bool) {
	for _, fields := range []bool{false, true} {
		if int64(mutationTargetCount(all, fields)) == actual {
			return mutationTargetCount(known, fields), true
		}
	}
	return mutationTargetCount(known, false), false
}

func mutationTargetCount(mutations []quality.Mutation, fields bool) int {
	seen := make(map[[3]string]bool)
	for _, mutation := range mutations {
		key := [3]string{mutation.Root, mutation.RootID}
		if fields {
			key[2] = mutation.FieldPath
		}
		seen[key] = true
	}
	return len(seen)
}

func knownDensity(q *domain.FunctionQuality) (int64, bool) {
	if q.Complete {
		return q.DensityMilli, true
	}
	for _, diagnostic := range q.Diagnostics {
		if diagnostic.RuleID == "side_effect_density" && diagnostic.Statements > 0 && diagnostic.KindWeights != nil {
			return 1000 * (diagnostic.Weight - diagnostic.KindWeights[string(quality.UnknownEffect)]) / diagnostic.Statements, true
		}
	}
	return 0, false
}

func qualityDensityLabel(q *domain.FunctionQuality, mutations int) string {
	if density, supported := knownDensity(q); supported {
		return fmt.Sprintf("Known density %d milli · %d mutations", density, mutations)
	}
	return fmt.Sprintf("%d known mutations", mutations)
}

func qualityFindingCount(markers []PatternMarker) int {
	seen := make(map[*quality.Diagnostic]bool)
	for _, marker := range markers {
		if marker.Diagnostic != nil {
			seen[marker.Diagnostic] = true
		}
	}
	return len(seen)
}
