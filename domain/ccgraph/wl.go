package ccgraph

import (
	"context"
	"math/big"
	"slices"
	"strconv"
)

type Histogram map[int]uint64

type Round struct {
	Weight                       int
	ColorsA, ColorsB             []int
	HistogramA, HistogramB       Histogram
	Overlap, Cross, SelfA, SelfB *big.Int
}

type Kernel struct {
	Depth, DiameterA, DiameterB int
	Rounds                      []Round
	Overlap, LargerCoverage     *big.Rat
	Cosine                      Cosine
}

type neighbor struct {
	family int
	node   int
}

type indexedGraph struct {
	incoming, outgoing [][]neighbor
	colors             []int
}

var initialColors = map[Label]int{Decl: 1, Assign: 2, Control: 3, State: 4, Other: 5}

var familyOrder = map[Family]int{Data: 0, Ctrl: 1, Execution: 2}

type kernelPair struct {
	a, b                 Graph
	diameterA, diameterB int
}

func (p kernelPair) compare(ctx context.Context) (Kernel, error) {
	a, b := p.a, p.b
	da, db := p.diameterA, p.diameterB
	h := Depth(len(a.Nodes), len(b.Nodes), da, db)
	ga, gb := indexGraph(a), indexGraph(b)
	ca, cb := ga.colors, gb.colors
	out := Kernel{Depth: h, DiameterA: da, DiameterB: db}
	out.Rounds = append(out.Rounds, makeRound(ca, cb, h+1))
	for level := range h {
		if err := ctx.Err(); err != nil {
			return Kernel{}, err
		}
		intern := map[string]int{}
		ca, cb = refine(ga, ca, intern), refine(gb, cb, intern)
		out.Rounds = append(out.Rounds, makeRound(ca, cb, h-level))
	}
	setScores(&out, len(a.Nodes), len(b.Nodes))
	return out, nil
}

func indexGraph(g Graph) indexedGraph {
	index, _ := g.validate() // boundary validation precedes this helper
	out := indexedGraph{make([][]neighbor, len(g.Nodes)), make([][]neighbor, len(g.Nodes)), make([]int, len(g.Nodes))}
	for i, n := range g.Nodes {
		out.colors[i] = initialColors[n.Label]
	}
	for _, e := range g.Edges {
		from, to := index[e.From], index[e.To]
		out.incoming[to] = append(out.incoming[to], neighbor{familyOrder[e.Family], from})
		out.outgoing[from] = append(out.outgoing[from], neighbor{familyOrder[e.Family], to})
	}
	return out
}

func neighborKey(neighbors []neighbor, colors []int) string {
	parts := make([][2]int, len(neighbors))
	for i, n := range neighbors {
		parts[i] = [2]int{n.family, colors[n.node]}
	}
	slices.SortFunc(parts, func(a, b [2]int) int {
		if a[0] != b[0] {
			return a[0] - b[0]
		}
		return a[1] - b[1]
	})
	var key []byte
	for _, part := range parts {
		key = strconv.AppendInt(key, int64(part[0]), 10)
		key = append(key, ':')
		key = strconv.AppendInt(key, int64(part[1]), 10)
		key = append(key, ';')
	}
	return string(key)
}

func refine(g indexedGraph, colors []int, intern map[string]int) []int {
	out := make([]int, len(colors))
	for i, own := range colors {
		key := strconv.Itoa(own) + "|" + neighborKey(g.incoming[i], colors) + "|" + neighborKey(g.outgoing[i], colors)
		label, ok := intern[key]
		if !ok {
			label = len(intern) + 1
			intern[key] = label
		}
		out[i] = label
	}
	return out
}

func histogram(colors []int) Histogram {
	out := Histogram{}
	for _, color := range colors {
		out[color]++
	}
	return out
}

func histogramCounts(a, b Histogram) (*big.Int, *big.Int) {
	overlap, cross := new(big.Int), new(big.Int)
	for label, count := range a {
		overlap.Add(overlap, new(big.Int).SetUint64(min(count, b[label])))
		term := new(big.Int).Mul(new(big.Int).SetUint64(count), new(big.Int).SetUint64(b[label]))
		cross.Add(cross, term)
	}
	return overlap, cross
}

func makeRound(a, b []int, weight int) Round {
	ha, hb := histogram(a), histogram(b)
	overlap, cross := histogramCounts(ha, hb)
	_, aa := histogramCounts(ha, ha)
	_, bb := histogramCounts(hb, hb)
	return Round{weight, a, b, ha, hb, overlap, cross, aa, bb}
}

// CompareWL retains exact equality partitions, not hash-only labels. There is
// no depth cap or graph-size fallback. Cancellation returns a diagnostic error.
func CompareWL(ctx context.Context, a, b Graph) (Kernel, error) {
	da, err := Diameter(ctx, a)
	if err != nil {
		return Kernel{}, err
	}
	db, err := Diameter(ctx, b)
	if err != nil {
		return Kernel{}, err
	}
	return (kernelPair{a, b, da, db}).compare(ctx)
}

func setScores(out *Kernel, sizeA, sizeB int) {
	overlap, cross, aa, bb, weights := new(big.Int), new(big.Int), new(big.Int), new(big.Int), new(big.Int)
	for _, r := range out.Rounds {
		w := big.NewInt(int64(r.Weight))
		weights.Add(weights, w)
		weightedAdd(overlap, w, r.Overlap)
		weightedAdd(cross, w, r.Cross)
		weightedAdd(aa, w, r.SelfA)
		weightedAdd(bb, w, r.SelfB)
	}
	out.Overlap = normalized(overlap, weights, min(sizeA, sizeB))
	out.LargerCoverage = normalized(overlap, weights, max(sizeA, sizeB))
	out.Cosine = Cosine{cross, new(big.Int).Mul(aa, bb)}
}

func weightedAdd(sum, weight, count *big.Int) { sum.Add(sum, new(big.Int).Mul(weight, count)) }

func normalized(count, weights *big.Int, size int) *big.Rat {
	den := new(big.Int).Mul(weights, big.NewInt(int64(size)))
	if den.Sign() == 0 {
		return new(big.Rat)
	}
	return new(big.Rat).SetFrac(count, den)
}
