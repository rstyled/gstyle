package joern

import (
	"strings"

	"github.com/rstyled/gstyle/domain"
)

type scoreFrame struct {
	node   *vertex
	depth  int
	elseIf bool
}

type scorer struct {
	method   *vertex
	function domain.Function
}

var structural = map[string]bool{"IF": true, "FOR": true, "WHILE": true, "DO": true, "SWITCH": true, "CATCH": true}
var cyclomaticControl = map[string]bool{"IF": true, "FOR": true, "WHILE": true, "DO": true, "CATCH": true}

func scoreMethod(method *vertex, recursive bool, source *sourceFile) domain.Function {
	s := scorer{method: method, function: domain.Function{Package: "joern", Name: method.text("FULL_NAME"), Line: max(1, method.number("LINE_NUMBER")), EndLine: max(1, method.number("LINE_NUMBER_END")), Column: max(1, method.number("COLUMN_NUMBER")), Source: methodSource(method, source)}}
	s.cyclomatic(method, "function")
	queue := []scoreFrame{{node: method}}
	for len(queue) != 0 {
		frame := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		if frame.node.Owner != method {
			continue
		}
		queue = append(queue, s.visit(frame)...)
	}
	if recursive {
		s.cognitive(method, "recursion", 0)
	}
	s.function.EndLine = max(s.function.Line, s.function.EndLine)
	return s.function
}

func (s *scorer) visit(frame scoreFrame) []scoreFrame {
	node := frame.node
	if node.Label == "CONTROL_STRUCTURE" {
		return s.control(frame)
	}
	if node.Label == "JUMP_TARGET" && strings.HasPrefix(strings.TrimSpace(node.text("CODE")), "case ") {
		s.cyclomatic(node, "case")
	}
	if node.Label == "CALL" {
		return s.expression(frame)
	}
	return childFrames(node, frame.depth)
}

func childFrames(node *vertex, depth int) []scoreFrame {
	frames := make([]scoreFrame, 0, len(node.Children))
	for i := len(node.Children) - 1; i >= 0; i-- {
		frames = append(frames, scoreFrame{node: node.Children[i], depth: depth})
	}
	return frames
}

func (s *scorer) control(frame scoreFrame) []scoreFrame {
	node, depth := frame.node, frame.depth
	kind := node.text("CONTROL_STRUCTURE_TYPE")
	if cyclomaticControl[kind] {
		s.cyclomatic(node, kind)
	}
	if structural[kind] {
		cost := depth
		if frame.elseIf {
			cost = 0
		}
		s.cognitive(node, kind, cost)
		depth++
	}
	if kind == "GOTO" {
		s.cognitive(node, kind, 0)
	}
	frames := childFrames(node, depth)
	predicateDepth(node, frame.depth, frames)
	if kind == "IF" {
		s.ifAlternatives(node, frame.depth, frames)
	}
	return frames
}

func predicateDepth(node *vertex, depth int, frames []scoreFrame) {
	if len(node.Out["CONDITION"]) != 1 {
		return
	}
	for i := range frames {
		if frames[i].node == node.Out["CONDITION"][0] {
			frames[i].depth = depth
		}
	}
}

func (s *scorer) ifAlternatives(node *vertex, depth int, frames []scoreFrame) {
	if len(node.Children) < 3 {
		return
	}
	last := node.Children[len(node.Children)-1]
	if branch := elseIfBranch(last); branch != nil {
		frames[0] = scoreFrame{node: branch, depth: depth, elseIf: true}
		return
	}
	s.cognitive(node.Children[len(node.Children)-1], "else", 0)
}

func elseIfBranch(node *vertex) *vertex {
	if isIf(node) {
		return node
	}
	if node.Label != "BLOCK" || len(node.Children) != 1 {
		return nil
	}
	if strings.HasPrefix(strings.TrimSpace(node.text("CODE")), "{") {
		return nil
	}
	if isIf(node.Children[0]) {
		return node.Children[0]
	}
	return nil
}

func isIf(node *vertex) bool {
	return node.Label == "CONTROL_STRUCTURE" && node.text("CONTROL_STRUCTURE_TYPE") == "IF"
}

func logical(node *vertex) string {
	name := node.text("NAME")
	if node.Label == "CALL" && (name == "<operator>.logicalAnd" || name == "<operator>.logicalOr") {
		return name
	}
	return ""
}

func (s *scorer) expression(frame scoreFrame) []scoreFrame {
	node := frame.node
	if logical(node) != "" {
		s.cyclomatic(node, node.text("NAME"))
		if node.Parent == nil || logical(node.Parent) == "" {
			s.logicalSequence(node)
		}
	}
	if node.text("NAME") == "<operator>.conditional" {
		s.cyclomatic(node, "?:")
		s.cognitive(node, "?:", frame.depth)
		frame.depth++
	}
	return childFrames(node, frame.depth)
}

func (s *scorer) logicalSequence(root *vertex) {
	type item struct {
		node *vertex
		emit bool
	}
	stack := []item{{node: root}}
	previous := ""
	for len(stack) != 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		node := current.node
		kind := logical(node)
		if kind == "" {
			continue
		}
		if !current.emit {
			stack = append(stack, item{node: node.Children[len(node.Children)-1]}, item{node: node, emit: true}, item{node: node.Children[0]})
			continue
		}
		if kind != previous {
			s.cognitive(node, kind, 0)
			previous = kind
		}
	}
}

func (s *scorer) cyclomatic(node *vertex, kind string) {
	s.function.Complexity++
	s.function.CyclomaticDiagnostics = append(s.function.CyclomaticDiagnostics, domain.CyclomaticDiagnostic{Kind: kind, Line: max(1, node.number("LINE_NUMBER")), Column: max(1, node.number("COLUMN_NUMBER"))})
}

func (s *scorer) cognitive(node *vertex, kind string, nesting int) {
	s.function.CognitiveComplexity += nesting + 1
	s.function.CognitiveDiagnostics = append(s.function.CognitiveDiagnostics, domain.CognitiveDiagnostic{Kind: kind, Increment: nesting + 1, Nesting: nesting, Line: max(1, node.number("LINE_NUMBER")), Column: max(1, node.number("COLUMN_NUMBER"))})
}
