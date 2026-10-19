package goquality

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestSourceHelpersRetainConfiguredInterfacePolicy(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
 import("go/constant";"io";"example.com/quality-repro/lib")
 func Caller(value lib.Value)(bool,bool){return value.Bool()}
 func Nested(value lib.Value)(bool,bool){return lib.Wrap(value)}
 func Repeated(value lib.Value){lib.Wrap(value);lib.Wrap(value)}
 func Direct(value constant.Value)constant.Kind{return value.Kind()}
 func Callback(value lib.Callback){lib.Invoke(value)}
 func IOCaller(reader io.Reader){lib.Read(reader)}
 func DirectIO(reader io.Reader){reader.Read(nil)}
 `))
	path := filepath.Join(root, "lib", "helper.go")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`package lib
 import("go/constant";"io")
 type Value struct{Scalar constant.Value}
 func(value Value)Bool()(bool,bool){
  if value.Scalar==nil || value.Scalar.Kind()!=constant.Bool {return false,false}
  return constant.BoolVal(value.Scalar),true
 }
 func Wrap(value Value)(bool,bool){return value.Bool()}
 type Callback interface{Run()}
 func Invoke(value Callback){value.Run()}
 func Read(reader io.Reader){reader.Read(nil)}
 `), 0600); err != nil {
		t.Fatal(err)
	}
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	caller := findFunction(t, facts, "Caller")
	if len(caller.Calls) != 1 || !caller.Calls[0].Local {
		t.Fatalf("modeled interface policy blocks source helper: %+v", caller.Calls)
	}
	retained := false
	for _, helper := range caller.Helpers {
		for _, call := range helper.Calls {
			retained = retained || (call.Callee == "go/constant.Value.Kind" && call.Dynamic)
		}
	}
	if !retained {
		t.Fatal("source helper loses dynamic interface call evidence")
	}
	for _, name := range []string{"Callback", "IOCaller"} {
		f := findFunction(t, facts, name)
		if len(f.Calls) != 1 || f.Calls[0].Local {
			t.Fatalf("unmodeled or IO interface promoted: %s %+v", name, f.Calls)
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
	report, err := quality.Evaluate(restored, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Caller", "Nested", "Repeated", "Direct"} {
		got := resultNamed(t, report, name)
		if !got.Complete || got.UnclassifiedCalls != 0 || len(got.Effects) != 0 {
			t.Fatalf("configured read-only interface: %s %+v", name, got)
		}
	}
	for _, name := range []string{"Callback", "IOCaller"} {
		got := resultNamed(t, report, name)
		if got.Complete || got.UnclassifiedCalls != 1 || len(got.Effects) != 1 || got.Effects[0].Kind != quality.UnknownEffect {
			t.Fatalf("unmodeled source call loses uncertainty: %s %+v", name, got)
		}
	}
	directIO := resultNamed(t, report, "DirectIO")
	if len(directIO.Effects) != 1 || directIO.Effects[0].Kind != quality.IO {
		t.Fatalf("direct IO interface loses effect: %+v", directIO)
	}
	for _, kind := range []quality.Kind{quality.IO, quality.UnknownEffect} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "go/constant.Value.Kind", Kind: kind})
		changed, err := quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			name  string
			count int
		}{{"Caller", 1}, {"Nested", 1}, {"Repeated", 2}, {"Direct", 1}} {
			got := resultNamed(t, changed, tc.name)
			if len(got.Effects) != tc.count {
				t.Fatalf("cached source body hides override: %s %+v", tc.name, got)
			}
			for _, effect := range got.Effects {
				if effect.Kind != kind {
					t.Fatalf("override classification lost: %+v", effect)
				}
			}
			if got.Complete != (kind != quality.UnknownEffect) {
				t.Fatalf("override coverage lost: %s %+v", tc.name, got)
			}
		}
	}
}
