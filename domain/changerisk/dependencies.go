package changerisk

import "slices"

type dependencyComparison struct {
	risk           Level
	changed        bool
	added, removed []string
	reasons        []Reason
}

func compareDependencies(base, head Snapshot, config Config) dependencyComparison {
	reason, changed := dependencyReason(base, head, config)
	if reason == nil {
		return dependencyComparison{risk: Low}
	}
	if !changed {
		return dependencyComparison{risk: reason.Level, reasons: []Reason{*reason}}
	}
	return dependencyComparison{risk: reason.Level, changed: true, reasons: []Reason{*reason},
		added:   dependencyDifference(head.Dependencies, base.Dependencies),
		removed: dependencyDifference(base.Dependencies, head.Dependencies)}
}

func dependencyDifference(items, excluded []string) []string {
	set := map[string]bool{}
	for _, item := range excluded {
		set[item] = true
	}
	result := []string{}
	for _, item := range items {
		if !set[item] {
			result = append(result, item)
		}
	}
	slices.Sort(result)
	return slices.Compact(result)
}
