package application

import "slices"

type findingFile struct {
	markers []PatternMarker
	count   int
	starts  map[int][]PatternMarker
}

type findingIndex struct {
	patterns       map[string]findingFile
	quality        map[string]findingFile
	targets        []reviewTarget
	qualityTargets []reviewTarget
	qualityCount   int
}

func (m Model) indexFindings() Model {
	if m.qualityMarkers == nil {
		m.qualityMarkers = qualityMarkersFromReport(m.report)
	}
	all := slices.Concat(m.patternMarkers, m.qualityMarkers)
	m.findingLookup = &findingIndex{
		patterns:       m.indexFindingFiles(m.patternMarkers),
		quality:        m.indexFindingFiles(m.qualityMarkers),
		targets:        m.collectFindingTargets(all),
		qualityTargets: m.collectFindingTargets(m.qualityMarkers),
		qualityCount:   qualityFindingCount(m.qualityMarkers),
	}
	return m
}

func (m Model) ensureFindingIndex() Model {
	if m.findingLookup != nil {
		return m
	}
	return m.indexFindings()
}

func (m Model) indexFindingFiles(markers []PatternMarker) map[string]findingFile {
	paths := make(map[string][]PatternMarker)
	for _, marker := range markers {
		paths[marker.Path] = append(paths[marker.Path], marker)
	}
	files := make(map[string]findingFile, len(paths))
	for path, group := range paths {
		files[path] = newFindingFile(group)
	}
	for _, file := range m.report.Files {
		var matching []PatternMarker
		for _, marker := range markers {
			if patternPathMatches(marker.Path, file.Path) {
				matching = append(matching, marker)
			}
		}
		files[file.Path] = newFindingFile(matching)
	}
	return files
}

func newFindingFile(markers []PatternMarker) findingFile {
	starts := make(map[int][]PatternMarker)
	for _, marker := range markers {
		starts[marker.Line] = append(starts[marker.Line], marker)
	}
	return findingFile{markers: markers, count: len(uniquePatternMarkers(markers)), starts: starts}
}

func (m Model) patternFile(path string) findingFile {
	m = m.ensureFindingIndex()
	return m.findingLookup.patterns[path]
}

func (m Model) findingsStartingAt(line int) []PatternMarker {
	m = m.ensureFindingIndex()
	path := m.selectedFilePath()
	markers := m.findingLookup.quality[path].starts[line]
	if !m.showPatterns {
		return markers
	}
	return slices.Concat(m.findingLookup.patterns[path].starts[line], markers)
}

func (m Model) hasFindingAtLine(line int) bool {
	m = m.ensureFindingIndex()
	path := m.selectedFilePath()
	if len(m.findingLookup.quality[path].starts[line]) > 0 {
		return true
	}
	return m.showPatterns && slices.ContainsFunc(m.findingLookup.patterns[path].markers, func(marker PatternMarker) bool { return marker.covers(path, line) })
}
