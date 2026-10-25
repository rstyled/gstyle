package guard

import (
	"fmt"
	"path"
	"strings"
)

type Comment struct {
	Text         string
	Line, Column int
}

type Finding struct {
	RuleID  string `json:"rule_id"`
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
}

func (c Config) CheckPath(name string) []Finding {
	var findings []Finding
	for _, rule := range c.Files {
		if rule.rejects(name) {
			findings = append(findings, Finding{RuleID: rule.ID, Path: name, Line: 1, Column: 1,
				Message: fmt.Sprintf("%s files are restricted to allowed directories", path.Ext(name))})
		}
	}
	return findings
}

func (r FileRule) rejects(name string) bool {
	for _, directory := range r.AllowDirectories {
		if strings.HasPrefix(name, directory+"/") {
			return false
		}
	}
	for _, extension := range r.Extensions {
		if strings.EqualFold(path.Ext(name), extension) {
			return true
		}
	}
	return false
}

func (c Comments) Check(name string, current, previous []Comment) []Finding {
	if c.Mode == "" {
		return nil
	}
	remaining := map[string]int{}
	if c.Mode == "added" {
		for _, comment := range previous {
			remaining[comment.Text]++
		}
	}
	var findings []Finding
	for _, comment := range current {
		if remaining[comment.Text] > 0 {
			remaining[comment.Text]--
			continue
		}
		findings = append(findings, Finding{RuleID: "source_comment", Path: name, Line: comment.Line, Column: comment.Column,
			Message: "source comments are not allowed by policy"})
	}
	return findings
}
