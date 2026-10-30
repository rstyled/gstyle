package cpp

import (
	"strings"
	"unicode/utf8"
)

func utf16Boundaries(content string) []int {
	result := make([]int, 1, len(content)+1)
	for offset, ch := range content {
		if ch > 0xffff {
			result = append(result, -1)
		}
		result = append(result, offset+utf8.RuneLen(ch))
	}
	return result
}

func (f *sourceFile) offsets(node *vertex) (int, int, bool) {
	start := node.number("OFFSET")
	code := node.text("CODE")
	if start < 0 || code == "" {
		return 0, 0, false
	}
	if from, to, ok := f.unitCode(start, code); ok {
		return from, to, true
	}
	return f.byteCode(start, code)
}

func (f *sourceFile) unitCode(start int, code string) (int, int, bool) {
	if start >= len(f.units) {
		return 0, 0, false
	}
	return f.byteCode(f.units[start], code)
}

func (f *sourceFile) byteCode(start int, code string) (int, int, bool) {
	if start < 0 || start > len(f.content) {
		return 0, 0, false
	}
	if !strings.HasPrefix(f.content[start:], code) {
		return 0, 0, false
	}
	return start, start + len(code), true
}
