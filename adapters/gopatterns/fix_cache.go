package gopatterns

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"

	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
)

// FixExtractor caches canonical package IR, never compiler objects. Changes
// refresh their package families and every in-scope reverse importer.
type FixExtractor struct {
	pool    *pdg.Builder
	sources *pdg.SourceContext
	records map[string]sourcePackageRecord
	order   []string
	changed map[string]bool
}

func (e *FixExtractor) Invalidate(paths []string) {
	if e.changed == nil {
		e.changed = make(map[string]bool)
	}
	for _, path := range paths {
		e.changed[filepath.Clean(path)] = true
	}
}

func (e *FixExtractor) Extract(ctx context.Context, root string, paths []string, revision SourceOverlay) (*Extraction, error) {
	abs, err := absRoot(root)
	if err != nil {
		return nil, err
	}
	query, err := patternsQuery(abs, paths)
	if err != nil {
		return nil, err
	}
	sources, affected, err := e.prepare(abs, query, revision)
	if err != nil {
		return nil, err
	}
	options := sourceLoadOptions{tests: true, snapshots: sources, overlay: revision.Files}
	if err := e.refresh(ctx, paths, options, affected); err != nil {
		*e = FixExtractor{}
		if errors.Is(err, errFixScopeChanged) {
			return e.Extract(ctx, root, paths, revision)
		}
		return nil, err
	}
	e.sources, e.changed = sources, nil
	return e.merge(), nil
}

func (e *FixExtractor) prepare(root string, query []string, revision SourceOverlay) (*pdg.SourceContext, map[string]bool, error) {
	if e.sources == nil {
		sources := pdg.NewSourceContext(root, query)
		sources.Disk, sources.Overlay = revision.Disk, revision.Files
		e.pool, e.records = pdg.NewBuilder(), make(map[string]sourcePackageRecord)
		return sources, nil, nil
	}
	if e.sources.Root != root || !slices.Equal(e.sources.Queries, query) {
		return nil, nil, fmt.Errorf("fix extraction input scope changed")
	}
	e.trackOverlayChanges(revision.Files)
	if err := checkCachedFixSources(e.sources, e.changed); err != nil {
		return nil, nil, err
	}
	affected := affectedSourcePackages(e.records, e.changed)
	reload := packageReloadPaths(e.records, affected)
	sources := e.sources.NextRevision(reload)
	sources.Disk, sources.Overlay = revision.Disk, revision.Files
	tables := *e.pool.Tables
	tables.Sources = sources
	e.pool.Tables = &tables
	return sources, affected, nil
}

func (e *FixExtractor) refresh(ctx context.Context, paths []string, options sourceLoadOptions, affected map[string]bool) error {
	root := options.snapshots.Root
	if err := captureFixConfiguration(options.snapshots, options); err != nil {
		return err
	}
	e.pool.Tables.Sources = options.snapshots
	extraction := &sourceExtraction{root: root, paths: paths, pool: e.pool, identities: make(map[string]string), result: Extraction{Sources: options.snapshots, PackageNames: make(map[string][]string)}, records: e.records}
	config := &packages.Config{Context: ctx, Dir: root, Mode: packages.LoadSyntax, Tests: true, ParseFile: patternSourceParser(extraction.identities, options.snapshots)}
	visit := func(pkgs []*packages.Package) error { return extraction.collect(pkgs) }
	selected := selectedSourceQueries(e.records, affected)
	if err := visitSourcePackages(sourceVisit{config: config, query: options.snapshots.Queries, overlay: options.overlay, selected: selected, known: e.records}, visit); err != nil {
		return err
	}
	if len(e.order) == 0 {
		e.order = append(e.order, extraction.order...)
	}
	return nil
}

func (e *FixExtractor) merge() *Extraction {
	e.sources.Functions, e.sources.References = nil, nil
	result := &Extraction{Sources: e.sources, PackageNames: make(map[string][]string)}
	for _, name := range e.order {
		record := e.records[name]
		appendPackageExtraction(result, record.facts, e.pool.Tables)
		result.Sources.Functions = append(result.Sources.Functions, record.functions...)
		result.Sources.References = append(result.Sources.References, record.references...)
	}
	populateInterfaceNilHits(result.Funcs)
	return result
}

type sourcePackageRecord struct {
	facts      Extraction
	functions  []pdg.SourceFunction
	references []pdg.SourceReference
	files      []string
	imports    []string
	query      string
}

func appendPackageExtraction(result *Extraction, facts Extraction, tables *pdg.Tables) {
	maps.Copy(result.PackageNames, facts.PackageNames)
	for _, fn := range facts.Funcs {
		fn.Pdg.Tables = tables
		fn.TypedNilHits = slices.Clone(fn.TypedNilHits)
		result.Funcs = append(result.Funcs, fn)
	}
	result.DeclarationOrders = append(result.DeclarationOrders, facts.DeclarationOrders...)
	result.OffsetFallbacks = append(result.OffsetFallbacks, facts.OffsetFallbacks...)
	result.AnemicModels = append(result.AnemicModels, facts.AnemicModels...)
	result.MissingIdentities = append(result.MissingIdentities, facts.MissingIdentities...)
	result.DomainServices = append(result.DomainServices, facts.DomainServices...)
}
