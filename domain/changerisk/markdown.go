package changerisk

import (
	"cmp"
	"fmt"
	"html"
	"slices"
	"strings"
)

func Markdown(report Report) string {
	dependencies := append([]string{"**Dependencies:** " + dependencyVerdict(report) + "  ", dependencySummary(report)}, dependencyLines(report)...)
	sections := []string{
		"**Merge risk: " + string(report.Risk) + "** " + ratingIcon(report.Risk) + "\n" + ratingSummary(report),
		"**Code impact:** " + pdgVerdict(report) + " · " + pdgSummary(report),
		strings.Join(dependencies, "\n"), apiSection(report, maxAPIChanges),
	}
	return strings.Join(sections, "\n\n") + "\n\n" + analysisDetails(report) + graphSection(prioritizedChanges(report))
}

func analysisDetails(report Report) string {
	var modules []string
	for _, change := range moduleChanges(report) {
		modules = append(modules, "- "+moduleChangeLine(change))
	}
	lines := []string{"<details>", "<summary>Analysis details</summary>", "",
		unmeasuredSummary(report.Unmeasured),
		fmt.Sprintf("Compared `%s` → `%s`.", text(shortRevision(report.Base)), text(shortRevision(report.Head))), "",
		strings.Join(modules, "\n"), "", apiSection(report, len(report.GoAPI.Changes)), "", "</details>", ""}
	return strings.Join(lines, "\n")
}

func apiSection(report Report, limit int) string {
	if report.GoAPI.Status == "" {
		return ""
	}
	heading := "**API diff:** " + apiVerdict(report.GoAPI)
	changes := orderedAPIChanges(report.GoAPI.Changes)
	if len(changes) == 0 {
		return heading + "\n" + apiSummary(report.GoAPI)
	}
	return heading + "\n" + strings.Join(apiChangeLines(report, changes[:min(limit, len(changes))]), "\n")
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
	addedImports, _ := dependencyCounts(report.DependenciesAdded)
	removedImports, _ := dependencyCounts(report.DependenciesRemoved)
	return fmt.Sprintf("Imports: **+%d / −%d**.", addedImports, removedImports)
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
