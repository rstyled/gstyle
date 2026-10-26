package gorisk

import (
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/rstyled/gstyle/domain/changerisk"
)

type Auxiliary struct {
	Uncovered []string
	Tests     map[string]changerisk.TestFile
	Baseline  *changerisk.QualityBaseline
}

func ExtractAuxiliary(root string, changed []string) Auxiliary {
	result := Auxiliary{Tests: map[string]changerisk.TestFile{}}
	for _, path := range changed {
		if !result.include(root, path) {
			result.Uncovered = append(result.Uncovered, path)
		}
	}
	return result
}

func (result *Auxiliary) include(root, path string) bool {
	content, ok := readAuxiliary(root, path)
	if !ok {
		return false
	}
	if path == "quality-baseline.json" {
		result.Baseline = qualityBaseline(content)
		return result.Baseline != nil
	}
	file, ok := testFile(path, content)
	if ok {
		result.Tests[path] = file
	}
	return ok
}

func readAuxiliary(root, path string) ([]byte, bool) {
	if !auxiliaryCandidate(root, path) {
		return nil, false
	}
	current, ok := auxiliaryPath(root, path)
	if !ok {
		return nil, false
	}
	if !regularAuxiliaryFile(current) {
		return nil, false
	}
	content, err := os.ReadFile(current)
	return content, err == nil
}

func auxiliaryCandidate(root, path string) bool {
	if !fs.ValidPath(path) || strings.Contains(path, "\\") {
		return false
	}
	if nestedModule(root, path) {
		return false
	}
	return path == "quality-baseline.json" || strings.HasSuffix(path, "_test.go")
}

func regularAuxiliaryFile(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0111 == 0
}

func auxiliaryPath(root, path string) (string, bool) {
	current := root
	for _, part := range strings.Split(path, "/") {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return "", false
		}
	}
	return current, true
}

func qualityBaseline(content []byte) *changerisk.QualityBaseline {
	var result changerisk.QualityBaseline
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return nil
	}
	if decoder.Decode(new(any)) != io.EOF {
		return nil
	}
	if !validQualityBaseline(result) {
		return nil
	}
	return &result
}

func validQualityBaseline(baseline changerisk.QualityBaseline) bool {
	if baseline.Version != 1 || baseline.Functions <= 0 || baseline.Tolerance < 0 {
		return false
	}
	if len(baseline.Rules) == 0 {
		return false
	}
	return validBaselineRules(baseline)
}

func validBaselineRules(baseline changerisk.QualityBaseline) bool {
	for _, rule := range baseline.Rules {
		if !validBaselineRule(rule, baseline.Functions) {
			return false
		}
	}
	return true
}

func validBaselineRule(rule changerisk.BaselineRule, functions int) bool {
	if rule.Limit < 0 || rule.Excess < 0 || rule.Rate < 0 {
		return false
	}
	expected := float64(rule.Excess) / float64(functions) * 1000
	return math.Abs(rule.Rate-expected) <= 1e-9
}
