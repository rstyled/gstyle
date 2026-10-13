package gopatterns

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"

	"golang.org/x/tools/go/packages"
)

// Go's build overlay invalidates export data through the actual import graph.
// Config.Overlay instead forces go/packages to check every dependency from
// source, including unchanged standard-library and third-party packages.
func loadSourcePackages(config *packages.Config, query []string, overlay map[string][]byte) ([]*packages.Package, error) {
	if len(overlay) == 0 {
		return packages.Load(config, batchSourceQueries(config, query)...)
	}
	directory, err := os.MkdirTemp("", "gstyle-source-overlay-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(directory)
	path, err := writeBuildOverlay(directory, overlay)
	if err != nil {
		return nil, err
	}
	next := *config
	next.BuildFlags = append(append([]string(nil), config.BuildFlags...), "-overlay="+path)
	next.ParseFile = overlaySourceParser(config.ParseFile, overlay)
	return packages.Load(&next, batchSourceQueries(&next, query)...)
}

func writeBuildOverlay(directory string, overlay map[string][]byte) (string, error) {
	replacements := make(map[string]string, len(overlay))
	for path, content := range overlay {
		replacement := filepath.Join(directory, strconv.Itoa(len(replacements))+".go")
		if err := os.WriteFile(replacement, content, 0600); err != nil {
			return "", err
		}
		replacements[path] = replacement
	}
	content, err := json.Marshal(struct{ Replace map[string]string }{replacements})
	if err != nil {
		return "", err
	}
	path := filepath.Join(directory, "overlay.json")
	return path, os.WriteFile(path, content, 0600)
}

func overlaySourceParser(parse func(*token.FileSet, string, []byte) (*ast.File, error), overlay map[string][]byte) func(*token.FileSet, string, []byte) (*ast.File, error) {
	return func(fset *token.FileSet, path string, source []byte) (*ast.File, error) {
		if content, ok := overlay[path]; ok {
			source = content
		}
		if parse != nil {
			return parse(fset, path, source)
		}
		return parser.ParseFile(fset, path, source, parser.AllErrors|parser.ParseComments)
	}
}
