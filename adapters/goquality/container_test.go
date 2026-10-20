package goquality

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestContainerStorageOwnership(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import("cmp";"bytes";"encoding/json";"maps";"slices";"sort";"strings")
var shared = []int{2,1}
var counter int
type Numbers []int
type Text string
type Runes []rune
type Seed struct{}
func(Seed) seeds(input []int) []int {var out []int;for _,v:=range input {out=append(out,v)};return out}
func filled(n int) []int {out:=make([]int,n);for i:=range out {out[i]=i};return out}
func mapping(input map[int]int) map[int]int {out:=make(map[int]int);for k,v:=range input {out[k]=v};return out}
func mixedReturn(input []int, flag bool) []int {if flag{return input};out:=make([]int,1);return out}
func publishedReturn() []int {out:=make([]int,1);shared=out;return out}
func writtenReturn(dst *[]int) []int {out:=make([]int,1);*dst=out;return out}
var leakedElement *int
func retain(out []int){shared=out}
func escapedReturn() []int {out:=make([]int,1);retain(out);return out}
func addressedReturn() []int {out:=make([]int,1);leakedElement=&out[0];return out}
func closureReturn() []int {out:=make([]int,1);f:=func(){out[0]=1};_ =f;return out}
func deferredReturn(input []int) []int {out:=make([]int,1);defer copy(out,input);return out}
func outputReturn() []int {out:=make([]int,1);println("output");return out}
func globalReturn() []int {shared=make([]int,1);return shared}
func formalReturn(input []int,flag bool) []int {if flag{input=make([]int,1)};return input}
func ReturnedGlobal(){out:=globalReturn();slices.Reverse(out)}
func ReturnedFormal(){out:=formalReturn(shared,false);slices.Reverse(out)}
func sortedReturn() []int {out:=[]int{2,1};slices.Sort(out);out=slices.Delete(out,0,1);return out}
func copiedReturn(input map[int]int)map[int]int{out:=make(map[int]int);maps.Copy(out,input);return out}
func aliasOperationReturn()[]int{out:=make([]int,2);alias:=slices.Delete(out,0,1);shared=alias;return out}
var sharedNested [][]int
func insertedReturn()[]int{out:=make([]int,1);sharedNested=slices.Insert(sharedNested,0,out);return out}
func ReturnedSorted(){out:=sortedReturn();slices.Reverse(out)}
func ReturnedCopied(){maps.Copy(copiedReturn(nil),map[int]int{})}
func ReturnedOperationAlias(){out:=aliasOperationReturn();slices.Reverse(out)}
func ReturnedInserted(){out:=insertedReturn();slices.Reverse(out)}
func recursiveReturn(input *int,n int)[]*int{if n<=0{return []*int{input}};return append(recursiveReturn(input,n-1),recursiveReturn(input,n-1)...)}
func recursiveBorrowed(input []int,n int)[]int{if n<=0{return input};return append(recursiveBorrowed(input,n-1),1)}
func recursiveEscaped(n int)[]int{if n<=0{out:=make([]int,1);shared=out;return out};return append(recursiveEscaped(n-1),1)}
func relativeAppend(out []int)[]int{return append(out,1)}
func appendOther(dummy int,out []int)[]int{return append(out,dummy)}
func ReturnedRelativeSelf(){out:=make([]int,1);out=appendOther(0,out);slices.Reverse(out)}
func ReturnedRelativeSelfBorrowed(out []int){out=appendOther(0,out);slices.Reverse(out)}
func relativeEffect(out []int)[]int{counter++;return append(out,1)}
func relativeMixed(out,other []int,flag bool)[]int{if flag{return other};return append(out,1)}
func relativePublished(out []int)[]int{shared=out;return append(out,1)}
func relativeAliased(out []int)[]int{alias:=out;shared=alias;return append(out,1)}
func relativeAddress(out []int)[]int{leakedElement=&out[0];return out}
func relativeNamed(out []int)(result []int){return append(out,1)}
func relativeReferences(out []*int,input *int)[]*int{return append(out,input)}
func ReturnedRelativeNil(){out:=relativeAppend(nil);slices.Reverse(out)}
func ReturnedRelativeLocal(){input:=make([]int,1);out:=relativeAppend(input);slices.Reverse(out)}
func ReturnedRelativeRecursive(){out:=recursiveBorrowed(nil,1);slices.Reverse(out)}
func ReturnedRelativeBorrowed(input []int){out:=relativeAppend(input);slices.Reverse(out)}
func ReturnedRelativeMixed(){out:=relativeMixed(nil,shared,true);slices.Reverse(out)}
func ReturnedRelativePublished(){out:=relativePublished(nil);slices.Reverse(out)}
func ReturnedRelativeAliased(){out:=relativeAliased(nil);slices.Reverse(out)}
func ReturnedRelativeAddress(){out:=relativeAddress(make([]int,1));slices.Reverse(out)}
func ReturnedRelativeNamed(){out:=relativeNamed(nil);slices.Reverse(out)}
func ReturnedRelativeEffect(){out:=relativeEffect(nil);slices.Reverse(out)}
func ReturnedRelativePointee(input *int){out:=relativeReferences(nil,input);*out[0]=1}
func finalAppendReturn()[]int{out:=make([]int,1);return append(out,1)}
func directMakeReturn()[]int{return make([]int,1)}
func directCloneReturn(input []int)[]int{return slices.Clone(input)}
func ReturnedRecursive(input *int){out:=recursiveReturn(input,1);slices.Reverse(out)}
func ReturnedRecursivePointee(input *int){out:=recursiveReturn(input,1);*out[0]=1}
func ReturnedRecursiveBorrowed(){out:=recursiveBorrowed(shared,1);slices.Reverse(out)}
func ReturnedRecursiveEscaped(){out:=recursiveEscaped(1);slices.Reverse(out)}
func ReturnedFinalAppend(){out:=finalAppendReturn();slices.Reverse(out)}
func ReturnedDirectMake(){slices.Reverse(directMakeReturn())}
func ReturnedDirectClone(input []int){slices.Reverse(directCloneReturn(input))}
func referenceReturn(input *int) []*int {out:=make([]*int,1);out[0]=input;return out}
func decodedReturn(data []byte) []struct{Values []int} {var out []struct{Values []int};if err:=json.Unmarshal(data,&out);err!=nil{panic(err)};return out}
func calledReturn() []int {out:=make([]int,1);out[0]=bound();return out}
func viewReturn() []int {out:=make([]int,1);alias:=out[:];_ =alias;return out}
func namedReturn()(out []int){out=make([]int,1);defer func(){shared=out}();return out}
func fakeUnmarshal(data []byte,out *[]int){shared=*out}
func fakeDecodeReturn() []int {var out []int;fakeUnmarshal(nil,&out);return out}
func ReturnedReferences(input *int){out:=referenceReturn(input);slices.Reverse(out)}
func ReturnedPointee(input *int){out:=referenceReturn(input);*out[0]=1}
func ReturnedDecoded(){out:=decodedReturn(nil);slices.Reverse(out)}
func ReturnedCalled(){out:=calledReturn();slices.Reverse(out)}
func ReturnedView(){out:=viewReturn();slices.Reverse(out)}
func ReturnedNamed(){out:=namedReturn();slices.Reverse(out)}
func ReturnedFakeDecode(){out:=fakeDecodeReturn();slices.Reverse(out)}
func SelfView(){out:=make([]int,3);out=(out)[1:2:3];slices.Reverse(out)}
func SelfViewAppend(){out:=[]int{1,2};out=out[1:];out=append(out,3);slices.Reverse(out)}
func SharedView(out []int){out=out[1:];slices.Reverse(out)}
func MixedView(input []int, flag bool){out:=input;if flag{out=make([]int,3)};out=out[1:];slices.Reverse(out)}
func UnknownView(get func()[]int){out:=get();out=out[1:];slices.Reverse(out)}
func ViewBounds(){out:=make([]int,3);out=out[bound():];slices.Reverse(out)}
func bound()int{counter++;return 0}
func ReturnedQueue(input []int){out:=(Seed{}).seeds(input);out=append(out,1);slices.Reverse(out)}
func ReturnedFilled(){out:=filled(2);slices.Reverse(out)}
func ReturnedMap(){maps.Copy(mapping(nil),map[int]int{1:1})}
func ReturnedMixed(){out:=mixedReturn(shared,true);slices.Reverse(out)}
func ReturnedPublished(){out:=publishedReturn();slices.Reverse(out)}
func ReturnedWritten(){out:=writtenReturn(new([]int));slices.Reverse(out)}
func ReturnedEscape(){out:=escapedReturn();slices.Reverse(out)}
func ReturnedAddress(){out:=addressedReturn();slices.Reverse(out)}
func ReturnedClosure(){out:=closureReturn();slices.Reverse(out)}
func ReturnedDeferred(){out:=deferredReturn(shared);slices.Reverse(out)}
func ReturnedOutput(){out:=outputReturn();slices.Reverse(out)}
func FreshContainers(text string, data []byte, values []int){
slices.Reverse(strings.Split(text,","));slices.Reverse(strings.SplitN(text,",",1))
slices.Reverse(strings.SplitAfter(text,","));slices.Reverse(strings.SplitAfterN(text,",",1));slices.Reverse(strings.Fields(text))
slices.Reverse(bytes.Clone(data));slices.Reverse(bytes.Repeat(data,1))
slices.Reverse(bytes.Split(data,[]byte(",")));slices.Reverse(bytes.SplitN(data,[]byte(","),1))
slices.Reverse(bytes.SplitAfter(data,[]byte(",")));slices.Reverse(bytes.SplitAfterN(data,[]byte(","),1));slices.Reverse(bytes.Fields(data))
slices.Reverse(slices.Concat(values));slices.Reverse(slices.Repeat(values,1))
}
func BorrowedBytes(data []byte){values:=bytes.Split(data,[]byte(","));values[0][0]=1}
func BorrowedConcat(data []*int){values:=slices.Concat(data);*values[0]=1}
func CallbackFields(text string, test func(rune)bool){values:=strings.FieldsFunc(text,test);slices.Reverse(values)}
func FreshBytes(text string){values:=[]byte(text);slices.Reverse(values)}
func FreshRunes(text string){values:=[]rune(text);slices.Reverse(values)}
func NamedString(text Text){values:=Runes(text);slices.Reverse(values)}
func NilClone(src []int){values:=append([]int(nil),src...);slices.Sort(values)}
func NilSelf(src []int){values:=[]int(nil);values=append(values,src...);slices.Sort(values)}
func NilSort(){slices.Sort([]int(nil))}
func NilDirect(){slices.Reverse[[]int](nil)}
func OrdinaryConversion(values []int){converted:=Numbers(values);slices.Sort(converted)}
func ShadowNil(values []int){nil:=values;slices.Sort(nil)}
func NilFunction(){var callback func();callback()}
func ShallowNilClone(src []*int){values:=append([]*int(nil),src...);*values[0]=1}
func LocalSort(){values:=[]int{2,1}; slices.Sort(values)}
func NamedSort(){values:=Numbers{2,1}; slices.Sort(values)}
func LocalCopy(src map[int]int){dst:=make(map[int]int); maps.Copy(dst,src)}
func LocalReverse(){values:=[]int{2,1}; slices.Reverse(values)}
func LocalCompact(){values:=[]int{1,1}; _=slices.Compact(values)}
func LocalDelete(){values:=[]int{2,1}; _=slices.Delete(values,0,1)}
func LocalInsert(){values:=[]int{2,1}; _=slices.Insert(values,0,3)}
func LocalReplace(){values:=[]int{2,1}; _=slices.Replace(values,0,1,3)}
func LocalInts(){values:=[]int{2,1}; sort.Ints(values)}
func LocalFloats(){values:=[]float64{2,1}; sort.Float64s(values)}
func LocalStrings(){values:=[]string{"b","a"}; sort.Strings(values)}
func LocalAlias(){values:=[]int{2,1}; alias:=values[1:]; slices.Sort(alias)}
func LocalAppend(){values:=append([]int{2,1},3); slices.Sort(values)}
func ForcedAppend(input []int){out:=append(input[:len(input):len(input)],1);slices.Sort(out)}
func ForcedBound(input []int,n int){out:=append(input[1:n:n],1);slices.Sort(out)}
func ForcedConstant(input []int){out:=append(input[:1+1:2],1);slices.Sort(out)}
func forcedReturn(input []int)[]int{return append(input[:len(input):len(input)],1)}
func ReturnedForced(input []int){out:=forcedReturn(input);slices.Sort(out)}
func capped(input []int)[]int{return input[:len(input):len(input)]}
func cappedBranches(input []int,flag bool)[]int{if flag{return input[:1:1]};return input[:len(input):len(input)]}
func cappedMixed(input []int,flag bool)[]int{if flag{return input};return input[:len(input):len(input)]}
func cappedNamed(input []int)(out []int){out=input[:1:1];defer func(){out=input}();return out}
func cappedEffect(input []int)[]int{input[0]++;return input[:len(input):len(input)]}
func cappedReferences(input []*int)[]*int{return input[:len(input):len(input)]}
func cappedGeneric[T any](input []T)[]T{return input[:1:1]}
func cappedAppend(input []int)[]int{return append(capped(input),1)}
func ReturnedCappedAppend(input []int){out:=cappedAppend(input);slices.Sort(out)}
func ReturnedCapped(input []int){out:=append(capped(input),1);slices.Sort(out)}
func ReturnedCappedBranches(input []int){out:=append(cappedBranches(input,true),1);slices.Sort(out)}
func ReturnedCappedGeneric(input []int){out:=append(cappedGeneric(input),1);slices.Sort(out)}
func ReturnedCappedDirect(input []int){slices.Sort(capped(input))}
func ReturnedCappedMixed(input []int){out:=append(cappedMixed(input,true),1);slices.Sort(out)}
func ReturnedCappedNamed(input []int){out:=append(cappedNamed(input),1);slices.Sort(out)}
func ReturnedCappedDynamic(input []int, get func([]int)[]int){out:=append(get(input),1);slices.Sort(out)}
func ReturnedCappedEmpty(input,extra []int){out:=append(capped(input),extra...);slices.Sort(out)}
func ReturnedCappedEffect(input []int){out:=append(cappedEffect(input),1);slices.Sort(out)}
func ReturnedCappedPointee(input []*int){out:=append(cappedReferences(input),nil);*out[0]=1}
func LocalCapped(input []int){out:=capped(input);out=append(out,1);slices.Sort(out)}
func LocalCappedView(input []int){out:=input[:len(input):len(input)];out=append(out,1);slices.Sort(out)}
func LocalCappedAlias(input []int){out:=capped(input);alias:=out;out=input;alias=append(alias,1);slices.Sort(alias)}
func LocalCappedReset(input []int,n int){for range n{out:=capped(input);out=append(out,1);slices.Sort(out)}}
func LocalCappedTwice(input []int){out:=capped(input);out=append(out,1);out=append(out,2);slices.Sort(out)}
func LocalCappedBefore(input []int){out:=capped(input);out[0]=1;out=append(out,2);out[0]=3}
func LocalCappedMixed(input []int,flag bool){out:=capped(input);if flag{out=input};out=append(out,1);slices.Sort(out)}
func LocalCappedLoop(input []int,n int){out:=capped(input);for range n{out=append(out,1);slices.Sort(out)}}
func LocalCappedRange(input []int,rows [][]int){out:=capped(input);for _,out=range rows{};out=append(out,1);slices.Sort(out)}
func LocalCappedReslice(input []int){out:=capped(input);out=out[:0];out=append(out,1);slices.Sort(out)}
func LocalCappedAddress(input []int,change func(*[]int)){out:=capped(input);change(&out);out=append(out,1);slices.Sort(out)}
func LocalCappedCapture(input []int){out:=capped(input);change:=func(){out=input};change();out=append(out,1);slices.Sort(out)}
func LocalCappedLabel(input []int,again bool){out:=capped(input);next:out=append(out,1);out=input;if again{goto next};slices.Sort(out)}
func LocalCappedReborrow(input []int){out:=capped(input);out=append(out,1);out=input;slices.Sort(out)}
func LocalCappedEmpty(input,extra []int){out:=capped(input);out=append(out,extra...);slices.Sort(out)}
func LocalCappedEffect(input []int){out:=cappedEffect(input);out=append(out,1);slices.Sort(out)}
func LocalCappedPointee(input []*int){out:=cappedReferences(input);out=append(out,nil);*out[0]=1}
func tupleFresh(flag bool)(int,[]int){if flag{return 0,nil};return 1,[]int{1}}
func tupleBorrowed(input []int)([]int,[]int){return []int{1},input}
func tupleMixed(input []int,flag bool)(int,[]int){if flag{return 0,input};return 1,[]int{1}}
func tupleEscaped()(int,[]int){out:=[]int{1};shared=out;return 0,out}
func tuplePointees(input *int)(int,[]*int){return 0,[]*int{input}}
func tupleEffect(input *int)(int,[]int){*input++;return 0,[]int{1}}
func tupleNamed(input []int)(n int,out []int){defer func(){out=input}();return 0,[]int{1}}
func tuplePointer(input *int)(*int,[]int){return input,[]int{1}}
func tupleRecord(input []int)([]int,struct{Values []int}){return []int{1},struct{Values []int}{input}}
func tupleGeneric[T any](input T)(int,[]T){return 0,[]T{input}}
func tupleMap()(int,map[int]int){return 0,map[int]int{}}
func TupleLocal(){_,out:=tupleFresh(false);slices.Sort(out)}
func TupleGeneric(){_,out:=tupleGeneric(1);slices.Sort(out)}
func TupleMap(){_,out:=tupleMap();maps.Copy(out,map[int]int{1:1})}
func TupleBorrowed(input []int){out,borrowed:=tupleBorrowed(input);slices.Sort(out);slices.Sort(borrowed)}
func TupleMixed(input []int){_,out:=tupleMixed(input,true);slices.Sort(out)}
func TupleEscaped(){_,out:=tupleEscaped();slices.Sort(out)}
func TupleNamed(input []int){_,out:=tupleNamed(input);slices.Sort(out)}
func TupleDynamic(get func()(int,[]int)){_,out:=get();slices.Sort(out)}
func TuplePointee(input *int){_,out:=tuplePointees(input);*out[0]=1}
func TupleEffect(input *int){_,out:=tupleEffect(input);slices.Sort(out)}
func TuplePointer(input *int){out,_:=tuplePointer(input);*out=1}
func TupleRecord(input []int){_,out:=tupleRecord(input);out.Values[0]=1}
func tupleForward()(int,[]int){return tupleFresh(false)}
func tupleRecursive(n int)(int,[]int){if n==0{return 0,nil};_,out:=tupleMutual(n-1);return n,append(out,n)}
func tupleMutual(n int)(int,[]int){return tupleRecursive(n)}
func tupleBadA(input []int,n int)(int,[]int){if n==0{return 0,input};return tupleBadB(input,n-1)}
func tupleBadB(input []int,n int)(int,[]int){return tupleBadA(input,n)}
func tupleAddress()(int,[]int){out:=[]int{1};leakedElement=&out[0];return 0,out}
func tupleClosure()(int,[]int){out:=[]int{1};f:=func(){shared=out};_ =f;return 0,out}
func tupleDeferred()(int,[]int){out:=[]int{1};defer retain(out);return 0,out}
func TupleMutate(){_,out:=tupleFresh(false);out[0]=2;out=append(out,3)}
func TupleForward(){_,out:=tupleForward();slices.Sort(out)}
func TupleRecursive(){_,out:=tupleMutual(2);slices.Sort(out)}
func TupleRecursiveBorrowed(input []int){_,out:=tupleBadB(input,2);slices.Sort(out)}
func TupleAddress(){_,out:=tupleAddress();slices.Sort(out)}
func TupleClosure(){_,out:=tupleClosure();slices.Sort(out)}
func TupleDeferred(){_,out:=tupleDeferred();slices.Sort(out)}
func mapEntries()map[int][]int{out:=map[int][]int{};for _,n:=range []int{1,2}{out[n]=append(out[n],n)};return out}
func MapEntries(){_ =mapEntries()}
func MapMade(){out:=make(map[int][]int);out[1]=append(out[2],1)}
func MapSpread(input []int){out:=map[int][]int{};out[1]=append(out[1],input...)}
func MapShared(out map[int][]int){out[1]=append(out[1],1)}
func MapBorrowed(input []int){out:=map[int][]int{1:input};out[1]=append(out[1],1)}
func MapInserted(input []int){out:=map[int][]int{};out[1]=input;out[1]=append(out[1],1)}
func MapCopied(input map[int][]int){out:=map[int][]int{};maps.Copy(out,input);out[1]=append(out[1],1)}
func MapAliased(input []int){out:=map[int][]int{};alias:=out;alias[1]=input;out[1]=append(out[1],1)}
func MapPublished(){out:=map[int][]int{};out[1]=append(out[1],1);shared=out[1];out[1]=append(out[1],1)}
func MapRanged(){out:=map[int][]int{};out[1]=append(out[1],1);for _,value:=range out{shared=value};out[1]=append(out[1],1)}
func MapAddressed(){out:=map[int][]int{};ptr:=&out;_ =ptr;out[1]=append(out[1],1)}
func MapCaptured(){out:=map[int][]int{};f:=func(){shared=out[1]};_ =f;out[1]=append(out[1],1)}
func MapDeferred(){out:=map[int][]int{};defer func(){shared=out[1]}();out[1]=append(out[1],1)}
func MapShadowed(input []int){out:=map[int][]int{};append:=func([]int,int)[]int{return input};out[1]=append(out[1],1)}
func MapEffect(input []int){out:=map[int][]int{};out[1]=append(out[1],appendElement(input))}
func mapReferences(input *int)map[int][]*int{out:=map[int][]*int{};out[1]=append(out[1],input);return out}
func MapPointee(input *int){out:=mapReferences(input);*out[1][0]=1}
func SpareCapacity(input []int){out:=append(input[:len(input):cap(input)],1);slices.Sort(out)}
func UnequalBounds(input []int,n,m int){out:=append(input[:n:m],1);slices.Sort(out)}
func EmptyExpansion(input,extra []int){out:=append(input[:len(input):len(input)],extra...);slices.Sort(out)}
func ChangingBound(input []int,next func([]int)int){out:=append(input[:next(input):next(input)],1);slices.Sort(out)}
func ShadowedLength(input []int){len:=func(s []int)int{return cap(s)};out:=append(input[:len(input):len(input)],1);slices.Sort(out)}
func ChannelBounds(input []int,ch chan int){out:=append(input[:len(ch):len(ch)],1);slices.Sort(out)}
func CapturedBounds(input []int,n int){next:=func()int{n++;return 0};out:=append(input[next():n:n],1);slices.Sort(out)}
func AddressedBounds(input []int,n *int){bound:=*n;alias:=&bound;out:=append(input[func()int{*alias++;return 0}():bound:bound],1);slices.Sort(out)}
func ForcedPointee(input []*int){out:=append(input[:len(input):len(input)],nil);*out[0]=1}
func appendElement(input []int)int{input[0]++;return 1}
func ForcedArgumentEffect(input []int){out:=append(input[:len(input):len(input)],appendElement(input));slices.Sort(out)}
func SelfAppend(){values:=[]int{2,1}; values=append(values,3); slices.Sort(values)}
func CloneSort(values []int){local:=slices.Clone(values); slices.Sort(local)}
func CloneCopy(dst,src map[int]int){local:=maps.Clone(dst); maps.Copy(local,src)}
func Shallow(values []*int){local:=slices.Clone(values); *local[0]=1}
func Clone(values []int)[]int{return values}
func FakeClone(values []int){local:=Clone(values); slices.Sort(local)}
func ExternalSort(values []int){slices.Sort(values)}
func ExternalCopy(dst,src map[int]int){maps.Copy(dst,src)}
func GenericSort[S ~[]int](values S){slices.Sort(values)}
func GlobalSort(){slices.Sort(shared)}
func ExternalAlias(values []int){alias:=values[1:]; slices.Sort(alias)}
func ExternalAppend(values []int){alias:=append(values,3); slices.Sort(alias)}
func Mixed(values []int, flag bool){alias:=[]int{2,1}; if flag{alias=values}; slices.Sort(alias)}
func Unknown(factory func()[]int){values:=factory(); slices.Sort(values)}
func Callback(compare func(int,int)int){values:=[]int{2,1}; slices.SortFunc(values,compare)}
func InlineSort(){v:=[]int{2,1};slices.SortFunc(v,func(a,b int)int{return a-b})}
func InlineStable(){v:=[]int{2,1};slices.SortStableFunc(v,func(a,b int)int{return a-b})}
func InlineReflect(){v:=[]int{2,1};sort.Slice(v,func(a,b int)bool{return v[a]<v[b]})}
func InlineReflectStable(){v:=[]int{2,1};sort.SliceStable(v,func(a,b int)bool{return v[a]<v[b]})}
func InlineShared(v []int){slices.SortFunc(v,func(a,b int)int{return a-b})}
func NamedCallback(){v:=[]int{2,1};slices.SortFunc(v,compare)}
func compare(a,b int)int{return a-b}
func NamedStandard(){v:=[]int{2,1};slices.SortFunc(v,cmp.Compare[int])}
func NamedStable(){v:=[]int{2,1};slices.SortStableFunc(v,compare)}
func compareNoisy(a,b int)int{counter++;return compare(a,b)}
func NamedNoisy(){v:=[]int{2,1};slices.SortFunc(v,compareNoisy)}
func comparePointers(a,b *int)int{*a=1;return 0}
func NamedPointee(a,b *int){v:=[]*int{a,b};slices.SortFunc(v,comparePointers)}
func NamedShared(v []int){slices.SortFunc(v,compare)}
type comparer struct{}
func(comparer)Compare(a,b int)int{return a-b}
func MethodCallback(){v:=[]int{2,1};slices.SortFunc(v,(comparer{}).Compare)}
func VariableCallback(){v:=[]int{2,1};compare:=func(a,b int)int{return a-b};slices.SortFunc(v,compare)}
func InlineOutput(){v:=[]int{2,1};slices.SortFunc(v,func(a,b int)int{println(a);return a-b})}
func InlineCaptured(shared *int){v:=[]int{2,1};slices.SortFunc(v,func(a,b int)int{*shared++;return a-b})}
func InlineGlobal(){v:=[]int{2,1};slices.SortFunc(v,func(a,b int)int{return a-b+counter})}
func InlineLaunch(){v:=[]int{2,1};slices.SortFunc(v,func(a,b int)int{go func(){}();return a-b})}
func InlineChannel(ch chan int){v:=[]int{2,1};slices.SortFunc(v,func(a,b int)int{ch<-a;return a-b})}
func InlinePolicy(){v:=[]string{"b","a"};slices.SortFunc(v,func(a,b string)int{return strings.Compare(a,b)})}
func helper(){values:=[]int{2,1}; slices.Sort(values)}
func Caller(){helper()}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ReturnedRelativeSelf", "ReturnedRelativeNil", "ReturnedRelativeLocal", "ReturnedRecursive", "ReturnedFinalAppend", "ReturnedDirectMake", "ReturnedDirectClone", "ReturnedSorted", "ReturnedCopied", "ReturnedReferences", "SelfView", "SelfViewAppend", "ReturnedQueue", "ReturnedFilled", "ReturnedMap", "MapEntries", "MapMade", "MapSpread", "NamedCallback", "NamedStandard", "NamedStable", "InlineSort", "InlineStable", "InlineReflect", "InlineReflectStable", "InlinePolicy", "FreshContainers", "FreshBytes", "FreshRunes", "NamedString", "NilClone", "NilSelf", "NilSort", "NilDirect", "LocalSort", "NamedSort", "LocalCopy", "LocalReverse", "LocalCompact", "LocalDelete", "LocalInsert", "LocalReplace", "LocalInts", "LocalFloats", "LocalStrings", "LocalAlias", "LocalAppend", "SelfAppend", "CloneSort", "CloneCopy", "Caller"} {
		t.Run(name, func(t *testing.T) {
			r := resultNamed(t, report, name)
			if len(r.Effects) != 0 || !r.Complete {
				t.Fatalf("local container effects: %+v", r)
			}
		})
	}
	for _, name := range []string{"TupleMutate", "TupleLocal", "TupleGeneric", "TupleMap", "TupleForward", "ForcedAppend", "ForcedBound", "ForcedConstant", "ReturnedForced", "ReturnedCapped", "ReturnedCappedBranches", "ReturnedCappedGeneric", "ReturnedCappedAppend", "LocalCapped", "LocalCappedView", "LocalCappedAlias", "LocalCappedReset", "LocalCappedTwice"} {
		if r := resultNamed(t, report, name); len(r.Effects) != 0 || !r.Complete {
			t.Fatalf("%s loses fresh append ownership: %+v", name, r)
		}
	}
	for _, name := range []string{"MapShared", "MapBorrowed", "MapInserted", "MapCopied", "MapAliased", "MapPublished", "MapRanged", "MapAddressed", "MapCaptured", "MapDeferred", "MapShadowed", "MapEffect", "MapPointee", "TupleBorrowed", "TupleMixed", "TupleEscaped", "TupleNamed", "TupleDynamic", "TuplePointee", "TupleEffect", "TuplePointer", "TupleRecord", "TupleRecursiveBorrowed", "TupleAddress", "TupleClosure", "TupleDeferred", "SpareCapacity", "UnequalBounds", "EmptyExpansion", "ChangingBound", "ShadowedLength", "ChannelBounds", "CapturedBounds", "AddressedBounds", "ForcedPointee", "ForcedArgumentEffect", "ReturnedCappedDirect", "ReturnedCappedMixed", "ReturnedCappedNamed", "ReturnedCappedDynamic", "ReturnedCappedEmpty", "ReturnedCappedEffect", "ReturnedCappedPointee", "LocalCappedBefore", "LocalCappedMixed", "LocalCappedLoop", "LocalCappedRange", "LocalCappedReslice", "LocalCappedAddress", "LocalCappedCapture", "LocalCappedLabel", "LocalCappedReborrow", "LocalCappedEmpty", "LocalCappedEffect", "LocalCappedPointee"} {
		if len(resultNamed(t, report, name).Effects) == 0 {
			t.Fatalf("%s loses borrowed storage or argument effects", name)
		}
	}
	if f := findFunction(t, facts, "TupleBorrowed"); f.Calls[1].MutationTarget != quality.Local {
		t.Fatalf("first tuple result loses fresh storage: %+v", f.Calls)
	}
	for _, name := range []string{"TupleRecursive", "TupleEffect"} {
		f := findFunction(t, facts, name)
		if f.Calls[len(f.Calls)-1].MutationTarget != quality.Local {
			t.Fatalf("%s loses owned tuple storage: %+v", name, f.Calls)
		}
	}
	for _, name := range []string{"TupleBorrowed", "TupleMixed", "TupleEscaped", "TupleNamed", "TupleDynamic", "TupleRecursiveBorrowed", "TupleAddress", "TupleClosure", "TupleDeferred"} {
		f := findFunction(t, facts, name)
		if f.Calls[len(f.Calls)-1].MutationTarget == quality.Local {
			t.Fatalf("%s hides borrowed or escaped tuple storage: %+v", name, f.Calls)
		}
	}
	for _, name := range []string{"MapShared", "MapBorrowed", "MapInserted", "MapCopied", "MapAliased", "MapPublished", "MapRanged", "MapAddressed", "MapCaptured", "MapDeferred"} {
		f := findFunction(t, facts, name)
		last := f.Mutations[len(f.Mutations)-1]
		if last.Provenance == quality.Local {
			t.Fatalf("%s hides borrowed or published entry storage: %+v", name, f.Mutations)
		}
	}
	before := findFunction(t, facts, "LocalCappedBefore")
	if before.Mutations[0].Provenance == quality.Local || before.Mutations[len(before.Mutations)-1].Provenance != quality.Local {
		t.Fatalf("append loses the borrowed-to-fresh storage boundary: %+v", before.Mutations)
	}
	for _, name := range []string{"LocalCappedMixed", "LocalCappedLoop", "LocalCappedRange", "LocalCappedReslice", "LocalCappedAddress", "LocalCappedCapture", "LocalCappedLabel", "LocalCappedReborrow", "LocalCappedEmpty"} {
		f := findFunction(t, facts, name)
		if f.Calls[len(f.Calls)-1].MutationTarget == quality.Local {
			t.Fatalf("%s loses a header write: %+v", name, f.Calls)
		}
	}
	for _, name := range []string{"ChannelBounds", "CapturedBounds", "AddressedBounds"} {
		f := findFunction(t, facts, name)
		if f.Calls[len(f.Calls)-1].MutationTarget == quality.Local {
			t.Fatalf("%s incorrectly proves stable capacity", name)
		}
	}
	if result := resultNamed(t, report, "ViewBounds"); len(result.Effects) == 0 {
		t.Fatal("self slice lost bound evaluation effect")
	}
	for _, tc := range []struct {
		name       string
		provenance quality.Provenance
	}{
		{"ReturnedRelativeSelfBorrowed", quality.External}, {"ReturnedRelativeRecursive", quality.Unknown}, {"ReturnedRelativeBorrowed", quality.Unknown}, {"ReturnedRelativeMixed", quality.Unknown}, {"ReturnedRelativePublished", quality.Unknown}, {"ReturnedRelativeAliased", quality.Unknown}, {"ReturnedRelativeAddress", quality.Unknown}, {"ReturnedRelativeNamed", quality.Unknown}, {"ReturnedRecursiveBorrowed", quality.Unknown}, {"ReturnedRecursiveEscaped", quality.Unknown}, {"ReturnedOperationAlias", quality.Unknown}, {"ReturnedInserted", quality.Unknown}, {"ReturnedGlobal", quality.Unknown}, {"ReturnedFormal", quality.Unknown}, {"ReturnedView", quality.Unknown}, {"ReturnedNamed", quality.Unknown}, {"ReturnedFakeDecode", quality.Unknown}, {"SharedView", quality.External}, {"MixedView", quality.Unknown}, {"UnknownView", quality.Unknown}, {"ReturnedMixed", quality.Unknown}, {"ReturnedPublished", quality.Unknown}, {"ReturnedWritten", quality.Unknown}, {"ReturnedEscape", quality.Unknown}, {"ReturnedAddress", quality.Unknown}, {"ReturnedClosure", quality.Unknown}, {"ReturnedDeferred", quality.Unknown}, {"InlineShared", quality.External}, {"CallbackFields", quality.Unknown}, {"ShadowNil", quality.External}, {"OrdinaryConversion", quality.External}, {"ExternalSort", quality.External}, {"ExternalCopy", quality.External}, {"GenericSort", quality.External}, {"GlobalSort", quality.External}, {"ExternalAlias", quality.External}, {"ExternalAppend", quality.Unknown}, {"FakeClone", quality.Unknown}, {"Mixed", quality.Unknown}, {"Unknown", quality.Unknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := findFunction(t, facts, tc.name)
			call := f.Calls[len(f.Calls)-1]
			if call.MutationTarget != tc.provenance {
				t.Fatalf("target: %+v, want %s", call, tc.provenance)
			}
			r := resultNamed(t, report, tc.name)
			found := false
			for _, e := range r.Effects {
				found = found || e.Kind == quality.MutationEffect
			}
			if !found {
				t.Fatalf("observable or unknown target lost mutation: %+v", r)
			}
		})
	}
	decoded := resultNamed(t, report, "ReturnedDecoded")
	foundPanic := false
	for _, effect := range decoded.Effects {
		if effect.Kind == quality.MutationEffect || effect.Kind == quality.UnknownEffect {
			t.Fatalf("decoded storage loses ownership: %+v", decoded)
		}
		foundPanic = foundPanic || effect.Kind == quality.Panic
	}
	if !foundPanic {
		t.Fatal("decoded factory lost panic evidence")
	}
	for _, name := range []string{"ReturnedRelativeEffect", "ReturnedDecoded", "ReturnedOutput", "ReturnedCalled"} {
		fact := findFunction(t, facts, name)
		if got := fact.Calls[len(fact.Calls)-1].MutationTarget; got != quality.Local {
			t.Fatalf("%s backing storage=%s, want local", name, got)
		}
	}
	for _, name := range []string{"ReturnedRelativePointee", "ReturnedRecursivePointee", "ReturnedPointee", "Shallow", "ShallowNilClone", "NilFunction", "BorrowedBytes", "BorrowedConcat"} {
		if len(resultNamed(t, report, name).Effects) == 0 {
			t.Fatalf("%s loses pointee or dynamic-call evidence", name)
		}
	}
	for _, name := range []string{"Callback", "MethodCallback", "VariableCallback"} {
		callback := findFunction(t, facts, name)
		if callback.Calls[0].MutationTarget != "" || callback.Calls[0].CallbackExpanded || len(resultNamed(t, report, name).Effects) == 0 {
			t.Fatalf("unproved callback %s became local: %+v", name, callback)
		}
	}
	for _, tc := range []struct {
		name string
		kind quality.Kind
	}{
		{"NamedNoisy", quality.Global}, {"NamedPointee", quality.UnknownEffect}, {"NamedShared", quality.MutationEffect},
		{"ReturnedRelativeEffect", quality.Global}, {"ReturnedCalled", quality.Global}, {"ReturnedOutput", quality.IO}, {"InlineOutput", quality.IO}, {"InlineCaptured", quality.MutationEffect}, {"InlineGlobal", quality.Global}, {"InlineLaunch", quality.UnknownEffect}, {"InlineChannel", quality.UnknownEffect},
	} {
		got := resultNamed(t, report, tc.name)
		found := false
		for _, effect := range got.Effects {
			found = found || effect.Kind == tc.kind
		}
		if !found {
			t.Fatalf("%s loses %s callback evidence: %+v", tc.name, tc.kind, got)
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
	roundtrip, err := quality.Evaluate(restored, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(report, roundtrip) {
		t.Fatal("saved ownership facts changed evaluation")
	}
	policy := quality.DefaultConfig()
	policy.Prefixes = append(policy.Prefixes, quality.Prefix{Path: "example.com/quality-repro.compare", Kind: quality.IO})
	callbackOverride, err := quality.Evaluate(restored, policy)
	if err != nil {
		t.Fatal(err)
	}
	if got := resultNamed(t, callbackOverride, "NamedCallback"); len(got.Effects) != 1 || got.Effects[0].Kind != quality.IO {
		t.Fatalf("named callback hides configured effect: %+v", got)
	}
	for _, path := range []string{"strings.Compare", "slices.SortFunc"} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: path, Kind: quality.IO})
		replay, err := quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		got := resultNamed(t, replay, "InlinePolicy")
		if len(got.Effects) != 1 || got.Effects[0].Kind != quality.IO {
			t.Fatalf("saved callback hid %s override: %+v", path, got)
		}
	}
	override := quality.DefaultConfig()
	override.Prefixes = append(override.Prefixes, quality.Prefix{Path: "slices.Sort", Kind: quality.IO})
	changed, err := quality.Evaluate(restored, override)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"LocalSort", "Caller"} {
		r := resultNamed(t, changed, name)
		if len(r.Effects) != 1 || r.Effects[0].Kind != quality.IO {
			t.Fatalf("policy override lost for %s: %+v", name, r)
		}
	}
}

func TestContainerHelpersAcrossPackages(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import "example.com/quality-repro/lib"
func Caller(values []int)[]int{return lib.Sorted(values)}
func Shared(values []int){lib.Shared(values)}
func InlineHelper(values []int)[]int{return lib.SortedInline(values)}
`))
	if err := os.MkdirAll(filepath.Join(root, "lib"), 0700); err != nil {
		t.Fatal(err)
	}
	source := `package lib
import "slices"
func Sorted(values []int)[]int{local:=slices.Clone(values); slices.Sort(local); return local}
func Shared(values []int){slices.Sort(values)}
func SortedInline(values []int)[]int{local:=slices.Clone(values);slices.SortFunc(local,func(a,b int)int{return a-b});return local}
`
	if err := os.WriteFile(filepath.Join(root, "lib/helper.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	caller := resultNamed(t, report, "Caller")
	if len(caller.Effects) != 0 || !caller.Complete {
		t.Fatalf("fresh helper reported effects: %+v", caller)
	}
	shared := resultNamed(t, report, "Shared")
	if len(shared.Effects) != 1 || shared.Effects[0].Kind != quality.MutationEffect || !shared.Complete {
		t.Fatalf("external helper lost observable write: %+v", shared)
	}
	encoded, err := json.Marshal(facts)
	if err != nil {
		t.Fatal(err)
	}
	var restored []quality.Function
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	inline := resultNamed(t, report, "InlineHelper")
	if len(inline.Effects) != 0 || !inline.Complete {
		t.Fatalf("inline helper stays effectful: %+v", inline)
	}
	callbackConfig := quality.DefaultConfig()
	callbackConfig.Prefixes = append(callbackConfig.Prefixes, quality.Prefix{Path: "slices.SortFunc", Kind: quality.IO})
	changed, err := quality.Evaluate(restored, callbackConfig)
	if err != nil {
		t.Fatal(err)
	}
	inline = resultNamed(t, changed, "InlineHelper")
	if len(inline.Effects) != 1 || inline.Effects[0].Kind != quality.IO {
		t.Fatalf("saved helper callback proof hid override: %+v", inline)
	}
	for _, callee := range []string{"slices.Sort", "slices.Clone"} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: callee, Kind: quality.IO})
		changed, err := quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		caller := resultNamed(t, changed, "Caller")
		if len(caller.Effects) != 1 || caller.Effects[0].Kind != quality.IO {
			t.Fatalf("saved helper hid %s override: %+v", callee, caller)
		}
	}
	for i := range restored {
		for j := range restored[i].Calls {
			restored[i].Calls[j].MutationTarget = ""
		}
		for j := range restored[i].Helpers {
			for k := range restored[i].Helpers[j].Calls {
				restored[i].Helpers[j].Calls[k].MutationTarget = ""
			}
		}
	}
	legacy, err := quality.Evaluate(restored, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if len(resultNamed(t, legacy, "Caller").Effects) == 0 {
		t.Fatal("missing ownership evidence silently treated as local")
	}
}
