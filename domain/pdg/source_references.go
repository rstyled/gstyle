package pdg

// SourceParameter stores typed binding uses as ranges in the original file.
// TypeSyntax preserves the producer's spelling; Type records resolved meaning.
type SourceParameter struct {
	Name, Type, TypeSyntax string
	Uses                   []ByteRange
	Addressed              bool
}

// SourceFunction connects semantic identity to syntax needed for edits.
type SourceFunction struct {
	ID, Name, Package                  string
	Source                             Ref
	Declaration, NameRange, Parameters ByteRange
	ParametersList                     []SourceParameter
	Bindings                           []string
	Method, Generic, Variadic          bool
}

// SourceReference resolves a use to a function identity, not its spelling.
// A non-direct use blocks a signature rewrite until a planner supports it.
type SourceReference struct {
	Target           string
	Source           Ref
	Range, Callee    ByteRange
	Arguments        []ByteRange
	Direct, Variadic bool
}
