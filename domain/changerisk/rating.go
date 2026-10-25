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
	if report.Risk == Unrated {
		return unratedSummary(report)
	}
	if report.Risk == Unknown {
		return "Static analysis is incomplete; manual review is needed."
	}
	return "Severity reflects configured static change rules; this is not merge approval."
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
