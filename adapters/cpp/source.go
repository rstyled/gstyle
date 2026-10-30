package cpp

import (
	"fmt"
	"strings"

	"github.com/rstyled/gstyle/domain/pdg"
)

type sourceFile struct {
	content      string
	lines        []int
	supplemental bool
	units        []int
}

func newSourceFile(content string) *sourceFile {
	if content == "<empty>" {
		content = ""
	}
	file := &sourceFile{content: content, lines: []int{0}}
	unicode := false
	for i, ch := range content {
		if ch > 127 {
			unicode = true
		}
		if ch == '\n' {
			file.lines = append(file.lines, i+1)
		}
	}
	if unicode {
		file.units = utf16Boundaries(content)
	}
	return file
}

func (p *projection) source() {
	p.file = p.doc.Files[p.path()]
	if p.file == nil || p.file.content == "" {
		p.extension("gstyle.cpp:source-span", "unavailable; zero-width function span is a placeholder; node properties retain source coordinates")
		return
	}
	p.code = p.file.content
	context := p.doc.Sources
	ref := context.Reference(p.path())
	if ref == 0 {
		ref = context.Add(p.path(), []byte(p.code), nil)
	}
	p.pool.Tables.Sources = context
	p.graph.Sources[0].Snapshot = ref
	snapshot, _ := context.Snapshot(ref)
	p.graph.Sources[0].ContentIdentity = p.pool.Known(fmt.Sprintf("sha256:%x", snapshot.Identity))
	if p.file.supplemental {
		p.extension("gstyle.cpp:source-origin", "explicit --cpg-source; method code verified against this snapshot")
	}
	if span, ok := p.span(p.method); ok {
		p.graph.Function.Span = span
	} else {
		p.extension("gstyle.cpp:source-span", "method byte range is unavailable; CPG source coordinates are preserved")
	}
}

func (p *projection) span(node *vertex) (pdg.Span, bool) {
	if !p.sameSource(node) {
		return pdg.Span{}, false
	}
	if span, ok := p.offsetSpan(node); ok {
		return span, true
	}
	start, ok := p.file.position(node.number("LINE_NUMBER"), node.number("COLUMN_NUMBER"))
	code := node.text("CODE")
	if !ok || code == "" || !strings.HasPrefix(p.code[start:], code) {
		return pdg.Span{}, false
	}
	return pdg.Span{Source: 1, Start: uint32(start), End: uint32(start + len(code))}, true
}

func (p *projection) sameSource(node *vertex) bool {
	return node.text("FILENAME") == "" || node.text("FILENAME") == p.path()
}

func (p *projection) offsetSpan(node *vertex) (pdg.Span, bool) {
	if p.file == nil {
		return pdg.Span{}, false
	}
	start, end, ok := p.file.offsets(node)
	if !ok {
		return pdg.Span{}, false
	}
	return pdg.Span{Source: 1, Start: uint32(start), End: uint32(end)}, true
}

func (f *sourceFile) position(line, column int) (int, bool) {
	if !f.hasPosition(line, column) {
		return 0, false
	}
	start := f.lines[line-1]
	end := len(f.content)
	if line < len(f.lines) {
		end = f.lines[line] - 1
	}
	if column-1 > end-start {
		return 0, false
	}
	return start + column - 1, true
}

func (f *sourceFile) hasPosition(line, column int) bool {
	return f != nil && line >= 1 && line <= len(f.lines) && column >= 1
}

func methodSource(method *vertex, file *sourceFile) string {
	if source, ok := methodOffsets(method, file); ok {
		return source
	}
	return method.text("CODE")
}

func methodOffsets(method *vertex, file *sourceFile) (string, bool) {
	if file == nil {
		return "", false
	}
	start, end, ok := file.offsets(method)
	if !ok {
		return "", false
	}
	return file.content[start:end], true
}
