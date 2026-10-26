package changerisk

import (
	"slices"
	"strings"
)

func ratingIcon(level Level) string {
	if level == Low {
		return "✅"
	}
	if level == Medium || level == High {
		return "⚠️"
	}
	return "👀"
}

func ratingSummary(report Report) string {
	if report.Risk == Unknown {
		return unknownSummary(report.Reasons)
	}
	if report.Risk == Unrated {
		return unratedSummary(report)
	}
	return ""
}

func unknownSummary(reasons []Reason) string {
	var lines []string
	for _, reason := range reasons {
		if reason.Level != Unknown || reason.Rule == "pdg_matching_ambiguous" {
			continue
		}
		lines = append(lines, "- "+reasonSummary(reason))
	}
	slices.Sort(lines)
	return strings.Join(slices.Compact(lines), "\n")
}

func unratedSummary(report Report) string {
	policy := report.Policy
	if policy == "" {
		policy = "gstyle.toml"
	}
	location := " in `" + text(policy) + "` at the base commit."
	if !report.PolicyConfigured {
		return "No risk rules" + location
	}
	return "Missing risk rules for " + strings.Join(unratedCategories(report.Reasons), ", ") + location
}

func unratedCategories(reasons []Reason) []string {
	var names []string
	for _, kind := range []struct{ rule, name string }{
		{"pdg_change", "code impact"},
		{"dependency_change", "dependency changes"},
		{"go_api_breaking", "breaking API changes"},
	} {
		if slices.ContainsFunc(reasons, func(reason Reason) bool { return reason.Level == Unrated && reason.Rule == kind.rule }) {
			names = append(names, kind.name)
		}
	}
	return names
}
