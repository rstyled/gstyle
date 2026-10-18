// Package sarif writes static analysis findings as SARIF 2.1.0.
package sarif

import (
	"encoding/json"
	"io"
	"net/url"
	"path/filepath"
	"strings"
)

const Schema = "https://docs.oasis-open.org/sarif/sarif/v2.1.0/errata01/os/schemas/sarif-schema-2.1.0.json"

type Finding struct {
	RuleID    string
	Message   string
	Locations []Location
}

type Location struct {
	Path string
	Line int
}

type log struct {
	Schema  string `json:"$schema"`
	Version string `json:"version"`
	Runs    []run  `json:"runs"`
}

type run struct {
	Tool    tool     `json:"tool"`
	Results []result `json:"results"`
}

type tool struct {
	Driver driver `json:"driver"`
}
type driver struct {
	Name           string `json:"name"`
	InformationURI string `json:"informationUri"`
}
type message struct {
	Text string `json:"text"`
}
type result struct {
	RuleID    string     `json:"ruleId"`
	Level     string     `json:"level"`
	Message   message    `json:"message"`
	Locations []physical `json:"locations,omitempty"`
	Related   []physical `json:"relatedLocations,omitempty"`
}
type physical struct {
	PhysicalLocation source `json:"physicalLocation"`
}
type source struct {
	Artifact artifact `json:"artifactLocation"`
	Region   *region  `json:"region,omitempty"`
}
type artifact struct {
	URI string `json:"uri"`
}
type region struct {
	Line int `json:"startLine"`
}

// Write preserves finding order. Additional sites are related locations: SARIF
// permits only one primary location per result.
func Write(output io.Writer, findings []Finding) error {
	results := make([]result, 0, len(findings))
	for _, finding := range findings {
		results = append(results, toResult(finding))
	}
	report := log{Schema, "2.1.0", []run{{tool{driver{"gstyle", "https://github.com/rstyled/gstyle"}}, results}}}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func toResult(finding Finding) result {
	locations := make([]physical, 0, len(finding.Locations))
	for _, location := range finding.Locations {
		if location.Path == "" {
			continue
		}
		locations = append(locations, toPhysical(location))
	}
	if len(locations) == 0 {
		return result{RuleID: finding.RuleID, Level: "warning", Message: message{finding.Message}}
	}
	return result{RuleID: finding.RuleID, Level: "warning", Message: message{finding.Message}, Locations: locations[:1], Related: locations[1:]}
}

func toPhysical(location Location) physical {
	return physical{source{Artifact: artifact{sourceURI(location.Path)}, Region: sourceRegion(location.Line)}}
}

func sourceURI(path string) string {
	normalized := strings.ReplaceAll(filepath.ToSlash(path), "\\", "/")
	scheme := ""
	if filepath.IsAbs(path) {
		scheme = "file"
	}
	uri := url.URL{Scheme: scheme, Path: normalized}
	return uri.String()
}

func sourceRegion(line int) *region {
	if line <= 0 {
		return nil
	}
	return &region{Line: line}
}
