package changerisk

import (
	"cmp"
	"fmt"
	"path"
	"slices"
	"strings"
)

func orderedAPIChanges(changes []APIChange) []APIChange {
	result := slices.Clone(changes)
	slices.SortStableFunc(result, func(a, b APIChange) int {
		return cmp.Compare(apiChangeRank(a), apiChangeRank(b))
	})
	return result
}

func apiChangeRank(change APIChange) int {
	if !change.Compatible {
		return 0
	}
	if strings.HasPrefix(change.Message, "package ") {
		return 2
	}
	return 1
}

func apiVerdict(api APIReport) string {
	verdict := map[APIStatus]string{
		APIUnchanged: "unchanged ✅", APICompatible: "compatible changes ✅",
		APIBreaking: "breaking changes ⚠️", APIUnavailable: "unavailable ⚠️",
		APIPartial: "partial 👀",
	}[api.Status]
	if len(api.Changes) != 0 {
		return fmt.Sprintf("%s (%d)", verdict, len(api.Changes))
	}
	return verdict
}

func apiSummary(api APIReport) string {
	if api.Status == APIUnavailable {
		return text(api.Error) + "."
	}
	if api.Status == APIPartial {
		return text(strings.Join(api.Limits, "; ")) + "."
	}
	if len(api.Changes) == 0 {
		return fmt.Sprintf("%d root-module packages (current build).", api.PackagesAfter)
	}
	return fmt.Sprintf("Exported API changes: %d (root module, current build).", len(api.Changes))
}

func apiChangeLines(report Report, changes []APIChange) []string {
	var lines []string
	for _, change := range changes {
		lines = append(lines, "- "+apiChangeLine(report, change))
	}
	return lines
}

func apiChangeLine(report Report, change APIChange) string {
	if strings.HasPrefix(change.Message, "package ") {
		return apiPackageLine(report, change)
	}
	symbol, detail, found := strings.Cut(change.Message, ": ")
	if !found {
		return text(change.Package + ": " + change.Message)
	}
	label := path.Base(change.Package) + "." + symbol
	declaration, revision := apiChangeSource(report, change)
	line := apiSourceLabel(report, declaration, revision, label) + ": " + text(detail)
	if declaration != nil {
		line += " — " + codeSpan(declaration.Declaration)
	}
	return line + apiBeforeLink(report, change)
}

func apiPackageLine(report Report, change APIChange) string {
	declaration, revision := apiChangeSource(report, change)
	label := change.Package
	if declaration != nil {
		label = declaration.Path
	}
	line := text(change.Message) + ": " + apiSourceLabel(report, declaration, revision, label)
	if declaration != nil {
		line += fmt.Sprintf(" (%d exported declarations)", declaration.Exports)
	}
	return line
}

func apiChangeSource(report Report, change APIChange) (*APIDeclaration, string) {
	if change.After != nil {
		return change.After, report.Head
	}
	return change.Before, report.Base
}

func apiBeforeLink(report Report, change APIChange) string {
	if change.Before == nil || change.After == nil {
		return ""
	}
	return "; before: " + apiSourceLabel(report, change.Before, report.Base, change.Before.Declaration)
}
