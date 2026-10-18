package gopatterns

import (
	"cmp"
	"encoding/json"
	"fmt"
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"slices"
)

func (p *packageAnalysis) captureEnvironment(parent functionGraph, body *functionGraph) error {
	environment := bindingflow.CaptureEnvironment{Owner: parent.graph.Tables.Text(parent.graph.Function.ID), Cells: []bindingflow.CaptureCell{}}
	for object, inner := range body.inventory.symbols {
		if body.inventory.graph.Symbols[inner-1].Role != pdg.CaptureRole {
			continue
		}
		outer := parent.inventory.symbols[object]
		if outer == 0 {
			return fmt.Errorf("closure %s capture %s lacks an enclosing cell", body.graph.Tables.Text(body.graph.Function.ID), object.Name())
		}
		environment.Cells = append(environment.Cells, bindingflow.CaptureCell{
			Symbol: body.graph.Tables.Text(body.graph.Symbols[inner-1].ID), Source: parent.graph.Tables.Text(parent.graph.Symbols[outer-1].ID),
		})
	}
	slices.SortFunc(environment.Cells, func(a, b bindingflow.CaptureCell) int { return cmp.Compare(a.Symbol, b.Symbol) })
	// The record contains only strings and slices; Marshal cannot fail.
	data, _ := json.Marshal(environment)
	extension := p.pool.SingletonExtension("gstyle.closure:environment", string(data))
	body.inventory.graph.Extensions = append(body.inventory.graph.Extensions, extension...)
	body.graph.Extensions = append(body.graph.Extensions, extension...)
	return nil
}
