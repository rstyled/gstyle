package joern

// Tarjan components score each member of a resolved recursion cycle once.
// Same-spelled calls without a CALL edge do not establish recursion.
type components struct {
	next       int
	index, low map[*vertex]int
	active     map[*vertex]bool
	stack      []*vertex
	recursive  map[*vertex]bool
	calls      map[*vertex][]*vertex
}

func recursiveMethods(doc *document) map[*vertex]bool {
	c := components{index: map[*vertex]int{}, low: map[*vertex]int{}, active: map[*vertex]bool{}, recursive: map[*vertex]bool{}, calls: map[*vertex][]*vertex{}}
	for _, link := range doc.Edges {
		if resolvedCall(link) {
			c.calls[link.Source.Owner] = append(c.calls[link.Source.Owner], link.Target)
		}
	}
	for _, method := range doc.Methods {
		if c.index[method] == 0 {
			c.visit(method)
		}
	}
	return c.recursive
}

func resolvedCall(link edge) bool {
	return link.Label == "CALL" && link.Source.Owner != nil && link.Target.Label == "METHOD"
}

func (c *components) visit(method *vertex) {
	c.next++
	c.index[method], c.low[method], c.active[method] = c.next, c.next, true
	c.stack = append(c.stack, method)
	for _, target := range c.calls[method] {
		c.link(method, target)
	}
	if c.low[method] == c.index[method] {
		c.finish(method)
	}
}

func (c *components) link(method, target *vertex) {
	if c.index[target] == 0 {
		c.visit(target)
		c.low[method] = min(c.low[method], c.low[target])
		return
	}
	if c.active[target] {
		c.low[method] = min(c.low[method], c.index[target])
	}
	if target == method {
		c.recursive[method] = true
	}
}

func (c *components) finish(method *vertex) {
	var members []*vertex
	for len(c.stack) > 0 {
		last := c.stack[len(c.stack)-1]
		c.stack = c.stack[:len(c.stack)-1]
		c.active[last] = false
		members = append(members, last)
		if last == method {
			break
		}
	}
	if len(members) < 2 {
		return
	}
	for _, member := range members {
		c.recursive[member] = true
	}
}
