package gopatterns

import (
	"os"

	"github.com/rstyled/gstyle/domain/pdg"
)

func (e *FixExtractor) trackOverlayChanges(overlay map[string][]byte) {
	for path, content := range overlay {
		if e.sources.Check(path, content) != nil {
			e.Invalidate([]string{path})
		}
	}
	for path := range e.sources.Overlay {
		if _, ok := overlay[path]; !ok {
			e.Invalidate([]string{path})
		}
	}
}

func affectedSourcePackages(records map[string]sourcePackageRecord, changed map[string]bool) map[string]bool {
	affected := make(map[string]bool)
	for name, record := range records {
		if sourceFilesChanged(record.files, changed) {
			affected[name] = true
		}
	}
	for expandAffectedPackages(records, affected) {
	}
	return affected
}

func sourceFilesChanged(files []string, changed map[string]bool) bool {
	for _, path := range files {
		if changed[path] {
			return true
		}
	}
	return false
}

func expandAffectedPackages(records map[string]sourcePackageRecord, affected map[string]bool) bool {
	expanded := false
	for name, record := range records {
		if affected[name] {
			continue
		}
		if sourceFilesChanged(record.imports, affected) {
			affected[name] = true
			expanded = true
		}
	}
	return expanded
}

func packageReloadPaths(records map[string]sourcePackageRecord, affected map[string]bool) map[string]bool {
	paths := make(map[string]bool)
	for name := range affected {
		for _, path := range records[name].files {
			paths[path] = true
		}
	}
	return paths
}

func checkCachedFixSources(sources *pdg.SourceContext, changed map[string]bool) error {
	if sources.Disk != nil {
		return CheckFixSources(sources.Disk)
	}
	for _, path := range sources.Paths() {
		if changed[path] {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := sources.Check(path, content); err != nil {
			return err
		}
	}
	return checkMissingFixSources(sources)
}
