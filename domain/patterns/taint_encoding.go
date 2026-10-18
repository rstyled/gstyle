package patterns

import (
	"encoding/json"
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/taintflow"
	"strings"
)

// Decimal formatting is safe for SQL syntax, not for every sink. In particular,
// decimal IP addresses can still be SSRF targets. Unknown calls can decode text
// and therefore discard this encoding evidence.
func preservesTaintEncoding(graph *pdg.Graph, ref pdg.Ref) bool {
	switch typedTaintCallee(graph, ref) {
	case "fmt.Sprintf", "strings.Join", "strings.Clone", "append":
		return true
	default:
		return false
	}
}
func typedTaintCallee(graph *pdg.Graph, ref pdg.Ref) string {
	attrs := graph.NodeAttributes(graph.Nodes[ref-1])
	if attrs.CallForm != pdg.DirectCall || attrs.Callee.Status != pdg.Known {
		return ""
	}
	return graph.Tables.Text(attrs.Callee.Value)
}
func decimalEncodings(graph *pdg.Graph, paths *bindingflow.Query, ref pdg.Ref) []taintflow.ArgumentEncoding {
	if typedTaintCallee(graph, ref) != "fmt.Sprintf" {
		return nil
	}
	arguments := commandArguments{graph: graph, values: paths}
	positions := decimalFormatPositions(arguments.literal(int(ref)-1, 0))
	var result []taintflow.ArgumentEncoding
	for _, position := range positions {
		operand := paths.Operand(ref, position)
		if basicInteger(graph, operand) {
			result = append(result, taintflow.ArgumentEncoding{Position: position, SafeFor: "SQL injection"})
		}
	}
	return result
}

// Deliberately accept only simple sequential directives. Indexed arguments,
// dynamic widths, flags, and malformed formats retain all dependencies.
func decimalFormatPositions(format string) []pdg.Position {
	var result []pdg.Position
	position := pdg.Position(2)
	for index := strings.IndexByte(format, '%'); index >= 0; index = strings.IndexByte(format, '%') {
		format = format[index+1:]
		if format == "" {
			return nil
		}
		verb := format[0]
		format = format[1:]
		if verb == '%' {
			continue
		}
		switch verb {
		case 'd':
			result = append(result, position)
		case 's', 'v', 'c', 'q', 'x', 'X', 'b', 'o', 'O', 'U', 't', 'f', 'e', 'E', 'g', 'G', 'p', 'T':
		default:
			return nil
		}
		position++
	}

	return result
}
func basicInteger(graph *pdg.Graph, ref pdg.Ref) bool {
	if ref == 0 {
		return false
	}
	return integerBasicName(exactBasicName(graph, ref))
}
func preservesValueEncoding(graph *pdg.Graph, ref pdg.Ref) bool {
	return exactBasicName(graph, ref) == "string"
}
func exactBasicName(graph *pdg.Graph, ref pdg.Ref) string {
	count, name := 0, ""
	for _, ext := range graph.NodeAttributes(graph.Nodes[ref-1]).Extensions {
		if graph.Tables.Text(ext.Name) != "gstyle.value:basic" {
			continue
		}
		count++
		if json.Unmarshal([]byte(graph.Tables.Text(ext.Value)), &name) != nil {
			return ""
		}
	}
	if count != 1 {
		return ""
	}
	return name
}

func integerBasicName(name string) bool {
	switch name {
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "untyped int", "untyped rune":
		return true
	default:
		return false
	}
}

func indexTaintEncodings(graphs map[string]*pdg.Graph) (map[taintflow.Site][]taintflow.ArgumentEncoding, error) {
	result := map[taintflow.Site][]taintflow.ArgumentEncoding{}
	for name, graph := range graphs {
		paths, err := bindingflow.New(graph)
		if err != nil {
			return nil, err
		}
		for index, node := range graph.Nodes {
			if node.Category != pdg.Call {
				continue
			}
			ref := pdg.Ref(index + 1)
			result[taintflow.Site{Function: name, Node: ref}] = decimalEncodings(graph, paths, ref)
		}
	}
	return result, nil
}
