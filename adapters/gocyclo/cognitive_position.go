package gocyclo

import (
	"go/ast"
	"go/token"

	cognit "github.com/uudashr/gocognit"
)

type logicalSequence struct {
	previous  token.Token
	positions []cognit.DiagnosticPosition
	fileSet   *token.FileSet
}

func cognitiveLogicalPositions(diagnostics []cognit.Diagnostic, node ast.Node, fileSet *token.FileSet) map[int][]cognit.DiagnosticPosition {
	positions := logicalDiagnosticOffsets(diagnostics)
	if len(positions) == 0 {
		return positions
	}
	for node := range ast.Preorder(node) {
		binary, ok := node.(*ast.BinaryExpr)
		if !ok {
			continue
		}
		offset := fileSet.Position(binary.OpPos).Offset
		if _, needed := positions[offset]; !needed {
			continue
		}
		sequence := logicalSequence{fileSet: fileSet}
		sequence.collect(binary)
		positions[offset] = sequence.positions
	}
	return positions
}

func logicalDiagnosticOffsets(diagnostics []cognit.Diagnostic) map[int][]cognit.DiagnosticPosition {
	positions := map[int][]cognit.DiagnosticPosition{}
	previous := -1
	for _, diagnostic := range diagnostics {
		if diagnostic.Text != "&&" && diagnostic.Text != "||" {
			continue
		}
		if diagnostic.Pos.Offset == previous {
			positions[diagnostic.Pos.Offset] = nil
		}
		previous = diagnostic.Pos.Offset
	}
	return positions
}

func (s *logicalSequence) collect(expression ast.Expr) {
	binary, ok := expression.(*ast.BinaryExpr)
	if !ok {
		return
	}
	s.collect(binary.X)
	if binary.Op == token.LAND || binary.Op == token.LOR {
		if binary.Op != s.previous {
			position := s.fileSet.Position(binary.OpPos)
			s.positions = append(s.positions, cognit.DiagnosticPosition{Offset: position.Offset, Line: position.Line, Column: position.Column})
		}
		s.previous = binary.Op
	}
	s.collect(binary.Y)
}
