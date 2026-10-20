// Package ccgraph implements compat.ccgraph.strict/0.1.0 matching stages.
// Inputs are already validated projections. This package does not certify
// compiler extraction, CFG correspondence or interchange-schema conformance.
package ccgraph

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"strings"
)

const Profile = "compat.ccgraph.strict/0.1.0"

const ContractCommit = "04d9aa0db8b1f8e4a3fcad16622e2a4e3bc3e53f"

const Overlap = "smaller_graph_overlap"

const CrossCosine = "cross_product_cosine"

// Decimal preserves the exact configuration lexeme. Decisions use rationals.
type Decimal string

func (d *Decimal) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	next := Decimal(value)
	if _, err := next.rational(); err != nil {
		return err
	}
	*d = next
	return nil
}

func (d Decimal) rational() (*big.Rat, error) {
	whole, fraction, point := strings.Cut(string(d), ".")
	if !decimalDigits(whole) || (point && !decimalDigits(fraction)) {
		return nil, fmt.Errorf("invalid decimal %q", d)
	}
	r, ok := new(big.Rat).SetString(string(d))
	if !ok || r.Cmp(big.NewRat(1, 1)) > 0 {
		return nil, fmt.Errorf("threshold outside [0,1]: %q", d)
	}
	return r, nil
}

type Config struct {
	Size      Decimal `json:"minimum_size_ratio"`
	Numerical Decimal `json:"numerical_threshold"`
	Strings   Decimal `json:"string_threshold"`
	Final     Decimal `json:"final_threshold"`
	Scoring   string  `json:"scoring"`
}

func (c Config) Validate() error {
	if c.Scoring != Overlap && c.Scoring != CrossCosine {
		return fmt.Errorf("unknown scoring %q", c.Scoring)
	}
	for _, d := range []Decimal{c.Size, c.Numerical, c.Strings, c.Final} {
		if _, err := d.rational(); err != nil {
			return err
		}
	}
	return nil
}

func decimalDigits(text string) bool {
	return text != "" && strings.Trim(text, "0123456789") == ""
}

// Cosine records exact nonnegative components, avoiding square-root drift.
type Cosine struct{ Dot, NormProduct *big.Int }

func (c Cosine) AtLeast(threshold Decimal) bool {
	t, err := threshold.rational()
	if err != nil {
		return false
	}
	if c.Dot == nil || c.NormProduct == nil {
		return false
	}
	if c.Dot.Sign() < 0 || c.NormProduct.Sign() <= 0 {
		return false
	}
	left := new(big.Int).Mul(c.Dot, c.Dot)
	left.Mul(left, new(big.Int).Mul(t.Denom(), t.Denom()))
	right := new(big.Int).Mul(t.Num(), t.Num())
	right.Mul(right, c.NormProduct)
	return left.Cmp(right) >= 0
}

func DefaultConfig() Config { return Config{"0.8", "0.9", "0.5", "0.9", Overlap} }

// ParseConfig applies only declared overrides and rejects unknown fields.
func ParseConfig(data []byte) (Config, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return Config{}, fmt.Errorf("configuration must be an object")
	}
	c := DefaultConfig()
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return Config{}, err
	}
	if err := configEnd(d); err != nil {
		return Config{}, err
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

func configEnd(d *json.Decoder) error {
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return fmt.Errorf("trailing or malformed configuration: %v", err)
	}
	return nil
}

func atLeast(score *big.Rat, threshold Decimal) bool {
	t, _ := threshold.rational() // validated at the run boundary
	return score.Cmp(t) >= 0
}
