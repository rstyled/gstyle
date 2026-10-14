package gopatterns

import (
	"context"

	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
)

type sourceExtraction struct {
	records    map[string]sourcePackageRecord
	order      []string
	root       string
	paths      []string
	pool       *pdg.Builder
	result     Extraction
	identities map[string]string
}

func (e *sourceExtraction) collect(pkgs []*packages.Package) error {
	if e.records != nil {
		return e.collectFamilies(pkgs)
	}
	return e.collectBatch(pkgs)
}

func (e *sourceExtraction) collectBatch(pkgs []*packages.Package) error {
	functionStart, referenceStart := len(e.result.Sources.Functions), len(e.result.Sources.References)
	collectSourceReferences(e.result.Sources, pkgs)
	internSourceMetadataFrom(e.pool, functionStart, referenceStart)
	for path, names := range packageBindingNames(pkgs) {
		e.result.PackageNames[path] = names
	}
	releaseTestCompilerData(pkgs)
	for _, pkg := range productionPatternPackages(pkgs) {
		if err := e.collectPackage(pkg); err != nil {
			return err
		}
		pkg.Syntax, pkg.TypesInfo = nil, nil
	}
	return nil
}

func (e *sourceExtraction) collectPackage(pkg *packages.Package) error {
	funcs, err := packagePdgsWithSources(pkg, e.root, e.pool, e.identities)
	if err != nil {
		return err
	}
	projectSourceParameterFlow(e.result.Sources, funcs)
	e.result.Funcs = append(e.result.Funcs, funcs...)
	e.result.DeclarationOrders = append(e.result.DeclarationOrders, findDeclarationOrders(pkg, e.root, e.paths, e.result.Sources)...)
	e.result.OffsetFallbacks = append(e.result.OffsetFallbacks, findOffsetFallbacks(pkg, e.root, e.paths, e.result.Sources)...)
	e.result.AnemicModels = append(e.result.AnemicModels, findAnemicModels(pkg, e.root)...)
	e.result.MissingIdentities = append(e.result.MissingIdentities, findMissingIdentities(pkg, e.root)...)
	e.result.DomainServices = append(e.result.DomainServices, findDomainServices(pkg, e.root)...)
	return nil
}

func extractBoundedFix(ctx context.Context, root string, paths []string, options sourceLoadOptions) (*Extraction, error) {
	if err := captureFixConfiguration(options.snapshots, options); err != nil {
		return nil, err
	}
	pool := pdg.NewBuilder()
	pool.Tables.Sources = options.snapshots
	extraction := &sourceExtraction{root: root, paths: paths, pool: pool, identities: make(map[string]string), result: Extraction{Sources: options.snapshots, PackageNames: make(map[string][]string)}}
	config := &packages.Config{Context: ctx, Dir: root, Mode: packages.LoadSyntax, Tests: true, ParseFile: patternSourceParser(extraction.identities, options.snapshots)}
	if err := visitSourcePackages(sourceVisit{config: config, query: options.snapshots.Queries, overlay: options.overlay}, extraction.collect); err != nil {
		return nil, err
	}
	populateInterfaceNilHits(extraction.result.Funcs)
	return &extraction.result, nil
}

// Inventories already own their strings and ranges. Test syntax and binding
// maps have no downstream role once caller references are captured.
func releaseTestCompilerData(pkgs []*packages.Package) {
	for _, pkg := range pkgs {
		if testPackageVariant(pkg.ID) {
			pkg.Syntax, pkg.TypesInfo = nil, nil
		}
	}
}
