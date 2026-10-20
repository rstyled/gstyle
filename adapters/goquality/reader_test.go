package goquality

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestClosedMemoryReaders(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import("bytes";"compress/flate";"io";"strings";"example.com/quality-repro/lib")
func Cross()string{return lib.Closed("data")}
func Direct(){io.ReadAll(strings.NewReader("data"))}
func Bytes(){io.ReadAll(bytes.NewReader([]byte("data")))}
func Limited(){io.ReadAll(io.LimitReader(flate.NewReader(strings.NewReader("data")),20))}
func Closed(data string){r:=flate.NewReader(strings.NewReader(data));defer r.Close();io.ReadAll(io.LimitReader(r,20))}
var global io.ReadCloser
func Global(){global=flate.NewReader(strings.NewReader("data"));io.ReadAll(global)}
func DirectAsync(){go io.ReadAll(strings.NewReader("data"))}
func DirectClosure(){func(){io.ReadAll(strings.NewReader("data"))}()}
func Borrowed(r io.Reader){io.ReadAll(r)}
func Alias(){r:=flate.NewReader(strings.NewReader("data"));other:=r;io.ReadAll(other)}
func Rebound(input io.Reader){r:=flate.NewReader(strings.NewReader("data"));r=io.NopCloser(input);io.ReadAll(r)}
func Reset(input io.Reader){r:=flate.NewReader(strings.NewReader("data"));r.(flate.Resetter).Reset(input,nil);io.ReadAll(r)}
func Returned()(io.ReadCloser,error){r:=flate.NewReader(strings.NewReader("data"));io.ReadAll(r);return r,nil}
func EscapedChain()io.Reader{r:=flate.NewReader(strings.NewReader("data"));io.ReadAll(r);return io.LimitReader(r,20)}
func Closure(){r:=flate.NewReader(strings.NewReader("data"));func(){io.ReadAll(r)}()}
func Async(){r:=flate.NewReader(strings.NewReader("data"));go io.ReadAll(r)}
func MethodValue(){r:=flate.NewReader(strings.NewReader("data"));close:=r.Close;close();io.ReadAll(r)}
func Dictionary(){io.ReadAll(flate.NewReaderDict(strings.NewReader("data"),nil))}
`))
	if err := os.Mkdir(filepath.Join(root, "lib"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "lib", "reader.go"), []byte(`package lib
import("compress/flate";"io";"strings")
func Closed(data string)string{r:=flate.NewReader(strings.NewReader(data));defer r.Close();out,_:=io.ReadAll(io.LimitReader(r,1024));return string(out)}
`), 0600); err != nil {
		t.Fatal(err)
	}
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Direct", "Bytes", "Limited", "Closed", "Borrowed", "Global", "DirectAsync", "DirectClosure", "Alias", "Rebound", "Reset", "Returned", "EscapedChain", "Closure", "Async", "MethodValue", "Dictionary"} {
		t.Run(name, func(t *testing.T) {
			fact := findFunction(t, facts, name)
			want := name == "Direct" || name == "Bytes" || name == "Limited" || name == "Closed"
			for _, call := range fact.Calls {
				if call.Callee != "io.ReadAll" && call.Callee != "io.Closer.Close" {
					continue
				}
				if call.LocalMemoryRead != want {
					t.Fatalf("%s: LocalMemoryRead=%v, want %v", call.Callee, call.LocalMemoryRead, want)
				}
			}
		})
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	got := resultNamed(t, report, "Cross")
	if !got.Complete || len(got.Effects) != 0 {
		t.Fatalf("cross-package memory read: %+v", got)
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
		t.Fatal("saved reader proof changes evaluation", err)
	}
	config := quality.DefaultConfig()
	config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "io.ReadAll", Kind: quality.IO})
	changed, err := quality.Evaluate(restored, config)
	if err != nil {
		t.Fatal(err)
	}
	if got := resultNamed(t, changed, "Cross"); !slices.ContainsFunc(got.Effects, func(e quality.Effect) bool { return e.Kind == quality.IO }) {
		t.Fatalf("cross-package override lost: %+v", got)
	}
}
