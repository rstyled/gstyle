package gopatterns

import (
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"strings"

	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
)

// CheckFixSources rejects edits against a different extraction revision.
func CheckFixSources(sources *pdg.SourceContext) error {
	if sources == nil {
		return fmt.Errorf("fix source context is missing")
	}
	if sources.Disk != nil {
		return CheckFixSources(sources.Disk)
	}
	for _, path := range sources.Paths() {
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := sources.Check(path, content); err != nil {
			return err
		}
	}
	return checkMissingFixSources(sources)
}

// ValidateFixOverlay checks the complete proposed result before any writes.
func ValidateFixOverlay(ctx context.Context, sources *pdg.SourceContext, overlay map[string][]byte) error {
	if err := CheckFixSources(sources); err != nil {
		return err
	}
	if err := checkOverlayComments(sources, overlay); err != nil {
		return err
	}
	config := &packages.Config{Context: ctx, Dir: sources.Root, Mode: packages.LoadTypes, Tests: true}
	pkgs, err := loadSourcePackages(config, sources.Queries, mergedFixOverlay(sources.Overlay, overlay))
	if err != nil {
		return fmt.Errorf("validate fixes: %w", err)
	}
	return fixPackageErrors(pkgs)
}

func fixPackageErrors(pkgs []*packages.Package) error {
	var errors []string
	for _, pkg := range pkgs {
		for _, err := range pkg.Errors {
			errors = append(errors, err.Error())
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("fix result does not compile:\n%s", strings.Join(errors, "\n"))
	}
	return nil
}

func checkOverlayComments(sources *pdg.SourceContext, overlay map[string][]byte) error {
	for path, content := range overlay {
		original, ok := sources.Snapshot(sources.Reference(path))
		if !ok {
			return fmt.Errorf("source evidence is missing: %s", path)
		}
		if err := checkComments(path, original, content); err != nil {
			return err
		}
	}
	return nil
}

func checkComments(path string, original pdg.SourceSnapshot, content []byte) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, content, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return fmt.Errorf("parse fixed source: %w", err)
	}
	if !checkCommentOwners(original, sourceCommentOwners(fset, file, content), content) {
		return fmt.Errorf("fix moves a declaration comment in %s", path)
	}
	counts := make(map[string]int)
	for _, span := range sourceCommentRanges(fset, file, content) {
		counts[string(content[span.Start:span.End])]++
	}
	for _, span := range original.Comments {
		text := original.Content[span.Start:span.End]
		if counts[text] == 0 {
			return fmt.Errorf("fix removes or changes a comment in %s", path)
		}
		counts[text]--
	}
	return nil
}

func checkMissingFixSources(sources *pdg.SourceContext) error {
	for _, path := range sources.MissingPaths() {
		_, err := os.Stat(path)
		if err == nil {
			return fmt.Errorf("build input appears after extraction: %s", path)
		}
		if !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func mergedFixOverlay(previous, next map[string][]byte) map[string][]byte {
	result := make(map[string][]byte, len(previous)+len(next))
	for path, content := range previous {
		result[path] = content
	}
	for path, content := range next {
		result[path] = content
	}
	return result
}
