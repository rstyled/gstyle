# Argument contexts for canonical exit queries

Local exit summaries now distinguish unknown, nil, non-nil, false and true
arguments for free-function bodies. Tuple and variadic receiving contracts are
now described in `pdg-call-packing.md`. The caller evaluates canonical
operand slots in its current immutable value state. Compiler argument-assignment
representation facts preserve interface boxing: a nil pointer assigned to an
interface is non-nil. Missing inventories, operands or representation facts do
not produce a known argument. No AST binding or dependency walk supplies them.

Each callee context seeds its formal symbols before body evaluation. Writes still
kill or replace those facts through shared binding transfers. Ordinary and
eligible deferred-panic invocations retain separate outcomes. An ordinary helper
cannot recover its caller's panic merely because it receives the same flag.
Deferred registration saves the current argument context; later stores do not
change that invocation's saved value.

Contexts are discovered on demand, without a bound on recursion or a truncation
count. Finite outcomes grow before possible divergence descends. If divergence
analysis discovers another context, finite solving and divergence seeding repeat
before the table is sealed. The argument domain is finite; exact numeric/string
values, arbitrary heap state and callback identity are not in this domain.

Constant-argument sites also prepare contexts independently of caller completion.
Resource queries can start after an acquisition whose whole-call exit is opaque.
Preparing a callee summary does not assert that its call site executes. Resource
consumers use these same summaries to distinguish a false abort flag from a true
one, and terminating argument-state recursion from a real recursive cycle.

Completed tables are read-only. A later query for a context absent from the
solved graph set falls back to the general may-exit summary. Receiver wrappers,
general result values, captured/heap effects and
callback bodies retain their existing conservative boundaries. General callback,
exceptional storage and concurrent execution remain required work.

Source regressions cover parameters, copies, overwrites, nil pointers, boxed nil
pointers, nil interfaces, multiple slots, stored declaration targets, recursion,
recovery and saved deferred inputs. Deleting argument representations, inventories
or operands loses precision. Indirect writes retain uncertainty instead of stale
argument values. A separate concrete-state walk checks 1,056 bounded observations
across one/two-function Boolean recursion tables. Native Go checks terminating
argument recursion and LIFO saved arguments. These checks do not prove full Go
conformance, arbitrary termination or completion of the full PDG migration.
