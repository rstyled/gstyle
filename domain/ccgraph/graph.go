package ccgraph

import (
	"context"
	"fmt"
	"slices"
)

type Label string

const (
	Decl    Label = "DECL"
	Assign  Label = "ASSIGN"
	Control Label = "CTRL"
	State   Label = "STATE"
	Other   Label = "OTHER"
)

type Family string

const (
	Data      Family = "data"
	Ctrl      Family = "control"
	Execution Family = "execution"
)

// IDs support navigation only. Every edge occurrence is retained.
type Node struct {
	ID    string
	Label Label
}
type Edge struct {
	From, To string
	Family   Family
}
type Graph struct {
	Nodes []Node
	Edges []Edge
}

func (g Graph) validate() (map[string]int, error) {
	index := make(map[string]int, len(g.Nodes))
	for i, n := range g.Nodes {
		if err := validateNode(n, index); err != nil {
			return nil, err
		}
		index[n.ID] = i
	}
	for _, e := range g.Edges {
		if err := validateEdge(e, index); err != nil {
			return nil, err
		}
	}
	return index, nil
}

func validateNode(n Node, index map[string]int) error {
	if n.ID == "" {
		return fmt.Errorf("empty node ID")
	}
	if _, ok := index[n.ID]; ok {
		return fmt.Errorf("duplicate node %q", n.ID)
	}
	if !slices.Contains([]Label{Decl, Assign, Control, State, Other}, n.Label) {
		return fmt.Errorf("unknown label %q", n.Label)
	}
	return nil
}

func validateEdge(e Edge, index map[string]int) error {
	if _, ok := index[e.From]; !ok {
		return fmt.Errorf("unknown source %q", e.From)
	}
	if _, ok := index[e.To]; !ok {
		return fmt.Errorf("unknown target %q", e.To)
	}
	if !slices.Contains([]Family{Data, Ctrl, Execution}, e.Family) {
		return fmt.Errorf("unknown family %q", e.Family)
	}
	return nil
}

func successors(g Graph, index map[string]int) [][]int {
	out := make([][]int, len(g.Nodes))
	for _, e := range g.Edges {
		out[index[e.From]] = append(out[index[e.From]], index[e.To])
	}
	return out
}

// Diameter follows recorded direction and ignores unreachable pairs.
func Diameter(ctx context.Context, g Graph) (int, error) {
	index, err := g.validate()
	if err != nil {
		return 0, err
	}
	out := successors(g, index)
	diameter := 0
	for start := range g.Nodes {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		d, err := eccentricity(ctx, out, start)
		if err != nil {
			return 0, err
		}
		diameter = max(diameter, d)
	}
	return diameter, nil
}

func eccentricity(ctx context.Context, out [][]int, start int) (int, error) {
	dist := make([]int, len(out))
	for i := range dist {
		dist[i] = -1
	}
	dist[start] = 0
	queue := []int{start}
	farthest := 0
	for i := 0; i < len(queue); i++ {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		for _, next := range out[queue[i]] {
			if dist[next] >= 0 {
				continue
			}
			dist[next] = dist[queue[i]] + 1
			farthest = max(farthest, dist[next])
			queue = append(queue, next)
		}
	}
	return farthest, nil
}

func Depth(sizeA, sizeB, diameterA, diameterB int) int {
	if sizeA < sizeB {
		return diameterA
	}
	if sizeB < sizeA {
		return diameterB
	}
	return min(diameterA, diameterB)
}
