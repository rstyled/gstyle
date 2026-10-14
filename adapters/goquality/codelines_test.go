package goquality

import (
	"go/scanner"
	"go/token"
	"strings"
	"testing"
)

func TestCodeLinesBoundaries(t *testing.T) {
	for _, test := range []struct {
		name, source string
		want         int
	}{
		{"empty", "", 0}, {"comments", "// comment\n/* block\ncomment */\n", 0},
		{"single line", "func f() {}", 1}, {"blank lines", "func f() {\n\n}\n", 2},
		{"multiline signature", "func f(\n x int,\n) {\n}\n", 4},
		{"raw string", "func f() {\n_ = `a\n\nb`\n}\n", 5},
		{"CRLF", "func f() {\r\n_ = `a\r\nb`\r\n}\r\n", 4},
		{"line directive", "func f() {\n//line other.go:1\nx := 1\n_ = x\n}\n", 4},
		{"escaped newline", "func f() {\n_ = \"a\\nb\"\n}\n", 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := codeLines(test.source)
			if err != nil || got != test.want {
				t.Fatalf("got (%d, %v), want %d", got, err, test.want)
			}
		})
	}
}

// The map oracle records individual physical lines. It does not use the
// streaming counter or its interval arithmetic.
func referenceCodeLines(source string) int {
	file := token.NewFileSet().AddFile("reference.go", -1, len(source))
	var lexer scanner.Scanner
	lexer.Init(file, []byte(source), nil, 0)
	lines := map[int]bool{}
	for {
		pos, kind, literal := lexer.Scan()
		if kind == token.EOF {
			return len(lines)
		}
		if kind == token.SEMICOLON && literal == "\n" {
			continue
		}
		first := file.PositionFor(pos, false).Line
		lines[first] = true
		if kind == token.STRING {
			for offset := 1; offset <= strings.Count(literal, "\n"); offset++ {
				lines[first+offset] = true
			}
		}
	}
}

func FuzzCodeLines(f *testing.F) {
	for _, source := range []string{"", "func f() {}", "func f() {\n//line other.go:1\n_ = `a\n\nb`\n}\n", "\r\n/* comment */\r\n", "`unterminated\nraw"} {
		f.Add(source)
	}
	f.Fuzz(func(t *testing.T, source string) {
		got, err := codeLines(source)
		if err != nil {
			t.Fatalf("source scanner produces invalid span: %v", err)
		}
		if want := referenceCodeLines(source); got != want {
			t.Fatalf("got %d, want %d", got, want)
		}
	})
}
