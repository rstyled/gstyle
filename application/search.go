package application

import (
	"fmt"
	"strings"

	"github.com/rstyled/gstyle/domain"
)

// SearchQuery selects functions, optionally restricted to quality findings.
// Text and path are case-insensitive literal substrings; rule is an exact ID.
type SearchQuery struct {
	Query                  string `json:"query"`
	Path                   string `json:"path"`
	Rule                   string `json:"rule"`
	FindingsOnly           bool   `json:"findingsOnly"`
	MinComplexity          int    `json:"minComplexity"`
	MinCognitiveComplexity int    `json:"minCognitiveComplexity"`
	Offset                 int    `json:"offset"`
	Limit                  int    `json:"limit"`
}

// Validate rejects invalid bounds before analysis or model access.
func (q SearchQuery) Validate() error {
	if q.Offset < 0 {
		return fmt.Errorf("offset must be non-negative")
	}
	if q.Limit < 1 || q.Limit > 200 {
		return fmt.Errorf("limit must be between 1 and 200")
	}
	if q.MinComplexity < 0 || q.MinCognitiveComplexity < 0 {
		return fmt.Errorf("minimum complexity must be non-negative")
	}
	return nil
}

type SearchFinding struct {
	RuleID  string `json:"ruleId"`
	Message string `json:"message"`
	Actual  int64  `json:"actual"`
	Limit   int64  `json:"limit"`
}

type SearchMatch struct {
	FileIndex           int             `json:"fileIndex"`
	FunctionIndex       int             `json:"functionIndex"`
	Path                string          `json:"path"`
	Package             string          `json:"package"`
	Name                string          `json:"name"`
	Line                int             `json:"line"`
	EndLine             int             `json:"endLine"`
	Column              int             `json:"column"`
	Complexity          int             `json:"complexity"`
	CognitiveComplexity int             `json:"cognitiveComplexity"`
	Findings            []SearchFinding `json:"findings"`
}

type SearchResult struct {
	Root       string                 `json:"root"`
	Total      int                    `json:"total"`
	Offset     int                    `json:"offset"`
	Limit      int                    `json:"limit"`
	NextOffset *int                   `json:"nextOffset,omitempty"`
	Matches    []SearchMatch          `json:"matches"`
	Quality    *domain.QualitySummary `json:"quality,omitempty"`
}

// SearchReport uses the same ranking and indexes as getReport. It does not
// change the report or shared selection. Empty query browses all functions.
func SearchReport(report domain.Report, query SearchQuery) (SearchResult, error) {
	if err := query.Validate(); err != nil {
		return SearchResult{}, err
	}
	return searchRankedReport(rankReport(report), query), nil
}

// searchRankedReport accepts a validated query and an already ranked report.
// The live model ranks on refresh, so queries reuse its arrays and indexes.
func searchRankedReport(report domain.Report, input SearchQuery) SearchResult {
	query := input.normalized()
	matches := []SearchMatch{}
	total := 0
	for fileIndex, file := range report.Files {
		page := query.remaining(total, len(matches))
		fileMatches, count := searchFile(fileIndex, file, page)
		matches = append(matches, fileMatches...)
		total += count
	}
	result := SearchResult{
		Root: report.Root, Offset: query.Offset, Limit: query.Limit,
		Total: total, Matches: matches, Quality: qualitySummary(report.Quality),
	}
	if result.Total-query.Offset > len(result.Matches) {
		next := query.Offset + len(result.Matches)
		result.NextOffset = &next
	}
	return result
}

func (q SearchQuery) normalized() SearchQuery {
	q.FindingsOnly = q.FindingsOnly || q.Rule != ""
	q.Query = strings.ToLower(q.Query)
	q.Path = strings.ToLower(q.Path)
	return q
}

// remaining returns a value for the next file. A filled page has zero limit;
// later files still contribute their counts to the report total.
func (q SearchQuery) remaining(total, returned int) SearchQuery {
	q.Offset = max(0, q.Offset-total)
	q.Limit -= returned
	return q
}

// searchFile returns new page storage and the full count. A zero remaining
// limit still counts matches in later files without collecting their rows.
func searchFile(fileIndex int, file domain.File, query SearchQuery) ([]SearchMatch, int) {
	if !strings.Contains(strings.ToLower(file.Path), query.Path) {
		return nil, 0
	}
	var matches []SearchMatch
	total := 0
	for functionIndex, function := range file.Functions {
		findings := searchFindings(function, query.Rule)
		if !query.matches(file.Path, function, findings) {
			continue
		}
		total++
		if total <= query.Offset || len(matches) >= query.Limit {
			continue
		}
		matches = append(matches, SearchMatch{
			FileIndex: fileIndex, FunctionIndex: functionIndex, Path: file.Path,
			Package: function.Package, Name: function.Name, Line: function.Line,
			EndLine: function.EndLine, Column: function.Column,
			Complexity: function.Complexity, CognitiveComplexity: function.CognitiveComplexity,
			Findings: findings,
		})
	}
	return matches, total
}

func (q SearchQuery) matches(path string, function domain.Function, findings []SearchFinding) bool {
	if function.Complexity < q.MinComplexity || function.CognitiveComplexity < q.MinCognitiveComplexity {
		return false
	}
	if q.FindingsOnly && len(findings) == 0 {
		return false
	}
	return q.Query == "" || q.textMatches(path, function, findings)
}

func (q SearchQuery) textMatches(path string, function domain.Function, findings []SearchFinding) bool {
	text := strings.Join([]string{path, function.Package, function.Name, function.Source}, "\n")
	for _, finding := range findings {
		text += "\n" + finding.RuleID + "\n" + finding.Message
	}
	return strings.Contains(strings.ToLower(text), q.Query)
}

func searchFindings(function domain.Function, rule string) []SearchFinding {
	findings := []SearchFinding{}
	if function.Quality == nil {
		return findings
	}
	for _, diagnostic := range function.Quality.Diagnostics {
		if rule != "" && diagnostic.RuleID != rule {
			continue
		}
		findings = append(findings, SearchFinding{
			RuleID: diagnostic.RuleID, Message: diagnostic.Message,
			Actual: diagnostic.Actual, Limit: diagnostic.Limit,
		})
	}
	return findings
}
