package gopatterns

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// A batch bounds transient compiler syntax and type maps. One package family
// (production plus its test variants) is indivisible; large families stand alone.
const sourceBatchBytes int64 = 8 << 20

type sourceFamily struct {
	imports []string
	name    string
	files   []string
	query   string
	bytes   int64
}

func sourcePackageBatches(config *packages.Config, query []string, request sourceVisit) ([][]string, error) {
	metadata := *config
	metadata.Mode = packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports
	metadata.ParseFile = nil
	pkgs, err := packages.Load(&metadata, batchSourceQueries(&metadata, query)...)
	if err != nil {
		return nil, err
	}
	if err := patternPackageErrors(pkgs, config.Dir); err != nil {
		return nil, err
	}
	families := packageSourceFamilies(pkgs)
	if err := verifySourceFamilies(families, request); err != nil {
		return nil, err
	}
	return planSourceBatches(families, request.selected), nil
}

func packageSourceFamilies(pkgs []*packages.Package) []sourceFamily {
	var families []sourceFamily
	indexes := make(map[string]int)
	for _, pkg := range pkgs {
		if testPackageVariant(pkg.ID) {
			continue
		}
		query := sourceFamilyQuery(pkg)
		indexes[pkg.PkgPath] = len(families)
		families = append(families, sourceFamily{name: pkg.PkgPath, query: query, imports: packageImportPaths(pkg)})
	}
	for _, pkg := range pkgs {
		index, ok := indexes[sourceFamilyPath(pkg)]
		if ok {
			families[index].bytes += packageSourceBytes(pkg)
			families[index].files = append(families[index].files, pkg.CompiledGoFiles...)
		}
	}
	return families
}

func sourceFamilyQuery(pkg *packages.Package) string {
	if pkg.PkgPath == "command-line-arguments" {
		return "file=" + pkg.GoFiles[0]
	}
	return pkg.PkgPath
}

func sourceFamilyPath(pkg *packages.Package) string {
	_, variant, ok := strings.Cut(pkg.ID, " [")
	if ok {
		return strings.TrimSuffix(strings.TrimSuffix(variant, "]"), ".test")
	}
	return strings.TrimSuffix(pkg.PkgPath, ".test")
}

func packageSourceBytes(pkg *packages.Package) int64 {
	var total int64
	for _, path := range pkg.CompiledGoFiles {
		info, err := os.Stat(filepath.Clean(path))
		if err == nil {
			total += info.Size()
		}
	}
	return total
}

func packSourceFamilies(families []sourceFamily) [][]string {
	var batches [][]string
	var batch []string
	var size int64
	for _, family := range families {
		if len(batch) > 0 && size+family.bytes > sourceBatchBytes {
			batches = append(batches, batch)
			batch = nil
			size = 0
		}
		batch = append(batch, family.query)
		size += family.bytes
	}
	if len(batch) > 0 {
		batches = append(batches, batch)
	}
	return batches
}

// Keep ordering, selection and packing one policy even when loader metadata
// arrives in a different order. The loader itself need not retain compiler data.
func planSourceBatches(families []sourceFamily, selected map[string]bool) [][]string {
	return packSourceFamilies(selectSourceFamilies(orderedSourceFamilies(families), selected))
}
