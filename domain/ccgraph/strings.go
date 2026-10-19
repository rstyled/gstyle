package ccgraph

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"slices"
	"unicode/utf8"
)

// Interface tuples are ordered and already resolved by the extraction adapter.
type Parameter struct {
	Role, Type string
	Variadic   bool
}

// A missing boolean is unknown, not the known false value of a Go zero field.
func (p *Parameter) UnmarshalJSON(data []byte) error {
	var wire struct {
		Role, Type *string
		Variadic   *bool
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&wire); err != nil {
		return err
	}
	if wire.Role == nil || wire.Type == nil || wire.Variadic == nil {
		return fmt.Errorf("interface tuple requires known role, type and variadic")
	}
	*p = Parameter{*wire.Role, *wire.Type, *wire.Variadic}
	return nil
}

type Strings struct{ ReturnPara, IntroPara, Name string }

var primitiveTypes = []string{"bool", "i8", "i16", "i32", "i64", "i128", "isize", "u8", "u16", "u32", "u64", "u128", "usize", "unit"}

type StringScore struct{ ReturnPara, IntroPara, Name, Mean *big.Rat }

func Interface(parameters []Parameter) (string, error) {
	tuples := make([][3]any, 0, len(parameters))
	for _, p := range parameters {
		if !slices.Contains(primitiveTypes, p.Type) {
			return "", fmt.Errorf("unsupported resolved type %q", p.Type)
		}
		if !slices.Contains([]string{"parameter", "return", "receiver", "capture"}, p.Role) {
			return "", fmt.Errorf("unsupported role %q", p.Role)
		}
		tuples = append(tuples, [3]any{p.Role, p.Type, p.Variadic})
	}
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(tuples); err != nil {
		return "", err
	}
	return string(bytes.TrimSuffix(buf.Bytes(), []byte{'\n'})), nil
}

// JaroWinkler uses scalar values, canonical orientation and exact arithmetic.
func JaroWinkler(a, b string) (*big.Rat, error) {
	if !utf8.ValidString(a) || !utf8.ValidString(b) {
		return nil, fmt.Errorf("invalid UTF-8 string")
	}
	if a == b {
		return big.NewRat(1, 1), nil
	}
	if min(len(a), len(b)) == 0 {
		return new(big.Rat), nil
	}
	x, y := orient([]rune(a), []rune(b))
	left, right := matchedRunes(x, y)
	if len(left) == 0 {
		return new(big.Rat), nil
	}
	j := jaroScore(len(x), len(y), left, right)
	boost := big.NewRat(int64(prefix(x, y)), 10)
	boost.Mul(boost, new(big.Rat).Sub(big.NewRat(1, 1), j))
	return new(big.Rat).Add(j, boost), nil
}

func orient(a, b []rune) ([]rune, []rune) {
	if len(a) > len(b) {
		return b, a
	}
	if len(a) == len(b) && slices.Compare(a, b) > 0 {
		return b, a
	}
	return a, b
}

func matchedRunes(a, b []rune) ([]rune, []rune) {
	seenA, seenB := make([]bool, len(a)), make([]bool, len(b))
	window := max(0, max(len(a), len(b))/2-1)
	for i, r := range a {
		for j := max(0, i-window); j < min(len(b), i+window+1); j++ {
			if seenB[j] || b[j] != r {
				continue
			}
			seenA[i], seenB[j] = true, true
			break
		}
	}
	return selectedRunes(a, seenA), selectedRunes(b, seenB)
}

func selectedRunes(runes []rune, seen []bool) []rune {
	var out []rune
	for i, r := range runes {
		if seen[i] {
			out = append(out, r)
		}
	}
	return out
}

func jaroScore(a, b int, left, right []rune) *big.Rat {
	m := int64(len(left))
	disagreements := int64(0)
	for i, r := range left {
		if r != right[i] {
			disagreements++
		}
	}
	sum := new(big.Rat).Add(big.NewRat(m, int64(a)), big.NewRat(m, int64(b)))
	sum.Add(sum, big.NewRat(2*m-disagreements, 2*m))
	return sum.Quo(sum, big.NewRat(3, 1))
}

func prefix(a, b []rune) int {
	n := min(4, min(len(a), len(b)))
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

func CompareStrings(a, b Strings) (StringScore, error) {
	values := [][2]string{{a.ReturnPara, b.ReturnPara}, {a.IntroPara, b.IntroPara}, {a.Name, b.Name}}
	scores := make([]*big.Rat, 3)
	sum := new(big.Rat)
	for i, pair := range values {
		score, err := JaroWinkler(pair[0], pair[1])
		if err != nil {
			return StringScore{}, err
		}
		scores[i] = score
		sum.Add(sum, score)
	}
	return StringScore{scores[0], scores[1], scores[2], sum.Quo(sum, big.NewRat(3, 1))}, nil
}
