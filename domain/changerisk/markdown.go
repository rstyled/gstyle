package changerisk

import (
	"cmp"
	"fmt"
	"html"
	"slices"
	"strings"
)

func Markdown(report Report) string {
	heading := "**Merge risk: " + string(report.Risk) + "** " + ratingIcon(report.Risk)
	if summary := ratingSummary(report); summary != "" {
		heading += "\n" + summary
	}
	sections := []string{
		heading,
		"**Code impact:** " + pdgVerdict(report) + " · " + pdgSummary(report),
		dependencySection(report), apiSection(report),
	}
	return strings.Join(sections, "\n\n") + "\n\n" + graphSection(prioritizedChanges(report))
}

func apiSection(report Report) string {
	if report.GoAPI.Status == "" {
		return ""
	}
	heading := "**API diff:** " + apiVerdict(report.GoAPI)
	changes := orderedAPIChanges(report.GoAPI.Changes)
	if len(changes) == 0 {
		return heading + "\n" + apiSummary(report.GoAPI)
	}
	return heading + "\n" + strings.Join(apiChangeLines(report, changes), "\n")
}

func prioritizedChanges(report Report) []Delta {
	levels := map[string]Level{}
	for _, reason := range report.Reasons {
		levels[reason.Function] = strongest(levels[reason.Function], reason.Level)
	}
	changes := slices.Clone(report.Changes)
	slices.SortStableFunc(changes, func(a, b Delta) int {
		return cmp.Or(cmp.Compare(levelScore(levels[b.Name]), levelScore(levels[a.Name])), cmp.Compare(a.Name, b.Name))
	})
	return changes
}

func text(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	return strings.ReplaceAll(html.EscapeString(value), "`", "&#96;")
}

func pdgVerdict(report Report) string {
	if unknownReasons(report.Reasons, "analysis_unavailable") != 0 || unknownReasons(report.Reasons, "pdg_change") != 0 {
		return "partial 👀"
	}
	if report.ChangedFunctions != 0 {
		return "changed 👀"
	}
	return "unchanged ✅"
}

func pdgSummary(report Report) string {
	for _, reason := range report.Reasons {
		if reason.Rule == "added_call" && reason.Level == report.PDGRisk {
			return reasonSummary(reason)
		}
	}
	return fmt.Sprintf("Affected functions: %d.", report.ChangedFunctions)
}

func reasonSummary(reason Reason) string {
	message := text(reason.Message)
	if reason.Path != "" {
		message += fmt.Sprintf(" in `%s:%d`", text(reason.Path), reason.Line)
	}
	return message + "."
}

func dependencyVerdict(report Report) string {
	if report.DependencyRisk == Unknown && !report.DependenciesChanged {
		return "unavailable ⚠️"
	}
	if report.DependenciesChanged {
		return "changed 👀"
	}
	return "unchanged ✅"
}

func dependencySummary(report Report) string {
	for _, reason := range report.Reasons {
		if reason.Rule == "dependencies_unavailable" {
			return reasonSummary(reason)
		}
	}
	if !report.DependenciesChanged {
		return ""
	}
	return moduleSummary(moduleChanges(report))
}
