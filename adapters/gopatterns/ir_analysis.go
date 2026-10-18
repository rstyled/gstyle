package gopatterns

import "go/ast"

// Dependency passes start from the same source flow. Lowered operation flow is
// shared where supported; the frozen scalar matching profile retains its own
// admission policy and cannot silently certify the broader extraction.
func (b *irBuilder) dependencies(body *ast.BlockStmt) error {
	source, dom := b.execution(body)
	if err := b.runtimeFlow(body, source); err != nil {
		return err
	}
	flow, dom, err := b.reachingDefinitions(body, source, dom)
	if err != nil {
		return err
	}
	b.normalDependenciesFor(flow, dom)
	return b.semanticControlFor(body, source)
}
