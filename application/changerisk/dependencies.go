package changerisk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

type module struct {
	Path, Version, GoVersion string
	Replace                  *replacement
}

type replacement struct{ Path, Version string }

type dependencyPackage struct {
	ImportPath string
	Imports    []string
	Dir        string
	EmbedFiles []string
	Module     *struct{ Main bool }
}

func dependencies(ctx context.Context, root string) ([]string, error) {
	data, err := command(ctx, root, "git", "ls-files", "-z", "--", "go.mod", "**/go.mod")
	if err != nil {
		return nil, err
	}
	files := projectModuleFiles(fields(data))
	if len(files) == 0 {
		return nil, fmt.Errorf("no Go modules found")
	}
	var result []string
	for _, file := range files {
		items, err := moduleDependencies(ctx, filepath.Join(root, filepath.Dir(file)))
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			result = append(result, filepath.ToSlash(filepath.Dir(file))+":"+item)
		}
	}
	slices.Sort(result)
	return slices.Compact(result), nil
}

func projectModuleFiles(files []string) []string {
	var result []string
	for _, file := range files {
		if !fixtureModule(file) {
			result = append(result, file)
		}
	}
	return result
}

func fixtureModule(file string) bool {
	for _, component := range strings.Split(filepath.ToSlash(filepath.Dir(file)), "/") {
		if component == "testdata" || component == "vendor" {
			return true
		}
	}
	return false
}

func moduleDependencies(ctx context.Context, dir string) ([]string, error) {
	graph, err := command(ctx, dir, "go", "mod", "graph")
	if err != nil {
		return nil, err
	}
	modules, err := command(ctx, dir, "go", "list", "-mod=readonly", "-m", "-json", "all")
	if err != nil {
		return nil, err
	}
	items, err := moduleRecords(modules)
	if err != nil {
		return nil, err
	}
	return appendPackageDependencies(ctx, dir, append(items, strings.Split(strings.TrimSpace(string(graph)), "\n")...))
}

func moduleRecords(data []byte) ([]string, error) {
	var items []string
	decoder := json.NewDecoder(bytes.NewReader(data))
	for {
		var value module
		if err := decoder.Decode(&value); errors.Is(err, io.EOF) {
			if len(items) == 0 {
				return nil, fmt.Errorf("module output contains no records")
			}
			return items, nil
		} else if err != nil {
			return nil, err
		}
		if !validModule(value) {
			return nil, fmt.Errorf("module output is missing its module or replacement path")
		}
		items = append(items, moduleIdentity(value))
	}
}

func validModule(value module) bool {
	return value.Path != "" && (value.Replace == nil || value.Replace.Path != "")
}

func moduleIdentity(value module) string {
	identity := fmt.Sprintf("module:%q:%q:%q", value.Path, value.Version, value.GoVersion)
	if value.Replace == nil {
		return identity
	}
	return identity + fmt.Sprintf(":replace:%q:%q", value.Replace.Path, value.Replace.Version)
}

func appendPackageDependencies(ctx context.Context, dir string, items []string) ([]string, error) {
	data, err := command(ctx, dir, "go", "list", "-mod=readonly", "-deps", "-test", "-json", "./...")
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	for {
		var pkg dependencyPackage
		if err := decoder.Decode(&pkg); errors.Is(err, io.EOF) {
			return items, nil
		} else if err != nil {
			return nil, err
		}
		records, err := packageRecords(dir, pkg)
		if err != nil {
			return nil, err
		}
		items = append(items, records...)
	}
}

func packageRecords(root string, pkg dependencyPackage) ([]string, error) {
	if pkg.ImportPath == "" {
		return nil, fmt.Errorf("package output is missing its import path")
	}
	resources, err := packageResources(root, pkg)
	if err != nil {
		return nil, err
	}
	return append(packageImports(pkg), resources...), nil
}

func packageResources(root string, pkg dependencyPackage) ([]string, error) {
	if pkg.Module == nil || !pkg.Module.Main {
		return nil, nil
	}
	var records []string
	for _, name := range pkg.EmbedFiles {
		relative, err := filepath.Rel(root, filepath.Join(pkg.Dir, name))
		if err != nil || !fs.ValidPath(filepath.ToSlash(relative)) {
			return nil, fmt.Errorf("embedded resource is outside the project module")
		}
		records = append(records, "embed:"+filepath.ToSlash(relative))
	}
	return records, nil
}

func embeddedScope(changed, records []string) string {
	paths := map[string]bool{}
	for _, name := range changed {
		paths[name] = true
	}
	var affected []string
	for _, record := range records {
		scope, name, found := strings.Cut(record, ":embed:")
		if found && paths[path.Join(scope, name)] {
			affected = append(affected, path.Join(scope, name))
		}
	}
	if len(affected) == 0 {
		return ""
	}
	slices.Sort(affected)
	return "changed embedded resources need review: " + strings.Join(slices.Compact(affected), ", ")
}

func packageImports(pkg dependencyPackage) []string {
	if pkg.Module == nil || !pkg.Module.Main {
		return nil
	}
	var items []string
	for _, imported := range pkg.Imports {
		items = append(items, "import:"+pkg.ImportPath+" -> "+imported)
	}
	return items
}
