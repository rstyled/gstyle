package pdg

// SourceParameter stores typed binding uses as ranges in the original file.
// TypeSyntax preserves the producer's spelling; Type records resolved meaning.
type SourceParameter struct {
	Name, Type, TypeSyntax      string
	Uses                        []ByteRange
	Addressed                   bool
	Captured, FlowKnown, Stable bool
}

// SourceFunction connects semantic identity to syntax needed for edits.
type SourceFunction struct {
	ID, Name, Package                  string
	Source                             Ref
	Declaration, NameRange, Parameters ByteRange
	Body                               ByteRange
	ParametersList                     []SourceParameter
	Bindings                           []string
	Method, Generic, Variadic          bool
}

// SourceReference resolves a use to a function identity, not its spelling.
// A non-direct use blocks a signature rewrite until a planner supports it.
type SourceReference struct {
	Receiver         SourceReceiver
	Target           string
	Source           Ref
	Range, Callee    ByteRange
	Arguments        []ByteRange
	Direct, Variadic bool
}

// SourceReceiver proves when a function argument already has the receiver
// type. Copy refers to the binding in a complete, unchanged struct projection.
// False ExactType keeps the formal conversion, including for untyped nil.
type SourceReceiver struct {
	Copy                    ByteRange
	ExactType, Parenthesize bool
}
