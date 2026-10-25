package repoguard

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"go/scanner"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/rstyled/gstyle/adapters/gitchanged"
	"github.com/rstyled/gstyle/domain/guard"
)

type repository struct {
	root  string
	paths []string
	base  string
}

func (r repository) inspect(ctx context.Context, config guard.Config) ([]guard.Finding, error) {
	files, err := fileList(ctx, r.root, r.paths)
	if err != nil {
		return nil, err
	}
	findings := []guard.Finding{}
	for _, name := range files {
		if _, err := os.Lstat(filepath.Join(r.root, name)); errors.Is(err, fs.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, err
		}
		findings = append(findings, config.CheckPath(filepath.ToSlash(name))...)
	}
	comments, err := r.inspectComments(ctx, files, config.Comments)
	if err != nil {
		return nil, err
	}
	findings = append(findings, comments...)
	slices.SortFunc(findings, compareFindings)
	return findings, nil
}

func compareFindings(a, b guard.Finding) int {
	return cmp.Or(cmp.Compare(a.Path, b.Path), cmp.Compare(a.Line, b.Line), cmp.Compare(a.Column, b.Column), cmp.Compare(a.RuleID, b.RuleID))
}

func (r repository) inspectComments(ctx context.Context, files []string, config guard.Comments) ([]guard.Finding, error) {
	if config.Mode == "" {
		return nil, nil
	}
	if config.Mode == "all" {
		return r.inspectCommentFiles(ctx, "", allFiles(files), config)
	}
	resolved, err := gitchanged.ResolveBase(r.root, r.base)
	if err != nil {
		return nil, err
	}
	changes, err := changedFiles(ctx, r.root, resolved, r.paths)
	if err != nil {
		return nil, err
	}
	return r.inspectCommentFiles(ctx, resolved, changes, config)
}

func allFiles(files []string) map[string]string {
	result := map[string]string{}
	for _, file := range files {
		result[file] = ""
	}
	return result
}

func (r repository) inspectCommentFiles(ctx context.Context, base string, files map[string]string, config guard.Comments) ([]guard.Finding, error) {
	var findings []guard.Finding
	for name, previous := range files {
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		current, err := os.ReadFile(filepath.Join(r.root, name))
		if err != nil {
			return nil, err
		}
		old, err := previousSource(ctx, r.root, base, previous)
		if err != nil {
			return nil, err
		}
		added, err := checkComments(name, current, old, config)
		if err != nil {
			return nil, err
		}
		findings = append(findings, added...)
	}
	return findings, nil
}

func previousSource(ctx context.Context, root, base, name string) ([]byte, error) {
	if !strings.HasSuffix(name, ".go") || base == "" {
		return nil, nil
	}
	return git(ctx, root, "show", base+":"+name)
}

func checkComments(name string, current, old []byte, config guard.Comments) ([]guard.Finding, error) {
	comments, err := scanComments(name, current)
	if err != nil {
		return nil, err
	}
	previous, err := scanComments(name, old)
	if err != nil {
		return nil, err
	}
	return config.Check(filepath.ToSlash(name), comments, previous), nil
}

func scanComments(name string, source []byte) ([]guard.Comment, error) {
	file := token.NewFileSet().AddFile(name, -1, len(source))
	var scan scanner.Scanner
	var scanError error
	scan.Init(file, source, func(position token.Position, message string) {
		scanError = fmt.Errorf("scan %s: %s", position, message)
	}, scanner.ScanComments)
	var comments []guard.Comment
	for {
		pos, kind, text := scan.Scan()
		if kind == token.EOF {
			return comments, scanError
		}
		if kind == token.COMMENT {
			position := file.PositionFor(pos, false)
			comments = append(comments, guard.Comment{Text: text, Line: position.Line, Column: position.Column})
		}
	}
}
