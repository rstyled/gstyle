package gopatterns

import (
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Each file= query starts a separate go list process. Resolve active package
// files in one metadata request, then batch those exact package directories.
// Unresolved and build-excluded files retain the original file= semantics.
func batchSourceQueries(config *packages.Config, query []string) []string {
	directories := queryDirectories(query)
	if len(directories) < 2 {
		return query
	}
	metadata := *config
	metadata.Mode, metadata.Tests, metadata.ParseFile = packages.NeedName|packages.NeedFiles, false, nil
	pkgs, err := packages.Load(&metadata, directories...)
	if err != nil {
		return query
	}
	files := activePackageFiles(pkgs)
	return resolvedSourceQueries(query, files)
}

func queryDirectories(query []string) []string {
	var directories []string
	seen := make(map[string]bool)
	for _, item := range query {
		file, ok := strings.CutPrefix(item, "file=")
		if !ok {
			continue
		}
		directory := filepath.Dir(file)
		if !seen[directory] {
			directories = append(directories, directory)
			seen[directory] = true
		}
	}
	return directories
}

func activePackageFiles(pkgs []*packages.Package) map[string]bool {
	files := make(map[string]bool)
	for _, pkg := range pkgs {
		if len(pkg.Errors) != 0 {
			continue
		}
		for _, path := range pkg.GoFiles {
			files[path] = true
		}
	}
	return files
}

func resolvedSourceQueries(query []string, files map[string]bool) []string {
	var result []string
	seen := make(map[string]bool)
	for _, item := range query {
		file, ok := strings.CutPrefix(item, "file=")
		if ok && files[file] {
			item = filepath.Dir(file)
		}
		if !seen[item] {
			result = append(result, item)
			seen[item] = true
		}
	}
	return result
}
