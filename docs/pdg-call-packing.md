# Canonical receiving values for tuple and variadic calls

The typed adapter records `gstyle.go:call-shape` on an invocation. Its four
required fields are `fixed`, `variadic`, `spread` and `tuple`. Fixed counts the
non-variadic formal slots; tuple is zero or the expanded result count of the
sole expression producer. `gstyle.go:tuple-arity` independently records that
producer's tuple size. Canonical operand edges still identify the evaluated
expressions. These facts do not add a source dependency engine.

The shared binding query checks the formal interface, contiguous expression
slots, complete operand inventory, tuple kind/arity, receiving count and argument
assignment representations. Variadic packing produces a nil slice for zero
remaining arguments and a non-nil slice for one or more. A nil element remains
inside a non-nil packed slice. Spread assignment preserves the original slice's
nilness, including a non-nil slice of length zero. Deferred registration saves
these assigned values before later binding changes.

Tuple expansion keeps separate representation slots. Boxing a concrete result
proves a non-nil interface even when its pointer payload is unknown. An interface
result assigned without boxing stays unknown; one result's value cannot stand
for its siblings. General call-result values remain required work. Evaluation
and faults of the sole tuple-producing call still precede the receiving call.

Records reject duplicate/unknown/missing fields, wrong JSON types, invalid
numbers, duplicate records and impossible spread/tuple combinations. Receiving
shape must agree with a saved defer's independent spread/tuple capture contract.
Missing or inconsistent evidence cannot establish precise arguments. Legacy
records can still describe fixed-arity scalar inputs; an unexpanded tuple cannot
be admitted as one such scalar argument.

## Conditional non-nil values

The canonical value record now also accepts `non_nil: true`. Slice/map literals,
closure/function values, ordinary addresses, and builtin constructors provide
whole-value non-nil facts when they produce a value. Address cancellation `&*p`
does not supply this fact. Successful value production does not assert that
operand evaluation or allocation/call execution completes without a fault.

Non-nilness is a partial value property. Dependency search still follows payload
inputs, so a slice containing the checked error remains a use of that error.
Nil-error checks also keep their separate empty-container property: a slice can
be both non-nil and empty. This preserves the existing empty-success rule without
turning empty slices into nil values.

Source cases cover packing, spread, tuple slot types, saved inputs, recursive
pack states, fault ordering and resource cleanup. A separate arity model checks
93 receiving layouts, while native Go checks empty/nonempty packs, nil elements,
nil/non-nil empty spread slices and tuple expansion. Metadata deletion/conflict
cases retain uncertainty or reject inconsistent records.

Receiver wrappers, general result payloads, callback bodies, heap/result writes,
complete exceptional graph edges and concurrent execution remain required work.
These checks do not establish full Go conformance or complete the PDG objective.
