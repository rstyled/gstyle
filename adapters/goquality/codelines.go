package goquality

import (
	"fmt"
	"go/scanner"
	"go/token"
	"strings"

	"github.com/rstyled/gstyle/domain/quality/linecount"
)

// codeLines counts physical lines containing tokens, including every line of
// a multiline string. Comments, blank lines and inserted semicolons do not count.
func codeLines(source string) (int, error) {
	file := token.NewFileSet().AddFile("function.go", -1, len(source))
	var lexer scanner.Scanner
	lexer.Init(file, []byte(source), nil, 0)
	var count linecount.Counter
	for {
		pos, kind, literal := lexer.Scan()
		if kind == token.EOF {
			return count.Lines, nil
		}
		if kind == token.SEMICOLON && literal == "\n" {
			continue
		}
		first := file.PositionFor(pos, false).Line
		last := literalLastLine(first, kind, literal)
		next, ok := count.Add(first, last)
		if !ok {
			return 0, fmt.Errorf("invalid code-line span %d..%d after %d", first, last, count.Through)
		}
		count = next
	}
}

func literalLastLine(first int, kind token.Token, literal string) int {
	if kind != token.STRING {
		return first
	}
	return first + strings.Count(literal, "\n")
}
