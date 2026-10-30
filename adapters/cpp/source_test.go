package cpp

import (
	"encoding/json"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestSourceRangeRequiresMatchingBytes(t *testing.T) {
	file := newSourceFile("é int f() {}")
	method := &vertex{Label: "METHOD", Props: map[string]any{"NAME": "f", "CODE": "int f() {}", "FILENAME": "sample.cpp", "OFFSET": json.Number("2"), "OFFSET_END": json.Number("12"), "LINE_NUMBER": json.Number("1"), "COLUMN_NUMBER": json.Number("3")}}
	p := &projection{method: method, file: file, code: file.content, pool: pdg.NewBuilder()}
	if span, ok := p.span(method); !ok || span.Start != 3 || span.End != 13 {
		t.Fatalf("UTF-16 source positions are not converted to verified byte positions: %+v %v", span, ok)
	}
	if got := methodSource(method, file); got != "int f() {}" {
		t.Fatalf("display source=%q", got)
	}
	method.Props["OFFSET"], method.Props["OFFSET_END"] = json.Number("3"), json.Number("13")
	if span, ok := p.span(method); !ok || file.content[span.Start:span.End] != "int f() {}" {
		t.Fatalf("verified byte span unavailable: %+v %v", span, ok)
	}
}

func TestCPPSourceOffsetsHandleSurrogatesAndMacroEndPositions(t *testing.T) {
	file := newSourceFile("😀 int f() { return 1; }")
	method := &vertex{Label: "METHOD", Props: map[string]any{"CODE": "int f() { return 1; }", "OFFSET": json.Number("3"), "OFFSET_END": json.Number("5")}}
	start, end, ok := file.offsets(method)
	if !ok || start != 5 || file.content[start:end] != method.text("CODE") {
		t.Fatalf("verified UTF-16 source with incomplete macro end is unavailable: %d %d %v", start, end, ok)
	}
	method.Props["OFFSET"] = json.Number("1")
	if _, _, ok := file.offsets(method); ok {
		t.Fatal("source range starts inside a surrogate pair")
	}
}
