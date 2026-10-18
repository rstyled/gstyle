package patterns

import (
	"path"
	"strconv"

	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

type commandArguments struct {
	graph  *pdg.Graph
	values *bindingflow.Query
}

// A tainted executable is a command boundary. Arguments become shell code only
// when a known shell consumes them with -c. Ordinary argv data is not shell
// syntax. Missing or conflicting canonical operands do not establish a shell.
func commandSink(f *MiningFacts, index int, sink *taintSink) *taintSink {
	if sink.desc != "command injection" {
		return sink
	}
	values, err := bindingflow.New(f.Canonical)
	if err != nil || values == nil {
		return sink
	}
	arguments := commandArguments{f.Canonical, values}
	executable := sink.dangerousArgs[0]
	if !shellExecutable(arguments.literal(index, executable)) || arguments.literal(index, executable+1) != "-c" {
		return sink
	}
	return &taintSink{sink.callee, []int{executable, executable + 2}, sink.desc}
}

func shellExecutable(name string) bool {
	switch path.Base(name) {
	case "sh", "bash", "dash", "ksh", "zsh":
		return true
	default:
		return false
	}
}

// Only direct typed AST literals establish executable and flag identity.
// Computed or unrecorded values stay unknown.
func (arguments commandArguments) literal(index, argument int) string {
	ref := arguments.values.Operand(pdg.Ref(index+1), pdg.Position(argument+1))
	if ref == 0 {
		return ""
	}
	value := arguments.graph.NodeAttributes(arguments.graph.Nodes[ref-1]).Literal
	if value.Status != pdg.Known {
		return ""
	}
	text, err := strconv.Unquote(arguments.graph.Tables.Text(value.Value))
	if err != nil {
		return ""
	}
	return text
}
