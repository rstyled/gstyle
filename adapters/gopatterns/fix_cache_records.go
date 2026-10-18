package gopatterns

import (
	"slices"
	"strings"

	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
)

func (e *sourceExtraction) collectFamilies(pkgs []*packages.Package) error {
	order, grouped := groupSourcePackages(orderedFamilyPackages(pkgs))
	for _, name := range order {
		sources := e.result.Sources
		functionStart, referenceStart := len(sources.Functions), len(sources.References)
		family := &sourceExtraction{program: e.program, root: e.root, paths: e.paths, pool: e.pool, identities: e.identities, result: Extraction{Sources: sources, PackageNames: make(map[string][]string)}}
		if err := family.collectBatch(grouped[name]); err != nil {
			return err
		}
		e.program = family.program
		e.records[name] = recordSourcePackage(grouped[name], family.result, sources.Functions[functionStart:], sources.References[referenceStart:])
		e.order = append(e.order, name)
	}
	return nil
}

func groupSourcePackages(pkgs []*packages.Package) ([]string, map[string][]*packages.Package) {
	var order []string
	grouped := make(map[string][]*packages.Package)
	for _, pkg := range pkgs {
		name := sourceFamilyPath(pkg)
		if grouped[name] == nil {
			order = append(order, name)
		}
		grouped[name] = append(grouped[name], pkg)
	}
	return order, grouped
}

func recordSourcePackage(pkgs []*packages.Package, facts Extraction, functions []pdg.SourceFunction, references []pdg.SourceReference) sourcePackageRecord {
	query := ""
	for _, pkg := range pkgs {
		if !testPackageVariant(pkg.ID) {
			query = sourceFamilyQuery(pkg)
			break
		}
	}
	record := sourcePackageRecord{query: query, facts: facts, functions: slices.Clone(functions), references: slices.Clone(references)}
	record.facts.Sources = nil
	record.files, record.imports = sourcePackageInputs(pkgs)
	slices.Sort(record.files)
	slices.Sort(record.imports)
	return record
}

func sourcePackageInputs(pkgs []*packages.Package) (paths, dependencies []string) {
	files, imports := make(map[string]bool), make(map[string]bool)
	for _, pkg := range pkgs {
		for _, path := range pkg.CompiledGoFiles {
			files[strings.Clone(path)] = true
		}
		for _, imported := range pkg.Imports {
			imports[strings.Clone(sourceFamilyPath(imported))] = true
		}
	}
	for path := range files {
		paths = append(paths, path)
	}
	for name := range imports {
		dependencies = append(dependencies, name)
	}
	return paths, dependencies
}

// Keep test variants in their family while ordering production imports first.
func orderedFamilyPackages(pkgs []*packages.Package) []*packages.Package {
	_, grouped := groupSourcePackages(pkgs)
	var result []*packages.Package
	for _, pkg := range orderedSourcePackages(productionPatternPackages(pkgs)) {
		result = append(result, grouped[pkg.PkgPath]...)
	}
	return result
}
