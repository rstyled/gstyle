package bindingflow

import (
	"encoding/json"
	"fmt"
	"github.com/rstyled/gstyle/domain/pdg"
)

// CaptureEnvironment identifies lexical cells, not runtime closure instances or
// registration-time value snapshots. Source symbol IDs belong to Owner's graph.
type CaptureEnvironment struct {
	Owner string        `json:"owner"`
	Cells []CaptureCell `json:"cells"`
}
type CaptureCell struct {
	Symbol string `json:"symbol"`
	Source string `json:"source"`
}

// CaptureBindings returns body-symbol -> enclosing-symbol links. A nil map
// means absent evidence; a nonnil empty map is an explicit empty environment.
// Links require the same known source revision. Both graphs remain borrowed
// and immutable.
func CaptureBindings(body, owner *pdg.Graph) (map[pdg.Ref]pdg.Ref, error) {
	for _, graph := range []*pdg.Graph{body, owner} {
		if err := pdg.Validate(graph); err != nil {
			return nil, err
		}
	}
	environment, err := readCaptureEnvironment(body)
	if err != nil {
		return nil, err
	}
	if environment == nil {
		return nil, nil
	}
	if !captureOwnerMatches(body, owner, *environment) {
		return nil, fmt.Errorf("capture owner or source revision does not match enclosing graph")
	}
	return linkCapturedCells(body, owner, *environment)
}

func readCaptureEnvironment(graph *pdg.Graph) (*CaptureEnvironment, error) {
	var data string
	found := false
	for _, extension := range graph.Extensions {
		if graph.Tables.Text(extension.Name) != "gstyle.closure:environment" {
			continue
		}
		if found {
			return nil, fmt.Errorf("duplicate closure environment")
		}
		data = graph.Tables.Text(extension.Value)
		found = true
	}
	if !found {
		return nil, nil
	}
	return decodeCaptureEnvironment(data)
}

func decodeCaptureEnvironment(data string) (*CaptureEnvironment, error) {
	var environment CaptureEnvironment
	if err := json.Unmarshal([]byte(data), &environment); err != nil {
		return nil, err
	}
	if environment.Owner == "" || environment.Cells == nil {
		return nil, fmt.Errorf("closure environment requires owner and cell inventory")
	}
	return &environment, nil
}

func linkCapturedCells(body, owner *pdg.Graph, environment CaptureEnvironment) (map[pdg.Ref]pdg.Ref, error) {
	result := map[pdg.Ref]pdg.Ref{}
	sources := map[pdg.Ref]bool{}
	for _, cell := range environment.Cells {
		inner, outer, err := captureCellPair(body, owner, cell)
		if err != nil {
			return nil, err
		}
		_, duplicate := result[inner]
		if duplicate || sources[outer] {
			return nil, fmt.Errorf("duplicate captured cell binding")
		}
		result[inner] = outer
		sources[outer] = true
	}
	if len(result) != capturedSymbolCount(body) {
		return nil, fmt.Errorf("closure environment omits captured symbols")
	}
	return result, nil
}

func captureCellPair(body, owner *pdg.Graph, cell CaptureCell) (pdg.Ref, pdg.Ref, error) {
	inner, found := captureSymbolID(body, cell.Symbol)
	if !found {
		return 0, 0, fmt.Errorf("capture body symbol is absent")
	}
	outer, found := captureSymbolID(owner, cell.Source)
	if !found || !sameCapturedSymbol(body, inner, owner, outer) {
		return 0, 0, fmt.Errorf("capture binding does not identify the same lexical cell")
	}
	return inner, outer, nil
}

func captureSymbolID(graph *pdg.Graph, id string) (pdg.Ref, bool) {
	for index, symbol := range graph.Symbols {
		if graph.Tables.Text(symbol.ID) == id {
			return pdg.Ref(index + 1), true
		}
	}
	return 0, false
}

func sameCapturedSymbol(body *pdg.Graph, inner pdg.Ref, owner *pdg.Graph, outer pdg.Ref) bool {
	left, right := body.Symbols[inner-1], owner.Symbols[outer-1]
	return left.Role == pdg.CaptureRole && captureCellRole(right.Role) &&
		sameCaptureFact(body, left.Name, owner, right.Name) && sameCaptureFact(body, left.Type, owner, right.Type) &&
		body.Tables.Text(left.Scope) == owner.Tables.Text(right.Scope)
}

func sameCaptureFact(left *pdg.Graph, a pdg.Fact, right *pdg.Graph, b pdg.Fact) bool {
	return a.Status == pdg.Known && b.Status == pdg.Known && left.Tables.Text(a.Value) == right.Tables.Text(b.Value)
}

func captureCellRole(role pdg.Role) bool {
	switch role {
	case pdg.LocalRole, pdg.ParameterRole, pdg.ReceiverRole, pdg.OutputRole, pdg.CaptureRole:
		return true
	default:
		return false
	}
}

func capturedSymbolCount(graph *pdg.Graph) int {
	count := 0
	for _, symbol := range graph.Symbols {
		if symbol.Role == pdg.CaptureRole {
			count++
		}
	}
	return count
}

func captureOwnerMatches(body, owner *pdg.Graph, environment CaptureEnvironment) bool {
	left := body.Sources[body.Function.Span.Source-1]
	right := owner.Sources[owner.Function.Span.Source-1]
	return environment.Owner == owner.Tables.Text(owner.Function.ID) &&
		body.Tables.Text(left.Path) == owner.Tables.Text(right.Path) && sameCaptureFact(body, left.ContentIdentity, owner, right.ContentIdentity)
}
