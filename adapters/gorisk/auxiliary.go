package gorisk

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"maps"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/rstyled/gstyle/domain/changerisk"
)

func ExtractRiskSource(ctx context.Context, root string, changed []string) (changerisk.Snapshot, error) {
	auxiliary, uncovered := extractAuxiliary(root, changed)
	paths, message, err := ExistingPaths(root, uncovered)
	if err != nil {
		return changerisk.Snapshot{}, err
	}
	source, err := ExtractSource(ctx, root, paths)
	if err != nil {
		return source, err
	}
	maps.Copy(auxiliary.PreservedDeclarations, source.PreservedDeclarations)
	source.PreservedDeclarations = auxiliary.PreservedDeclarations
	source.QualityBaseline = auxiliary.QualityBaseline
	source.AnalysisError = strings.TrimSpace(source.AnalysisError + "\n" + message)
	return source, nil
}

func extractAuxiliary(root string, changed []string) (changerisk.Snapshot, []string) {
	result := changerisk.Snapshot{PreservedDeclarations: map[string]map[string]string{}}
	var uncovered []string
	for _, path := range changed {
		if !includeAuxiliary(&result, root, path) {
			uncovered = append(uncovered, path)
		}
	}
	return result, uncovered
}

func includeAuxiliary(result *changerisk.Snapshot, root, path string) bool {
	content, ok := readAuxiliary(root, path)
	if !ok {
		return false
	}
	if path == "quality-baseline.json" {
		result.QualityBaseline = qualityBaseline(content)
		return result.QualityBaseline != nil
	}
	if !testFile(path, content) {
		return false
	}
	result.PreservedDeclarations[path] = map[string]string{"": string(content)}
	return true
}

func readAuxiliary(root, path string) ([]byte, bool) {
	if !auxiliaryCandidate(root, path) {
		return nil, false
	}
	current, ok := auxiliaryPath(root, path)
	if !ok {
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

func auxiliaryPath(root, path string) (string, bool) {
	current := root
	var info fs.FileInfo
	var err error
	for _, part := range strings.Split(path, "/") {
		current = filepath.Join(current, part)
		info, err = os.Lstat(current)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return "", false
		}
	}
	return current, info.Mode().IsRegular() && info.Mode().Perm()&0111 == 0
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
	if baseline.Version != 1 || baseline.Functions <= 0 || baseline.Tolerance < 0 || len(baseline.Rules) == 0 {
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
