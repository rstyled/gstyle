package gopatterns

import (
	"bytes"
	"fmt"
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
	"go/token"
	"os"
)

func sourceCommentRanges(fset *token.FileSet, file *ast.File, source []byte) []pdg.ByteRange {
	var ranges []pdg.ByteRange
	for _, group := range file.Comments {
		for _, comment := range group.List {
			start := fset.PositionFor(comment.Pos(), false).Offset
			ranges = append(ranges, pdg.ByteRange{Start: uint32(start), End: uint32(rawCommentEnd(source, start))})
		}
	}
	return ranges
}

func rawCommentEnd(source []byte, start int) int {
	if source[start+1] == '/' {
		if end := bytes.IndexByte(source[start:], '\n'); end >= 0 {
			return start + end
		}
		return len(source)
	}
	return start + bytes.Index(source[start:], []byte("*/")) + 2
}

func patternSourceBytes(path string, source []byte) ([]byte, error) {
	if source == nil {
		var err error
		source, err = os.ReadFile(path)
		if err != nil {
			return nil, err
		}
	}
	if uint64(len(source)) > uint64(^uint32(0)) {
		return nil, fmt.Errorf("source exceeds PDG range width: %s", path)
	}
	return source, nil
}
