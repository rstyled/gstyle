package gorisk

import (
	"go/ast"
	"go/parser"
	"go/printer"
	"go/scanner"
	"go/token"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/rstyled/gstyle/domain/pdg"
)

func sourceDeclarations(sources *pdg.SourceContext, paths []string) (map[string]string, string, error) {
	declarations := map[string]string{}
	var missing []string
	for _, path := range paths {
		_, content, ok := sources.Source(sources.Reference(filepath.Join(sources.Root, path)))
		if !ok {
			missing = append(missing, path)
			continue
		}
		fingerprint, err := declarationFingerprint(path, content)
		if err != nil {
			return nil, "", err
		}
		declarations[path] = fingerprint
	}
	message := ""
	if len(missing) > 0 {
		message = "changed Go sources were not compiled for analysis: " + strings.Join(missing, ", ")
	}
	return declarations, message, nil
}

func declarationFingerprint(path, content string) (string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, content, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return "", err
	}
	for _, declaration := range file.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok {
			function.Body = nil
		}
	}
	var printed strings.Builder
	if err := printer.Fprint(&printed, fset, file); err != nil {
		return "", err
	}
	fingerprint, err := sourceTokens([]byte(printed.String()), scanner.ScanComments)
	return fingerprint + rawSourceComments(fset, file, content), err
}

func rawSourceComments(fset *token.FileSet, file *ast.File, content string) string {
	var result strings.Builder
	for _, group := range file.Comments {
		for _, comment := range group.List {
			start := fset.PositionFor(comment.Pos(), false).Offset
			result.WriteByte(0)
			result.WriteString(strconv.Quote(rawSourceComment(content[start:])))
		}
	}
	return result.String()
}

func rawSourceComment(source string) string {
	if strings.HasPrefix(source, "//") {
		text, _, _ := strings.Cut(source, "\n")
		return text
	}
	text, _, _ := strings.Cut(source, "*/")
	return text + "*/"
}

func inertDocumentation(path string) bool {
	if path == "README.md" {
		return true
	}
	if !strings.HasPrefix(path, "docs/") || filepath.Ext(path) != ".md" {
		return false
	}
	return inertDocumentationComponents(path)
}

func inertDocumentationComponents(path string) bool {
	for _, component := range strings.Split(strings.ToLower(path), "/") {
		if strings.HasPrefix(component, ".") || slices.Contains([]string{
			"agent", "agents", "skill", "skills", "agent.md", "agents.md", "skill.md", "skills.md", "claude.md", "codex.md", "gemini.md",
		}, component) {
			return false
		}
	}
	return true
}
