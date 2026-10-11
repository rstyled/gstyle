package ccgraph

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"reflect"
	"slices"
	"testing"
)

type fixtureGraph struct {
	Nodes map[string]Label
	Edges [][3]string
}

func (g fixtureGraph) graph() Graph {
	var keys []string
	for key := range g.Nodes {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	out := Graph{}
	for _, key := range keys {
		out.Nodes = append(out.Nodes, Node{key, g.Nodes[key]})
	}
	for _, e := range g.Edges {
		out.Edges = append(out.Edges, Edge{e[0], e[1], Family(e[2])})
	}
	return out
}

type kernelExpectation struct {
	H       int
	Weights []int
	Overlap []uint64 `json:"overlap_per_round"`
	Cross   []uint64 `json:"cross_per_round"`
	SelfA   []uint64 `json:"self_a_per_round"`
	SelfB   []uint64 `json:"self_b_per_round"`
	Score   [2]int64 `json:"overlap_score"`
	Larger  [2]int64 `json:"larger_coverage"`
	Cosine  struct {
		Numerator   int64
		Denominator int64 `json:"sqrt_denominator"`
	} `json:"cosine_score"`
}
type fixtures struct {
	Projection []struct {
		ID    string
		Input struct {
			Nodes []struct {
				ID, Category   string
				Administrative bool
				Reachable      *bool
				Reads          []string `json:"read_symbols"`
			}
			Edges [][3]string
		}
		Expected struct {
			Retained []string `json:"retained_nodes"`
			Removed  []string `json:"removed_nodes"`
			Edges    [][3]string
			Features [9]uint64
		}
	} `json:"projection_cases"`
	Interfaces []struct {
		ID, Name        string
		Inputs, Outputs [][3]json.RawMessage
		Expected        Strings
	} `json:"interface_cases"`
	Jaro []struct {
		ID, A, B string
		Expected [2]int64
	} `json:"jaro_winkler_cases"`
	Diameters []struct {
		ID       string
		Graph    fixtureGraph
		Expected int
	} `json:"diameter_cases"`
	Depths []struct {
		ID               string
		Sizes, Diameters [2]int
		Expected         int
	} `json:"depth_cases"`
	WL []struct {
		ID       string
		A, B     fixtureGraph
		Expected kernelExpectation
	} `json:"wl_cases"`
	Histograms []struct {
		ID       string
		Sizes    [2]int
		H        int
		A, B     []map[string]uint64
		Expected kernelExpectation
	} `json:"histogram_score_cases"`
	Boundaries []struct {
		ID, Kind  string
		Sizes     [2]int
		Score     [2]int64
		Fields    [][2]int64
		Threshold Decimal
		Expected  bool
		RawDot    int64 `json:"dot"`
		RawNorm   int64 `json:"norm_product"`
	} `json:"boundary_cases"`
}

func loadFixtures(t *testing.T) fixtures {
	t.Helper()
	data, err := os.ReadFile("testdata/compat-matching.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != "4da4eebc4bbc1aabee109e78702e4348b756b792593a4c63029074e001979ef7" {
		t.Fatalf("pinned fixture changed: %s", got)
	}
	var out fixtures
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestCompatProjectionFixtures(t *testing.T) {
	for _, f := range loadFixtures(t).Projection {
		t.Run(f.ID, func(t *testing.T) {
			input := ProjectionInput{}
			for _, n := range f.Input.Nodes {
				input.Nodes = append(input.Nodes, Operation{n.ID, n.Category, n.Administrative, n.Reachable, n.Reads})
			}
			for _, e := range f.Input.Edges {
				input.Edges = append(input.Edges, Edge{e[0], e[1], Family(e[2])})
			}
			got, err := Project(input)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got.Retained, f.Expected.Retained) || !slices.Equal(got.Removed, f.Expected.Removed) {
				t.Fatalf("projection: %+v", got)
			}
			var edges [][3]string
			for _, e := range got.Graph.Edges {
				edges = append(edges, [3]string{e.From, e.To, string(e.Family)})
			}
			if !reflect.DeepEqual(edges, f.Expected.Edges) || got.Features != f.Expected.Features {
				t.Fatalf("edges/features: %+v", got)
			}
		})
	}
}

func fixtureParameters(t *testing.T, tuples [][3]json.RawMessage) []Parameter {
	t.Helper()
	out := make([]Parameter, len(tuples))
	for i, tuple := range tuples {
		if err := json.Unmarshal(tuple[0], &out[i].Role); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(tuple[1], &out[i].Type); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(tuple[2], &out[i].Variadic); err != nil {
			t.Fatal(err)
		}
	}
	return out
}

func TestCompatInterfaceFixtures(t *testing.T) {
	for _, f := range loadFixtures(t).Interfaces {
		t.Run(f.ID, func(t *testing.T) {
			in, err := Interface(fixtureParameters(t, f.Inputs))
			if err != nil {
				t.Fatal(err)
			}
			out, err := Interface(fixtureParameters(t, f.Outputs))
			if err != nil {
				t.Fatal(err)
			}
			if got := (Strings{out, in, f.Name}); got != f.Expected {
				t.Fatalf("got %+v want %+v", got, f.Expected)
			}
		})
	}
}

func TestCompatJaroFixtures(t *testing.T) {
	for _, f := range loadFixtures(t).Jaro {
		t.Run(f.ID, func(t *testing.T) {
			got, err := JaroWinkler(f.A, f.B)
			if err != nil {
				t.Fatal(err)
			}
			want := big.NewRat(f.Expected[0], f.Expected[1])
			if got.Cmp(want) != 0 {
				t.Fatalf("got %s want %s", got, want)
			}
			reverse, err := JaroWinkler(f.B, f.A)
			if err != nil || reverse.Cmp(got) != 0 {
				t.Fatalf("asymmetric: %v %v", reverse, err)
			}
		})
	}
}

func TestCompatDiameterAndDepthFixtures(t *testing.T) {
	for _, f := range loadFixtures(t).Diameters {
		t.Run(f.ID, func(t *testing.T) {
			got, err := Diameter(context.Background(), f.Graph.graph())
			if err != nil {
				t.Fatal(err)
			}
			if got != f.Expected {
				t.Fatalf("got %d want %d", got, f.Expected)
			}
		})
	}
	for _, f := range loadFixtures(t).Depths {
		t.Run(f.ID, func(t *testing.T) {
			if got := Depth(f.Sizes[0], f.Sizes[1], f.Diameters[0], f.Diameters[1]); got != f.Expected {
				t.Fatalf("got %d want %d", got, f.Expected)
			}
		})
	}
}

func assertKernel(t *testing.T, k Kernel, w kernelExpectation) {
	t.Helper()
	if len(k.Rounds) != len(w.Weights) {
		t.Fatalf("round count %d want %d", len(k.Rounds), len(w.Weights))
	}
	for i, r := range k.Rounds {
		got := []string{r.Overlap.String(), r.Cross.String(), r.SelfA.String(), r.SelfB.String()}
		want := []string{fmt.Sprint(w.Overlap[i]), fmt.Sprint(w.Cross[i]), fmt.Sprint(w.SelfA[i]), fmt.Sprint(w.SelfB[i])}
		if r.Weight != w.Weights[i] || !slices.Equal(got, want) {
			t.Fatalf("round %d: weight=%d counts=%v want %d/%v", i, r.Weight, got, w.Weights[i], want)
		}
	}
	if k.Overlap.Cmp(big.NewRat(w.Score[0], w.Score[1])) != 0 {
		t.Fatalf("overlap=%s", k.Overlap)
	}
	if k.Cosine.Dot.Cmp(big.NewInt(w.Cosine.Numerator)) != 0 || k.Cosine.NormProduct.Cmp(big.NewInt(w.Cosine.Denominator)) != 0 {
		t.Fatalf("cosine=%+v", k.Cosine)
	}
}

func TestCompatWLFixtures(t *testing.T) {
	for _, f := range loadFixtures(t).WL {
		t.Run(f.ID, func(t *testing.T) {
			got, err := CompareWL(context.Background(), f.A.graph(), f.B.graph())
			if err != nil {
				t.Fatal(err)
			}
			if got.Depth != f.Expected.H {
				t.Fatalf("depth=%d want %d", got.Depth, f.Expected.H)
			}
			assertKernel(t, got, f.Expected)
		})
	}
}

func fixtureHistograms(a, b []map[string]uint64) ([]Histogram, []Histogram) {
	labels := map[string]int{}
	convert := func(rounds []map[string]uint64) []Histogram {
		out := make([]Histogram, len(rounds))
		for i, r := range rounds {
			out[i] = Histogram{}
			for label, count := range r {
				id, ok := labels[label]
				if !ok {
					id = len(labels) + 1
					labels[label] = id
				}
				out[i][id] = count
			}
		}
		return out
	}
	return convert(a), convert(b)
}

func TestCompatHistogramFixtures(t *testing.T) {
	for _, f := range loadFixtures(t).Histograms {
		t.Run(f.ID, func(t *testing.T) {
			a, b := fixtureHistograms(f.A, f.B)
			k := Kernel{Depth: f.H}
			for i := range a {
				overlap, cross := histogramCounts(a[i], b[i])
				_, aa := histogramCounts(a[i], a[i])
				_, bb := histogramCounts(b[i], b[i])
				k.Rounds = append(k.Rounds, Round{Weight: f.H + 1 - i, Overlap: overlap, Cross: cross, SelfA: aa, SelfB: bb})
			}
			setScores(&k, f.Sizes[0], f.Sizes[1])
			assertKernel(t, k, f.Expected)
			if k.LargerCoverage.Cmp(big.NewRat(f.Expected.Larger[0], f.Expected.Larger[1])) != 0 {
				t.Fatalf("larger coverage=%s", k.LargerCoverage)
			}
		})
	}
}

func TestCompatBoundaryFixtures(t *testing.T) {
	for _, f := range loadFixtures(t).Boundaries {
		t.Run(f.ID, func(t *testing.T) {
			got := false
			switch f.Kind {
			case "ratio":
				p := Pair{}
				got = sizeGates(&p, f.Sizes[0], f.Sizes[1], f.Threshold)
			case "rational":
				got = atLeast(big.NewRat(f.Score[0], f.Score[1]), f.Threshold)
			case "cosine":
				got = (Cosine{big.NewInt(f.RawDot), big.NewInt(f.RawNorm)}).AtLeast(f.Threshold)
			case "mean":
				sum := new(big.Rat)
				for _, r := range f.Fields {
					sum.Add(sum, big.NewRat(r[0], r[1]))
				}
				got = atLeast(sum.Quo(sum, big.NewRat(int64(len(f.Fields)), 1)), f.Threshold)
			default:
				t.Fatalf("unknown boundary kind %q", f.Kind)
			}
			if got != f.Expected {
				t.Fatalf("got %v want %v", got, f.Expected)
			}
		})
	}
}
