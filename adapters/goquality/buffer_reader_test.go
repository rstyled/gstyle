package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestClosedBufferReaders(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import("bytes";"compress/flate";"io")
func Bytes(data []byte){io.ReadAll(bytes.NewBuffer(data))}
func String(data string){io.ReadAll(bytes.NewBufferString(data))}
func Local(data []byte){r:=bytes.NewBuffer(data);io.ReadAll(r)}
func Limited(data string){r:=bytes.NewBufferString(data);io.ReadAll(io.LimitReader(r,20))}
func Compressed(data []byte){r:=flate.NewReader(bytes.NewBuffer(data));defer r.Close();io.ReadAll(r)}
func Forward(data []byte){Local(data)}
type Buffer = bytes.Buffer
func Borrowed(r *Buffer){io.ReadAll(r)}
func Input(r io.Reader){io.ReadAll(r)}
func Alias(){r:=bytes.NewBuffer(nil);other:=r;io.ReadAll(other)}
func Rebound(r *bytes.Buffer){b:=bytes.NewBuffer(nil);b=r;io.ReadAll(b)}
func Reset(){r:=bytes.NewBuffer(nil);r.Reset();io.ReadAll(r)}
func Returned()*bytes.Buffer{r:=bytes.NewBuffer(nil);io.ReadAll(r);return r}
func Exposed()[]byte{r:=bytes.NewBufferString("data");data:=r.Bytes();io.ReadAll(r);return data}
func Closure(){r:=bytes.NewBuffer(nil);func(){io.ReadAll(r)}()}
func Async(){r:=bytes.NewBuffer(nil);go io.ReadAll(r)}
func DirectAsync(){go io.ReadAll(bytes.NewBuffer(nil))}
func MethodValue(){r:=bytes.NewBuffer(nil);read:=r.Read;read(nil);io.ReadAll(r)}
var saved *bytes.Buffer
func Global(){r:=bytes.NewBuffer(nil);saved=r;io.ReadAll(r)}
var counter int
func data()[]byte{counter++;return nil}
func ArgumentEffect(){io.ReadAll(bytes.NewBuffer(data()))}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Bytes", "String", "Local", "Limited", "Compressed", "Forward"} {
		result := resultNamed(t, report, name)
		if len(result.Effects) != 0 || result.DensityMilli != 0 || !result.Complete {
			t.Errorf("%s reads private buffer memory, want no effects: %+v", name, result)
		}
	}
	for _, name := range []string{"Borrowed", "Input", "Alias", "Rebound", "Reset", "Returned", "Exposed", "Closure", "Async", "DirectAsync", "MethodValue", "Global"} {
		result := resultNamed(t, report, name)
		if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == quality.IO }) {
			t.Errorf("%s must retain IO uncertainty: %+v", name, result)
		}
		for _, call := range findFunction(t, facts, name).Calls {
			if call.LocalMemoryRead {
				t.Errorf("%s has an unproved memory read: %+v", name, call)
			}
		}
	}
	argument := resultNamed(t, report, "ArgumentEffect")
	if len(argument.Effects) != 1 || argument.Effects[0].Kind != quality.Global {
		t.Errorf("reader proof must retain constructor argument effects: %+v", argument)
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
		t.Fatal("saved buffer reader proof changes evaluation", err)
	}
	for _, kind := range []quality.Kind{quality.IO, quality.Global, quality.None} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "io.ReadAll", Kind: kind})
		overridden, err := quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		result := resultNamed(t, overridden, "Forward")
		if kind == quality.None {
			if len(result.Effects) != 0 {
				t.Errorf("pure policy override lost: %+v", result)
			}
			continue
		}
		if len(result.Effects) != 1 || result.Effects[0].Kind != kind {
			t.Errorf("%s policy override lost: %+v", kind, result)
		}
	}
}
