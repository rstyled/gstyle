package pdg

// BodyFlow is the canonical sequential operation flow before deferred unwind.
// A registration-flow context excludes deferred invocations and is not a full
// normal-completion or ordinary reachability claim for other consumers.
func BodyFlow(g *Graph) *NormalFlow {
	if g.NormalFlow != nil {
		return g.NormalFlow
	}
	return bodyValueFlow(g)
}
func bodyValueFlow(g *Graph) *NormalFlow {
	if flow := sharedProjectionFlow(g); flow != nil {
		return flow
	}
	for _, extension := range g.Extensions {
		if g.Tables.Text(extension.Name) == "gstyle.pdg:registration-flow" && g.Tables.Text(extension.Value) == `"0.1.0"` {
			return g.ValueFlow
		}
	}
	return nil
}

// ExitBodyFlow includes sequential recovery operations. Its consumers must
// interpret recovery and unwind separately from ordinary control dependencies.
func ExitBodyFlow(g *Graph) *NormalFlow {
	if flow := BodyFlow(g); flow != nil {
		return flow
	}
	for _, extension := range g.Extensions {
		if g.Tables.Text(extension.Name) == "gstyle.pdg:exit-body-flow" && g.Tables.Text(extension.Value) == `"0.1.0"` {
			return g.ValueFlow
		}
	}
	return nil
}
