package changerisk

import (
	"maps"
	"slices"
	"strings"
)

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
		set[dependencyIdentity(item)] = true
	}
	result := []string{}
	for _, item := range items {
		if !set[dependencyIdentity(item)] {
			result = append(result, item)
		}
	}
	slices.Sort(result)
	return slices.Compact(result)
}

func dependencyInventory(records []string) []string {
	set := map[string]bool{}
	for _, record := range records {
		set[dependencyIdentity(record)] = true
	}
	return slices.Sorted(maps.Keys(set))
}

func dependencyIdentity(record string) string {
	scope, value, found := strings.Cut(record, ":import:")
	if !found {
		return record
	}
	_, imported, found := strings.Cut(value, " -> ")
	if !found {
		return record
	}
	return scope + ":import:" + imported
}
