package joern

import (
	"encoding/json"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestSourceRangeRequiresMatchingBytes(t *testing.T) {
	file := newSourceFile("é int f() {}")
	method := &vertex{Label: "METHOD", Props: map[string]any{"NAME": "f", "CODE": "int f() {}", "FILENAME": "sample.cpp", "OFFSET": json.Number("2"), "OFFSET_END": json.Number("12"), "LINE_NUMBER": json.Number("1"), "COLUMN_NUMBER": json.Number("3")}}
	p := &projection{method: method, file: file, code: file.content, pool: pdg.NewBuilder()}
	if _, ok := p.span(method); ok {
		t.Fatal("UTF-16 offset is accepted as a byte offset without matching source")
	}
	if got := methodSource(method, file); got != "int f() {}" {
		t.Fatalf("display source=%q", got)
	}
	method.Props["OFFSET"], method.Props["OFFSET_END"] = json.Number("3"), json.Number("13")
	if span, ok := p.span(method); !ok || file.content[span.Start:span.End] != "int f() {}" {
		t.Fatalf("verified byte span unavailable: %+v %v", span, ok)
	}
}
