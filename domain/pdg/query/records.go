package query

type Text uint32
type Ref uint32

type Status uint8

const (
	Unknown Status = iota
	Known
	Unsupported
	NotApplicable
	Supported
	Approximated
)

type Capability struct {
	Description Text
	Status      Status
}

type BooleanFact struct {
	Reason Text
	Status Status
	Value  bool
}
