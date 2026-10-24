package goquality

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestNilUvarintResultOwnsStorage(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import "encoding/binary"
// Mirrors captureSourceSyntax: append records to a fresh encoded header.
func Capture(records [][]byte) []byte {
 data:=binary.AppendUvarint(nil,uint64(len(records)))
 for _,record:=range records{data=append(data,record...)}
 return data
}
func Forward(records [][]byte)[]byte{return Capture(records)}
func Write(){data:=binary.AppendUvarint(nil,0);data[0]=1}
func Alias(){data:=binary.AppendUvarint(nil,1);other:=data;other[0]=1}
func Parentheses(){data:=binary.AppendUvarint((nil),1);data[0]=1}
func Shared(input []byte){data:=binary.AppendUvarint(input,1);data[0]=1}
func Spare(input []byte){data:=binary.AppendUvarint(input[:0],1);data[0]=1}
func ShadowNil(input []byte){nil:=input;data:=binary.AppendUvarint(nil,1);data[0]=1}
func Rebound(input []byte){data:=binary.AppendUvarint(nil,1);data=input;data[0]=1}
func Mixed(input []byte,flag bool){data:=binary.AppendUvarint(nil,1);if flag{data=input};data[0]=1}
func Dynamic(encode func([]byte,uint64)[]byte){data:=encode(nil,1);data[0]=1}
type Encoder struct{}
var shared=[]byte{0}
func(Encoder)AppendUvarint([]byte,uint64)[]byte{return shared}
func Shadow(){binary:=Encoder{};data:=binary.AppendUvarint(nil,1);data[0]=1}
func IO(){data:=binary.AppendUvarint(nil,1);data[0]=1;println("write")}
func Callback(encode func()uint64){data:=binary.AppendUvarint(nil,encode());data[0]=1}
var count uint64
func next()uint64{count++;return count}
func Argument(){data:=binary.AppendUvarint(nil,next());data[0]=1}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Capture", "Forward", "Write", "Alias", "Parentheses"} {
		result := resultNamed(t, report, name)
		t.Logf("%s: %+v", name, result)
		if !result.Complete || len(result.Effects) != 0 {
			t.Errorf("%s loses nil uvarint result ownership: %+v", name, result)
		}
		if name == "Forward" {
			continue
		}
		mutations := findFunction(t, facts, name).Mutations
		want := 1
		if name == "Capture" {
			want = 2 // The header assignment and backing write share one site.
		}
		if len(mutations) != want || slices.ContainsFunc(mutations, func(m quality.Mutation) bool { return m.Provenance != quality.Local }) {
			t.Errorf("%s loses its raw local write: %+v", name, mutations)
		}
	}
	for _, name := range []string{"Shared", "Spare", "ShadowNil", "Rebound", "Mixed", "Dynamic", "Shadow"} {
		mutations := findFunction(t, facts, name).Mutations
		result := resultNamed(t, report, name)
		t.Logf("%s: %+v; raw=%+v", name, result, mutations)
		if len(mutations) == 0 || mutations[len(mutations)-1].Provenance == quality.Local || len(result.Effects) == 0 {
			t.Errorf("%s hides its shared or unknown write", name)
		}
	}
	for name, kind := range map[string]quality.Kind{"IO": quality.IO, "Callback": quality.UnknownEffect, "Argument": quality.Global} {
		result := resultNamed(t, report, name)
		if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
			t.Errorf("%s loses independent %s: %+v", name, kind, result)
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
		t.Fatal("saved uvarint result facts change evaluation", err)
	}
	for _, callee := range []string{"encoding/binary.AppendUvarint", "example.com/quality-repro.Capture"} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: callee, Kind: quality.IO})
		overridden, err := quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.ContainsFunc(resultNamed(t, overridden, "Forward").Effects, func(e quality.Effect) bool { return e.Kind == quality.IO }) {
			t.Errorf("uvarint ownership hides explicit %s policy", callee)
		}
	}
}

func TestUvarintResultStorageContract(t *testing.T) {
	for _, value := range []uint64{0, 127, 128, 1<<64 - 1} {
		first := binary.AppendUvarint(nil, value)
		second := binary.AppendUvarint(nil, value)
		original := second[0]
		first[0] ^= 0x80
		if second[0] != original {
			t.Fatalf("nil destinations share storage for %d", value)
		}
		shared := make([]byte, 0, binary.MaxVarintLen64)
		result := binary.AppendUvarint(shared, value)
		result[0] ^= 0x80
		if shared[:cap(shared)][0] != result[0] {
			t.Fatalf("spare destination does not expose the control write for %d", value)
		}
	}
}
