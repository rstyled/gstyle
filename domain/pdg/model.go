// Package pdg defines the compact native representation of compat's PDG IR
// draft 0.1.0. IDs are table indexes, never matching labels. Zero is absent.
package pdg

const SchemaVersion = "0.1.0"

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

// Fact represents the schema's string fact; unknown values always have reasons.
type Fact struct {
	Value  Text
	Status Status
}

// BooleanFact occupies the same space as Fact and distinguishes known false
// from unavailable evidence without interning a boolean spelling.
type BooleanFact struct {
	Reason Text
	Status Status
	Value  bool
}

type Profile struct{ ID, Version Text }
type Capability struct {
	Description Text
	Status      Status
}

type Span struct{ Source, Start, End uint32 }

type Origin uint8

const (
	SourceOrigin Origin = iota
	Desugaring
	MacroExpansion
	Analysis
	Projection
)

type Provenance struct {
	Description Text
	Nodes       []Text
	Spans       []Span
	Origin      Origin
}

// Extension.Value is interned JSON text. Unknown extension data is preserved.
type Extension struct{ Name, Value Text }

type Source struct {
	Snapshot        Ref `json:"-"` // Shared source context index; omitted from interchange JSON.
	ID, Path        Text
	ContentIdentity Fact
}

type Producer struct {
	Name, Version, Language Text
	LanguageVersion         Fact
	Configuration           Text // JSON object; stored once, not copied onto graph items.
}

type Role uint8

const (
	ParameterRole Role = iota
	LocalRole
	GlobalRole
	CaptureRole
	ReceiverRole
	OutputRole
	OtherRole
	ReturnRole
)

type Parameter struct {
	Name, Type Fact
	Variadic   BooleanFact
	Symbol     Ref
	Position   uint32
	Role       Role
	Extensions []Extension
}

type CountFact struct {
	Value  uint64
	Reason Text
	Status Status
}

type Function struct {
	ID                  Text
	Name, QualifiedName Fact
	Span                Span
	Inputs, Outputs     []Parameter
	Interface           Ref
	ReferenceCount      CountFact
}

type Symbol struct {
	ID, Scope  Text
	Name, Type Fact
	Role       Role
	Extensions []Extension
}

type Definition struct {
	ID                     Text
	Node, Symbol, Location Ref
}

type Location struct {
	ID, Abstraction, Description Text
	Evidence                     Ref
	Extensions                   []Extension
}

type Category uint8

const (
	Declaration Category = iota
	Read
	Assignment
	Computation
	Control
	Call
	Entry
	FormalInput
	Return
	FormalOutput
	Exit
	Merge
	MemorySummary
	Other
)

// Position encodes an optional zero-based position as position+1; zero is absent.
type Position uint32

type CallForm uint8

const (
	UnrecordedCall CallForm = iota
	DirectCall
	IndirectCall
	UnknownCall
)

type ExitKind uint8

const (
	UnrecordedExit ExitKind = iota
	NormalExit
	ExceptionalExit
	OtherExit
)

type Access struct {
	Symbol, Location Ref
	UnresolvedReason Text
	Position         Position
}

// Attributes are sparse: Node.Attributes is zero when the attribute object is
// empty. Slices live only here, not in every node's hot record.
type Attributes struct {
	Symbols, Defines                  []Ref
	Reads, Writes                     []Access
	Operator, Literal, Callee         Fact
	Construct, WriteForm, Description Text
	Outcomes                          []Text
	Position                          Position
	CallForm                          CallForm
	ExitKind                          ExitKind
	Extensions                        []Extension
}

// MatchLabel and Line belong to the gstyle matching profile, not base IR identity.
type Node struct {
	MatchLabel                        Ref
	Line                              uint32
	ID                                Text
	OriginalKind                      Fact
	Span                              Ref
	Attributes                        Ref
	Provenance, Evidence, Annotations Ref
	Category                          Category
	Synthetic                         bool
}

type EdgeKind uint8

const (
	Data EdgeKind = iota
	ControlEdge
	Execution
)

type Edge struct {
	ID, Subkind                       Text
	Source, Target                    Ref
	Evidence, Provenance, Annotations Ref
	Symbol, Definition, Location      Ref
	Outcome                           Text
	Position                          Position
	Kind                              EdgeKind
	LoopCarried                       uint8 // 0 unrecorded, 1 false, 2 true.
}

type Severity uint8

const (
	Info Severity = iota
	Warning
	Error
)

type Diagnostic struct {
	Code, Message, Capability Text
	Span, Node                Ref
	Severity                  Severity
}

type ConformanceStatus uint8

const (
	NotEvaluated ConformanceStatus = iota
	Conformant
	NonConformant
)

type Conformance struct {
	Reasons []Text
	Status  ConformanceStatus
}

type NamedCapability struct {
	Name     Text
	Evidence Ref
}

// Graph has all draft fields without per-item string maps or repeated metadata.
// Tables can be shared by all functions in an extraction; graph-local references
// address the corresponding graph slice. References are one-based.
type Graph struct {
	Tables          *Tables
	NormalFlow      *NormalFlow
	ValueFlow       *NormalFlow // Shared typed operation flow; no AST or source copies.
	Profile         Ref
	Producer        Producer
	Sources         []Source
	Function        Function
	Symbols         []Symbol
	Definitions     []Definition
	Locations       []Location
	Nodes           []Node
	Edges           []Edge
	Spans           []Span
	Attributes      []AttributeSet
	AttributeTables AttributeTables
	Annotations     [][]Extension
	Capabilities    []NamedCapability
	Diagnostics     []Diagnostic
	Conformance     Conformance
	Extensions      []Extension
}

// MatchLabel preserves the versioned gstyle abstraction labels in a shared table.
type MatchLabel struct {
	Kind, TypeClass, SignatureClass, Callee, LiteralKind, Detail Text
	Effects                                                      uint16
}
