package cpp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rstyled/gstyle/domain/pdg"
)

type operandWitness struct {
	Source, Target pdg.Ref
	Position       pdg.Position
	Subkind        string
}

func (p *projection) expectedOperands(node *vertex, expected map[string]operandWitness) {
	if node.Label != "CALL" && node.Label != "RETURN" {
		return
	}
	for _, operand := range node.Children {
		if p.refs[operand] == 0 || !valueOperand(node, operand) {
			continue
		}
		expected[operandID(operand, node)] = operandWitness{Source: p.refs[operand], Target: p.refs[node], Position: importedPosition(operand.number("ARGUMENT_INDEX")), Subkind: operandRelation(node, operand)}
	}
}

func (p *projection) verifyOperands(nodes []*vertex) error {
	expected := map[string]operandWitness{}
	for _, node := range nodes {
		p.expectedOperands(node, expected)
	}
	for _, link := range p.graph.Edges {
		id := p.pool.Tables.Text(link.ID)
		if strings.HasPrefix(id, "cpp:e") {
			continue
		}
		if err := p.verifyOperand(link, expected); err != nil {
			return err
		}
	}
	if len(expected) != 0 {
		return fmt.Errorf("CPG operand dependencies are missing")
	}
	return nil
}

func (p *projection) verifyOperand(link pdg.Edge, expected map[string]operandWitness) error {
	id := p.pool.Tables.Text(link.ID)
	want, exists := expected[id]
	actual := operandWitness{Source: link.Source, Target: link.Target, Position: link.Position, Subkind: p.pool.Tables.Text(link.Subkind)}
	if !exists || actual != want || link.Kind != pdg.Data {
		return fmt.Errorf("CPG operand dependency %s changes direction, role, or position", id)
	}
	delete(expected, id)
	return nil
}

func operandID(source, target *vertex) string {
	return "operand:" + strconv.FormatUint(source.ID, 10) + ":" + strconv.FormatUint(target.ID, 10)
}

func importedPosition(index int) pdg.Position {
	if index < 0 {
		return 0
	}
	return pdg.Position(index + 1)
}
