package gopatterns

import (
	"golang.org/x/tools/go/packages"
	"slices"
)

type sourceDependencyOrder struct {
	imports map[string][]string
	seen    map[string]bool
	names   []string
}

// Compiler loading rejects import cycles. Marking on entry also makes metadata
// ordering finite before that diagnostic. Unloaded imports are not pulled in.
func (order *sourceDependencyOrder) visit(name string) {
	if order.seen[name] {
		return
	}
	dependencies, found := order.imports[name]
	if !found {
		return
	}
	order.seen[name] = true
	for _, dependency := range slices.Sorted(slices.Values(dependencies)) {
		order.visit(dependency)
	}
	order.names = append(order.names, name)
}

func orderedSourcePackages(pkgs []*packages.Package) []*packages.Package {
	order := sourceDependencyOrder{imports: map[string][]string{}, seen: map[string]bool{}}
	byName := map[string]*packages.Package{}
	for _, pkg := range pkgs {
		byName[pkg.PkgPath] = pkg
		order.imports[pkg.PkgPath] = packageImportPaths(pkg)
	}
	for _, pkg := range pkgs {
		order.visit(pkg.PkgPath)
	}
	result := make([]*packages.Package, 0, len(pkgs))
	for _, name := range order.names {
		result = append(result, byName[name])
	}
	return result
}

func packageImportPaths(pkg *packages.Package) []string {
	names := make([]string, 0, len(pkg.Imports))
	for name, imported := range pkg.Imports {
		if imported != nil && imported.PkgPath != "" {
			name = imported.PkgPath
		}
		names = append(names, name)
	}
	return names
}

func orderedSourceFamilies(families []sourceFamily) []sourceFamily {
	order := sourceDependencyOrder{imports: map[string][]string{}, seen: map[string]bool{}}
	byName := map[string]sourceFamily{}
	for _, family := range families {
		byName[family.name] = family
		order.imports[family.name] = family.imports
	}
	for _, family := range families {
		order.visit(family.name)
	}
	result := make([]sourceFamily, 0, len(families))
	for _, name := range order.names {
		result = append(result, byName[name])
	}
	return result
}
