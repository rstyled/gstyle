package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestHTTPHeaderSetKeepsMutationWithoutNetwork(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import ("io"; "net/http")
type Alias = http.Header
func Set(header http.Header) { header.Set("Content-Type", "application/json") }
func Aliased(header Alias) { header.Set("Content-Type", "application/json") }
func Expression(header http.Header) { http.Header.Set(header, "Content-Type", "application/json") }
func Forward(header http.Header) { Set(header) }
func Response(writer http.ResponseWriter) { writer.Header().Set("Content-Type", "application/json") }
func Get() (*http.Response, error) { return http.Get("https://example.com") }
func Write(header http.Header, out io.Writer) error { return header.Write(out) }
func WriteSubset(header http.Header, out io.Writer) error { return header.WriteSubset(out, nil) }
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Set", "Aliased", "Expression", "Forward", "Response"} {
		result := resultNamed(t, report, name)
		if len(result.Effects) != 1 || result.Effects[0].Kind != quality.MutationEffect || !result.Complete {
			t.Errorf("%s changes a map, want one mutation without network: %+v", name, result)
		}
	}
	for _, name := range []string{"Get", "Write", "WriteSubset"} {
		result := resultNamed(t, report, name)
		if len(result.Effects) != 1 || result.Effects[0].Kind != quality.Network {
			t.Errorf("%s must retain its external effect: %+v", name, result)
		}
	}
	encoded, err := json.Marshal(facts)
	if err != nil {
		t.Fatal(err)
	}
	var restored []quality.Function
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	replay, err := quality.Evaluate(restored, quality.DefaultConfig())
	if err != nil || !reflect.DeepEqual(report, replay) {
		t.Fatal("saved header effects changed", err)
	}
	for _, kind := range []quality.Kind{quality.Network, quality.None} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "net/http.Header.Set", Kind: kind})
		overridden, err := quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"Set", "Aliased", "Expression", "Forward", "Response"} {
			result := resultNamed(t, overridden, name)
			if kind == quality.None {
				if len(result.Effects) != 0 {
					t.Errorf("%s lost pure policy override: %+v", name, result)
				}
				continue
			}
			if len(result.Effects) != 1 || result.Effects[0].Kind != kind {
				t.Errorf("%s lost %s policy override: %+v", name, kind, result)
			}
		}
	}
}
