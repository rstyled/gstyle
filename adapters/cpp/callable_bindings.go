package cpp

import "github.com/rstyled/gstyle/domain/pdg"

func (d *document) indirectTarget(call *vertex) *vertex {
	if target := d.lambdaTarget(call); target != nil {
		return target
	}
	declaration := callableDeclaration(call)
	if declaration == nil || d.UnboundUses[call.Owner][declaration.text("NAME")] {
		return nil
	}
	assignment := stableInitializer(d.Referrers[declaration])
	if assignment == nil || !recordedDominates(assignment, call) {
		return nil
	}
	return initializedMethod(assignment, call)
}

func callableDeclaration(call *vertex) *vertex {
	declaration := receiverBinding(call)
	if declaration == nil {
		return nil
	}
	if declaration.Label != "LOCAL" || declaration.Owner != call.Owner || declaration.text("CLOSURE_BINDING_ID") != "" {
		return nil
	}
	return declaration
}

func receiverBinding(call *vertex) *vertex {
	if importedCallForm(call) != pdg.IndirectCall || len(call.Out["RECEIVER"]) != 1 {
		return nil
	}
	receiver := call.Out["RECEIVER"][0]
	if receiver.Label != "IDENTIFIER" || len(receiver.Out["REF"]) != 1 {
		return nil
	}
	return receiver.Out["REF"][0]
}

func stableInitializer(uses []*vertex) *vertex {
	var assignment *vertex
	for _, use := range uses {
		if receiverUse(use) {
			continue
		}
		if !initializerUse(use) || assignment != nil {
			return nil
		}
		assignment = use.Parent
	}
	return assignment
}

func receiverUse(node *vertex) bool {
	parent := node.Parent
	if parent == nil || node.Label != "IDENTIFIER" {
		return false
	}
	receivers := parent.Out["RECEIVER"]
	return importedCallForm(parent) == pdg.IndirectCall && len(receivers) == 1 && receivers[0] == node
}

func initializerUse(node *vertex) bool {
	parent := node.Parent
	if parent == nil || node.Label != "IDENTIFIER" {
		return false
	}
	return parent.text("NAME") == "<operator>.assignment" && node.number("ARGUMENT_INDEX") == 1
}

func initializedMethod(assignment, call *vertex) *vertex {
	node := argument(assignment, 2)
	if node == nil || node.Label != "METHOD_REF" {
		return nil
	}
	methods := node.Out["REF"]
	if len(methods) == 1 && methods[0].Label == "METHOD" && sameCallSignature(call, methods[0]) {
		return methods[0]
	}
	return nil
}

func argument(call *vertex, index int) *vertex {
	var result *vertex
	for _, node := range call.Children {
		if node.number("ARGUMENT_INDEX") != index {
			continue
		}
		if result != nil {
			return nil
		}
		result = node
	}
	return result
}

func recordedDominates(source, target *vertex) bool {
	queue := []*vertex{source}
	seen := map[*vertex]bool{source: true}
	for i := 0; i < len(queue); i++ {
		node := queue[i]
		if node == target {
			return true
		}
		for _, next := range node.Out["DOMINATE"] {
			if !seen[next] && next.Owner == source.Owner {
				seen[next] = true
				queue = append(queue, next)
			}
		}
	}
	return false
}
