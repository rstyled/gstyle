package changerisk

import (
	"cmp"
	"fmt"
	"html"
	"slices"
	"strings"
)

func Markdown(report Report) string {
	lines := []string{
		"**Merge risk: " + string(report.Risk) + "**", ratingSummary(report.Risk), "",
		"**Code impact:** " + pdgVerdict(report) + "  ", pdgSummary(report), "",
		"**Dependencies:** " + dependencyVerdict(report) + "  ", dependencySummary(report), "",
		"**Analysis limits:** " + analysisSummary(report), "",
		unmeasuredSummary(report.Unmeasured),
		fmt.Sprintf("Compared `%s` → `%s`.", text(shortRevision(report.Base)), text(shortRevision(report.Head))),
	}
	if report.GoAPI.Status != "" {
		lines = slices.Insert(lines, 9, "**API diff:** "+apiVerdict(report.GoAPI)+"  ", apiSummary(report.GoAPI), "")
	}
	return strings.Join(lines, "\n") + "\n" + graphSection(prioritizedChanges(report))
}

func ratingSummary(level Level) string {
	if level == Unrated {
		return "Base policy does not assign severity to all measured changes."
	}
	if level == Unknown {
		return "Static analysis has gaps; review the limits below."
	}
	return "Severity reflects configured static change rules; this is not merge approval."
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
		return "No module-requirement, replacement or package-import changes."
	}
	addedImports, addedModules := dependencyCounts(report.DependenciesAdded)
	removedImports, removedModules := dependencyCounts(report.DependenciesRemoved)
	return fmt.Sprintf("Import edges +%d/−%d; module records +%d/−%d.", addedImports, removedImports, addedModules, removedModules)
}

func dependencyCounts(records []string) (int, int) {
	imports := 0
	for _, record := range records {
		if strings.Contains(record, ":import:") {
			imports++
		}
	}
	return imports, len(records) - imports
}

func shortRevision(commit string) string {
	if len(commit) > 12 {
		return commit[:12]
	}
	return commit
}
