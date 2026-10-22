package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestLocalJSONDecodeReturnsAddressAfterConstruction(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import("encoding/json";"fmt")
type Cell struct{Symbol,Source string}
type Environment struct{Owner string;Cells []Cell}
type Alias = Environment
// Mirrors domain/bindingflow.decodeCaptureEnvironment in the master self-scan.
func Checked(data []byte)(*Environment,error){
 var environment Environment
 if err:=json.Unmarshal(data,&environment);err!=nil{return nil,err}
 if environment.Owner==""||environment.Cells==nil{return nil,fmt.Errorf("missing environment")}
 return &environment,nil
}
func Parenthesized(data []byte)*Alias{var result Alias;json.Unmarshal(data,&result);return ((&((result))))}
func Scalar(data []byte)*string{var result string;json.Unmarshal(data,&result);return &result}
func Forward(data []byte)(*Environment,error){return Checked(data)}
var escaped *Environment
func Published(data []byte)*Environment{var result Environment;escaped=&result;json.Unmarshal(data,&result);return &result}
func Aliased(data []byte)*Environment{var result Environment;p:=&result;json.Unmarshal(data,&result);return p}
func FieldAddress(data []byte)*string{var result Environment;json.Unmarshal(data,&result);return &result.Owner}
func Preloaded(data []byte,cells []Cell)*Environment{result:=Environment{Cells:cells};json.Unmarshal(data,&result);return &result}
func Deferred(data []byte)*Environment{var result Environment;defer json.Unmarshal(data,&result);return &result}
func DeferredPublish(data []byte)*Environment{var result Environment;defer func(){escaped=&result}();json.Unmarshal(data,&result);return &result}
func Concurrent(data []byte)*Environment{var result Environment;go json.Unmarshal(data,&result);return &result}
func Closure(data []byte)*Environment{var result Environment;f:=func()*Environment{return &result};json.Unmarshal(data,&result);return f()}
func Loop(data []byte)*Environment{var result Environment;for i:=0;i<2;i++{json.Unmarshal(data,&result)};return &result}
func Reused(data []byte)*Environment{var result Environment;json.Unmarshal(data,&result);json.Unmarshal(data,&result);return &result}
func Method(data []byte)*Environment{var result Environment;result.Publish();json.Unmarshal(data,&result);return &result}
func(p *Environment)Publish(){escaped=p}
type Callback string
func(*Callback)UnmarshalJSON([]byte)error{println("callback");return nil}
type Text string
func(*Text)UnmarshalText([]byte)error{println("callback");return nil}
func Custom(data []byte)*Callback{var result Callback;json.Unmarshal(data,&result);return &result}
func Nested(data []byte)*struct{Value Callback}{var result struct{Value Callback};json.Unmarshal(data,&result);return &result}
func TextCallback(data []byte)*Text{var result Text;json.Unmarshal(data,&result);return &result}
func Open(data []byte)*any{var result any;json.Unmarshal(data,&result);return &result}
var counter int
func next()[]byte{counter++;return nil}
func ArgumentEffect()*Environment{var result Environment;json.Unmarshal(next(),&result);return &result}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Checked", "Parenthesized", "Scalar", "Forward"} {
		result := resultNamed(t, report, name)
		if !result.Complete || len(result.Effects) != 0 || result.DensityMilli != 0 {
			t.Errorf("%s constructs private storage before returning its address: %+v", name, result)
		}
	}
	for _, name := range []string{"Published", "Aliased", "FieldAddress", "Preloaded", "Deferred", "DeferredPublish", "Concurrent", "Closure", "Loop", "Reused", "Method", "Custom", "Nested", "TextCallback", "Open"} {
		result := resultNamed(t, report, name)
		if result.Complete || !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == quality.UnknownEffect }) {
			t.Errorf("%s loses decode uncertainty: %+v", name, result)
		}
		for _, call := range findFunction(t, facts, name).Calls {
			if call.LocalJSONDecode {
				t.Errorf("%s has an unproved local decode: %+v", name, call)
			}
		}
	}
	argument := resultNamed(t, report, "ArgumentEffect")
	if !slices.ContainsFunc(argument.Effects, func(e quality.Effect) bool { return e.Kind == quality.Global }) {
		t.Errorf("argument effect is lost: %+v", argument)
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
		t.Fatal("saved address-return proof changes evaluation", err)
	}
	for _, kind := range []quality.Kind{quality.IO, quality.UnknownEffect, quality.MutationEffect, quality.None} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "encoding/json.Unmarshal", Kind: kind})
		overridden, err := quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		result := resultNamed(t, overridden, "Forward")
		if kind == quality.None {
			if len(result.Effects) != 0 {
				t.Errorf("none policy override is lost: %+v", result)
			}
			continue
		}
		if len(result.Effects) != 1 || result.Effects[0].Kind != kind {
			t.Errorf("%s policy override is lost: %+v", kind, result)
		}
	}
}
