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

	"github.com/rstyled/gstyle/domain/changerisk"
	"github.com/rstyled/gstyle/domain/pdg"
)

func sourceDeclarations(sources *pdg.SourceContext, paths []string, functions []changerisk.Function) (changerisk.Snapshot, error) {
	result := changerisk.Snapshot{Declarations: map[string]string{}, PreservedDeclarations: map[string]map[string]string{}}
	identities := declarationIdentities(functions)
	var missing []string
	for _, path := range paths {
		_, content, ok := sources.Source(sources.Reference(filepath.Join(sources.Root, path)))
		if !ok {
			missing = append(missing, path)
			continue
		}
		fingerprint, helpers, err := declarationFacts(path, content, identities)
		if err != nil {
			return result, err
		}
		result.Declarations[path] = fingerprint
		result.PreservedDeclarations[path] = helpers
	}
	if len(missing) > 0 {
		result.AnalysisError = "changed Go sources were not compiled for analysis: " + strings.Join(missing, ", ")
	}
	return result, nil
}

func declarationFacts(path, content string, functions map[declarationBody]string) (string, map[string]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, content, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return "", nil, err
	}
	helpers, err := separatePrivateDeclarations(fset, file, []byte(content), functions)
	if err != nil {
		return "", nil, err
	}
	for _, declaration := range file.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok {
			function.Body = nil
		}
	}
	var printed strings.Builder
	if err := printer.Fprint(&printed, fset, file); err != nil {
		return "", nil, err
	}
	fingerprint, err := sourceTokens([]byte(printed.String()), scanner.ScanComments)
	return fingerprint + rawSourceComments(fset, file, content), helpers, err
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
