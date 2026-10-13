// Original gstyle synthetic cases. A non-nil dynamic type makes an interface
// non-nil, even when its contained pointer, slice, or map is nil.
package interfacenilfixture

type Parcel struct{ Code int }
type Trouble struct{}

func (*Trouble) Error() string { return "delivery stopped" }

func parcelBox(parcel *Parcel) any { return parcel }
func relayBox(parcel *Parcel) any  { return parcelBox(parcel) }
func parcelValue() any             { return Parcel{Code: 81} }
func emptyTrouble() error          { return (*Trouble)(nil) }
func optionalBox(enabled bool) any {
	if enabled {
		return Parcel{Code: 82}
	}
	return nil
}
func variedBox(enabled bool) any {
	if enabled {
		return Parcel{Code: 83}
	}
	return "deferred"
}
func recursiveBox() any         { return recursiveBox() }
func (parcel *Parcel) box() any { return parcel }
func namedBox() (result any) {
	result = Parcel{Code: 84}
	return
}
func genericBox[T any](value T) any { return value }

func localPointer() {
	var parcel *Parcel
	var boxed any = parcel
	_ = boxed == nil // want-nonnil: a nil pointer still has a dynamic type
}
func concreteRecord() {
	var boxed any = Parcel{Code: 85}
	_ = boxed != nil // want-nonnil: the interface contains a concrete record
}
func concreteNumber() {
	var boxed any = 0
	_ = nil == boxed // want-nonnil: numeric zero is not an empty interface
}
func typedReturn(parcel *Parcel) {
	_ = parcelBox(parcel) == nil // want-nonnil: every return boxes a pointer
}
func forwardedReturn(parcel *Parcel) {
	_ = relayBox(parcel) != nil // want-nonnil: a direct forwarding call preserves the proof
}
func concreteReturn() {
	_ = parcelValue() == nil // want-nonnil: a concrete record is returned through any
}
func errorReturn() {
	_ = emptyTrouble() == nil // want-nonnil: the error has a concrete pointer type
}
func branchReturns(enabled bool) {
	_ = variedBox(enabled) == nil // want-nonnil: both return paths supply concrete values
}
func assignedCall(parcel *Parcel) {
	boxed := relayBox(parcel)
	alias := boxed
	_ = alias == nil // want-nonnil: the call result reaches the alias read
}
func overwrittenBox() {
	var boxed any = nil
	boxed = Parcel{Code: 86}
	_ = boxed == nil // want-nonnil: the later definition kills the empty interface
}
func branchAssignments(enabled bool) {
	var boxed any
	if enabled {
		boxed = Parcel{Code: 87}
	} else {
		boxed = "queued"
	}
	_ = boxed == nil // want-nonnil: every reaching definition is non-nil
}
func namedReturn() {
	_ = namedBox() == nil // want-nonnil: the named result reads its assigned value
}
func methodReturn(parcel *Parcel) {
	_ = parcel.box() == nil // want-nonnil: this statically resolved method boxes its receiver
}
func boxedSlice() {
	var parcels []Parcel
	var boxed any = parcels
	_ = boxed == nil // want-nonnil: a nil slice is boxed into an interface
}
func boxedMap() {
	var parcels map[string]Parcel
	var boxed any = parcels
	_ = boxed == nil // want-nonnil: a nil map is boxed into an interface
}

// Valid or unresolved checks must remain unflagged.
func pointerGuard(parcel *Parcel)      { _ = parcel == nil }
func emptyInterface()                  { var boxed any; _ = boxed == nil }
func mixedReturn(enabled bool)         { _ = optionalBox(enabled) == nil }
func parameterGuard(boxed any)         { _ = boxed == nil }
func dynamicReturn(factory func() any) { _ = factory() == nil }
func recursiveReturn()                 { _ = recursiveBox() == nil }
func clearedBox() {
	var boxed any = Parcel{Code: 88}
	boxed = nil
	_ = boxed == nil
}
func mixedAssignment(enabled bool) {
	var boxed any
	if enabled {
		boxed = Parcel{Code: 89}
	}
	_ = boxed == nil
}
func genericReturn[T any](value T) { _ = genericBox(value) == nil }
func replaceBox(boxed *any)        { *boxed = nil }
func escapedBox() {
	var boxed any = Parcel{Code: 90}
	replaceBox(&boxed)
	_ = boxed == nil
}
func laterDefinition() {
	var boxed any
	_ = boxed == nil
	boxed = Parcel{Code: 91}
	_ = boxed
}

func shadowedNil() {
	nil := Parcel{Code: 92}
	var boxed any = Parcel{Code: 93}
	_ = boxed == nil
}
func parenthesizedNil() {
	var boxed any = Parcel{Code: 94}
	_ = boxed == (nil) // want-nonnil: parentheses do not change the nil constant
}

type Inbox struct{ Value any }

func fieldGuard(inbox *Inbox)       { _ = inbox.Value == nil }
func mapGuard(inbox map[string]any) { _ = inbox["pending"] == nil }
func sliceGuard(inbox []any)        { _ = inbox[0] == nil }
func dereferenceGuard(inbox *any)   { _ = *inbox == nil }
