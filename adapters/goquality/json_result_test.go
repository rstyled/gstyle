package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestJSONMarshalResultOwnsBytesWithoutHidingEffects(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import("encoding/json";"slices")
// Mirrors application/qualitygate.runWrite's newline append.
func Indented(value string)([]byte,error){
 data,err:=json.MarshalIndent(value,"","  ")
 if err!=nil{return nil,err}
 data=append(data,'\n')
 return data,nil
}
func Plain(){data,_:=json.Marshal(1);data[0]='2';slices.Reverse(data)}
func Existing(){data:=[]byte{};var err error;data,err=json.Marshal(1);_ =err;data[0]='2'}
func Aliased(){data,_:=json.Marshal(1);alias:=data;alias[0]='2'}
type Custom int
func(Custom)MarshalJSON()([]byte,error){println("callback");return []byte("1"),nil}
func Callback(){data,_:=json.Marshal(Custom(1));data[0]='2'}
func CallbackIndent(){data,_:=json.MarshalIndent(Custom(1),"","");data[0]='2'}
var counter int
func next()int{counter++;return 1}
func Argument(){data,_:=json.Marshal(next());data[0]='2'}
func Shared(data []byte){data[0]='2'}
func Mixed(input []byte,flag bool){data,_:=json.Marshal(1);if flag{data=input};data[0]='2'}
func Rebound(input []byte){data,_:=json.Marshal(1);data=input;data[0]='2'}
func Dynamic(encode func(any)([]byte,error)){data,_:=encode(1);data[0]='2'}
type Encoder struct{}
var borrowed []byte
func(Encoder)Marshal(any)([]byte,error){return borrowed,nil}
func Shadow(){json:=Encoder{};data,_:=json.Marshal(1);data[0]='2'}
func ErrorResult(){_,err:=json.Marshal(1);failure:=err.(*json.UnsupportedValueError);failure.Str="changed"}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Indented", "Plain", "Existing", "Aliased", "Callback", "CallbackIndent", "Argument"} {
		function := findFunction(t, facts, name)
		for _, mutation := range function.Mutations {
			if mutation.Provenance != quality.Local {
				t.Errorf("%s must own returned bytes: %+v", name, mutation)
			}
		}
	}
	controls := []string{"Shared", "Mixed", "Rebound", "Dynamic", "Shadow", "ErrorResult"}
	for _, name := range controls {
		function := findFunction(t, facts, name)
		mutation := function.Mutations[len(function.Mutations)-1]
		if mutation.Provenance == quality.Local {
			t.Errorf("%s loses borrowed or unknown storage: %+v", name, mutation)
		}
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Indented", "Plain", "Existing", "Aliased"} {
		result := resultNamed(t, report, name)
		if !result.Complete || len(result.Effects) != 0 {
			t.Errorf("%s fresh byte writes must stay local: %+v", name, result)
		}
	}
	for _, name := range append(controls, "Callback", "CallbackIndent", "Argument") {
		if result := resultNamed(t, report, name); len(result.Effects) == 0 {
			t.Errorf("%s loses call or storage effects: %+v", name, result)
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
		t.Fatal("saved JSON result ownership changes evaluation", err)
	}
	for callee, name := range map[string]string{"encoding/json.Marshal": "Plain", "encoding/json.MarshalIndent": "Indented"} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: callee, Kind: quality.IO})
		overridden, err := quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		result := resultNamed(t, overridden, name)
		if len(result.Effects) != 1 || result.Effects[0].Kind != quality.IO {
			t.Errorf("%s hides policy override: %+v", name, result)
		}
	}
}
