package gopatterns

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rstyled/gstyle/domain/pdg"
)

// A signature rewrite needs all same-package source, not just active callers.
// Build-excluded Go and assembly files require evidence this producer lacks.
func checkSignatureCoverage(context *pdg.SourceContext, targets map[string]pdg.SourceFunction) error {
	checked := make(map[string]bool)
	for _, target := range targets {
		if err := checkSignatureDirectives(context, target); err != nil {
			return err
		}
		snapshot, _ := context.Snapshot(target.Source)
		dir := filepath.Dir(snapshot.Path)
		if checked[dir] {
			continue
		}
		checked[dir] = true
		if err := checkSourceDirectory(context, dir); err != nil {
			return err
		}
	}
	return nil
}

func checkSourceDirectory(context *pdg.SourceContext, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if unsupportedSourceFile(context, path) {
			return fmt.Errorf("signature reference coverage is incomplete: %s", path)
		}
	}
	return nil
}

func unsupportedSourceFile(context *pdg.SourceContext, path string) bool {
	if strings.HasSuffix(path, ".s") {
		return true
	}
	if !strings.HasSuffix(path, ".go") {
		return false
	}
	snapshot, ok := context.Snapshot(context.Reference(path))
	if !ok {
		return true
	}
	for _, span := range snapshot.Comments {
		comment := snapshot.Content[span.Start:span.End]
		if strings.HasPrefix(comment, "//go:linkname") {
			return true
		}
	}
	return false
}

func checkSignatureDirectives(context *pdg.SourceContext, target pdg.SourceFunction) error {
	snapshot, _ := context.Snapshot(target.Source)
	for _, comment := range snapshot.Owners {
		if comment.Owner != "func:"+target.Name {
			continue
		}
		text := snapshot.Content[comment.Range.Start:comment.Range.End]
		if strings.HasPrefix(text, "//go:") {
			return fmt.Errorf("signature rewrite requires unsupported compiler directive evidence: %s", target.Name)
		}
	}
	return nil
}
