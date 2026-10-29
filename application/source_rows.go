package application

type sourceRowKey struct {
	findings        *findingIndex
	annotations     *Annotation
	annotationCount int
	file, function  int
	width           int
	patterns        bool
}

type sourceRowCounter struct {
	key      sourceRowKey
	prefix   []int
	findings map[int][]string
}

func (c sourceRowCounter) count(start, end int) int {
	if end < start {
		return 0
	}
	first := min(max(start, 0), len(c.prefix)-1)
	last := min(max(end+1, first), len(c.prefix)-1)
	return c.prefix[last] - c.prefix[first]
}

func (m Model) ensureSourceRows() Model {
	m = m.ensureFindingIndex()
	key := m.sourceRowKey()
	if m.sourceRows != nil && m.sourceRows.key == key {
		return m
	}
	counter := m.buildSourceRows(key)
	m.sourceRows = &counter
	return m
}

func (m Model) sourceRowKey() sourceRowKey {
	key := sourceRowKey{
		findings: m.findingLookup, annotationCount: len(m.annotations),
		file: m.fileIndex, function: m.functionIndex,
		width: paneContentWidth(m.sourcePaneWidth()), patterns: m.showPatterns,
	}
	if len(m.annotations) > 0 {
		key.annotations = &m.annotations[0]
	}
	return key
}

func (m Model) buildSourceRows(key sourceRowKey) sourceRowCounter {
	function, _ := m.selectedFunction()
	count := max(sourceLineCount(function.Source), 1)
	counter := sourceRowCounter{key: key, prefix: make([]int, count+1), findings: make(map[int][]string)}
	annotations := m.visibleAnnotations()
	for index := range count {
		line := function.Line + index
		rows := sourcePatternRows(m.findingsStartingAt(line), key.width)
		counter.findings[line] = rows
		extra := sourceDeletedRowCount(function.DiffLines, line, line) + sourceAnnotationRowCount(annotations, line, line, key.width)
		counter.prefix[index+1] = counter.prefix[index] + 1 + len(rows) + extra
	}
	counter.prefix[count] += sourceDeletedRowCount(function.DiffLines, function.EndLine+1, function.EndLine+1)
	return counter
}

func (m Model) sourceRowsAtWidth(width int) sourceRowCounter {
	counter := m.sourceRowCounter()
	if counter.key.width == width {
		return counter
	}
	key := counter.key
	key.width = width
	return m.buildSourceRows(key)
}
