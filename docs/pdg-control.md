# Control dependence

Canonical PDG control edges are semantic dependencies. Lexical containment is
not emitted as a dependency, exported in the PDG, or supplied to matching.
Forward-dominator execution edges remain a separate relation.

## Analyses

The existing scalar-normal profile retains its acyclic operation lowering and
post-dominator-tree analysis. The independent finite-path oracle checks all
33,867 forward-edge DAGs through six vertices, including disconnected vertices.
This change does not broaden that profile's conformance or data-flow claims.

General extraction uses a separate divergence-sensitive analysis. For each real
CFG vertex z, it starts at {z} and adds a predecessor only when all of its
nonempty successor set must reach z. The least fixed point describes vertices
whose every maximal finite or infinite path encounters z. Cycles that avoid z
remain outside it. Mutual post-dominance in unavoidable cycles is permitted.
No synthetic executable exit or immediate-postdominator chain is used.

An outcome x -> y controls operations in post(y), excluding strict
post-dominators of x. Justified self-control edges are retained. The CFG splits
at each operation; basic-block membership does not imply self-control for all
operations in a cyclic block. Edge outcomes retain true/false labels and
one-based internal successor positions (matching uses zero-based arm positions).

The general analysis covers if/for/range, labelled and unlabelled transfers,
expression/type switches, fallthrough and short-circuit expressions. Switch
selectors are explicit predicates. Transfers retain their original source
operation. A private statement view adds lookup markers before transfers omitted
by x/tools; compiler-owned syntax is not changed. Repeated lowerings reuse each
short-circuit predicate. Compiler-known boolean outcomes and short-circuit
constant results remove impossible general-flow transfers.

## Limits

General control evidence is approximate. Calls are assumed to return except
resolved built-in panic, os.Exit, runtime.Goexit and resolved log Fatal/Panic variants. Implicit panics, arbitrary
non-returning callees, call effects, panic recovery, deferred execution and
concurrent/blocking execution are not complete. Bodies containing
defer, recover, go, select, sends or receives get unsupported control evidence
and no lexical fallback. No complete Go PDG or interprocedural claim is made.
Range iteration uses the CFG alternatives, without a type-specific termination
proof. General expression lowering does not prove every Go operand evaluation
order. Data-flow, alias and conformance capabilities remain separate.

The fixed-point calculation uses O(V(V+E)) work and O(V²) bits. General analysis
fails explicitly above 16,384 operation-CFG vertices; it does not report a clean
or complete result after truncation. Control edges can themselves require
quadratic output. This is a correctness/coverage change, not a speed claim.
The general extraction profile is gstyle.abstraction/0.4.0.

## Consumers and checks

Matching receives semantic control edges, including multiple controllers.
Single-parent helpers reject ambiguity. Legacy case/counter/propagation rewrites
run only when the actual control relation is an acyclic hierarchy; they do not
flatten a general PDG into source containment. Other matching behavior is
unchanged. An empty dependency set is not evidence of independence.

A second oracle checks every directed graph through four vertices (66,066),
including self loops and disconnected cycles. It independently searches for a
terminal path or reachable cycle avoiding each candidate. It checks exact
post-dominance and branch-labelled control edges against the fixed-point producer.
Hand-derived source regressions cover transfers, nested control, joins, explicit
exits, constant outcomes, short-circuiting, switches and unsupported constructs.
These finite checks are not a proof for all graphs or all Go semantics.

## Scalar data flow across cycles

The scalar portion of the general profile computes reaching definitions for
local integer/bool functions containing for loops, switches, labelled statements and transfers.
The entire body and interface are checked. Calls, memory operations, reference
values, closures, range, deferred execution and concurrency are excluded from
this capability. Unsupported functions retain partial evidence; the new solver
does not fill their missing data edges with first-binding guesses. The broader
conservative binding and memory lowering below handles ordinary sequential
functions outside that scalar subset.

The solver computes the least fixed point of predecessor unions and kill/gen
transfers over the finite set of source definitions. The entry always includes
formal input and named-result definitions, even when a back edge reaches entry.
Only changed block outputs schedule successors. Reads do not alter this analysis
domain. After convergence, each read receives its reaching definitions once;
multiple definitions form an explicit value join. This avoids making a fresh
merge on each loop iteration. No iteration cutoff can turn incomplete analysis
into success. The operation-CFG size limit applies before data analysis starts.

Data and control analysis share one source-derived operation CFG. Short-circuit
outcomes, constant boolean outcomes, switch decisions and branch transfers use
the same operation identities. Unreachable returns keep their source nodes for
navigation but cannot transfer a value to a function output. The existing
acyclic scalar-normal matching profile and its admission contract stay separate;
these cyclic functions are not certified by the acyclic verifier.

A path-search oracle checks 198,198 graph/definition layouts: every directed graph
through four vertices, with three write layouts over two bindings. The oracle
follows each definition until a write kills it, independently of the producer's
fixed-point solver. Source cases check zero/many iterations, reads before writes,
kill/gen, nested loops, loop post statements, break/continue, goto, shadowing,
short circuit, switches/fallthrough and named returns. Six deliberate changes to
the producer fail test assertions. These checks establish finite test coverage,
not correctness for all Go programs.

## Schema verification

Use the single-profile schema at compat revision
56edd6bb804b10edaf9bbc2343d69e5e298d8c1a, which this producer already follows.
Older schema copies require per-edge/count policies and a conformance array;
those fields were intentionally removed when compat adopted one root extraction
profile. Failures against that older schema are not exporter defects. Set
COMPAT_PDG_SCHEMA to the selected schema when running the export schema test.

## Shared decision flow for runtime guards

For bodies supported by sequential semantic control lowering, runtime nil/index
analysis and execution dependencies consume the same operation CFG. Old
statement-anchor execution edges are replaced rather than retained alongside it. Its condition references identify the
PDG decisions themselves. Boolean wrappers (if, for, short circuit and tagless
switch) read their boolean operand through PDG value edges; tagged switch
predicates retain both comparison operands. Runtime analysis no longer invents
a second interpretation of switch alternatives. A single surviving constant
outcome is unconditional in the retained flow and is not re-labelled true.

A range decision does not write the iteration bindings. A separate synthetic
binding operation lies on the successful iteration edge, with the original
writes and definition backreferences. Assignment-target expressions are also evaluated on that iteration edge, not
in the preheader where x/tools places them. The false edge retains incoming
binding values, including on zero iterations. Runtime range-index facts apply at that
binding operation. Source tests retain real nil accesses as well as reject false
switch warnings and false nonnegative range bounds.

Bodies with deferred or concurrent constructs still use the older
coarse value-flow lowering. That remaining path is tracked in pdg-work-plan.md;
shared sequential lowering does not establish those capabilities. Runtime flow
metadata advances to gstyle.runtime:flow/0.2.0. Formal verification status lives
in verification/lean/.

## General bindings and may-alias storage

Ordinary sequential functions outside the exact scalar subset now use the same
fixed-point definition solver. Strong direct binding updates replace older
values. Indirect stores and opaque-call effects are weak updates: both prior
values and possible writes reach later reads. Explicit location identities
represent addressable bindings, allocation sites and unknown external storage.
Memory is field-insensitive and allocation sites can represent many objects.
Unknown pointers expand to the represented may-alias places; missing precision
must not become a no-alias assertion.

The pdgmemory domain package solves finite copy/address/allocation constraints
from canonical operands and typed annotations. It does not inspect Go syntax.
The adapter emits type, storage-access and assignment-slot facts once. A separate
reachability oracle covers 1,570 copy-graph/seed cases. Source tests cover alias
stores, disjoint bindings, parallel declarations, nested pointers, allocation,
opaque calls and range element dependencies. This evidence does not prove full
Go memory semantics. Closures, deferred/exceptional flow and concurrency remain
outside the shared sequential lowering.

Compact matching composes canonical paths through omitted reads and joins while
preserving operand slots and dependency kinds. Constant counter seeds belong to
the loop matching abstraction; their canonical operations and edges remain for
semantic consumers. Side-effecting initializers retain matching operations.
Matching and semantic views consume shared operation reachability, so unreachable
calls and taint sinks no longer enter those views. The versioned
`gstyle.pdg:operation-flow` extension selects that context; legacy partial value
flow is not sufficient. Structural validation permits cycles and memory while
checking references, site uniqueness and reachability. This does not widen the
strict scalar-normal conformance contract.

## Range expression evaluation and type-switch bindings

A range with at most one iteration variable omits its expression when that
expression or its length is compiler-constant. A trailing blank value is absent
for this rule. Shared operation lowering emits a source-backed constant bound
instead of executing the original expression. Control and iteration-value edges
consume that bound. The original operations remain for navigation but have no
execution sites. Thus ranging over `*p` with a constant array length does not
invent a dereference; requesting element values still evaluates it. Nonconstant
calls and channel receives prevent constant-length elision. Constant subexpressions
and conversions follow their typed compiler facts. Generic core-type bounds and
exact iteration cardinality remain further work.

Each used type-switch clause receives a separate definition for its compiler
implicit variable. A case-entry operation owns that definition and depends on
the switched value; it executes only on that clause's selected path. This also
covers default. The guard's assignment syntax is not a shared binding definition
or an unknown memory write. Representation refinement is conservative. This is
source-to-PDG lowering; dependency consumers still use canonical operations and
edges.

## Output success uses operation flow

The output-completion consumer follows canonical operation sites and successors,
not adjacent AST statements. It computes a least fixed point for normal paths
that all reach a nil error result. Calls and operations without completion
facts block the claim, as do infinite paths. Compiler facts distinguish safe
scalar operations from possible interface-comparison, division, shift and address
faults. These facts describe each operation itself; evaluating its operands is
still separate. Tests cover branches, gotos, local work, fault boundaries and an
independent finite path oracle. Error-use classification and initial output-call
operand completion still have source-level boundaries; deferred and concurrent
completion remains unsupported here.

## Error-use facts follow definitions

Error-check facts now use canonical definitions and value operands. The selected
error-result slot follows reaching reads, explicit merges and copied bindings;
a later read after overwrite cannot check the earlier definition. Returns,
arguments/receivers and control predicates consume values through the PDG.
Classification still means reachable may-use, not universal handling.

Unknown dependency scope remains explicit in `CheckUnknown`; those sites are
excluded from belief mining and violations. Missing aggregate or indirect-use
support cannot become a definite unchecked error. Source syntax is retained for
result-slot/type recognition and discard shape, with no second dependency walk.

## Closure construction

A function literal is one construction operation in its parent flow. Its body is
not a parent execution path. Captured local bindings have addressable storage and
capture-address dependencies; package globals remain global storage. Calls carry
a separate call-target dependency at position zero, outside argument positions.
Opaque calls weakly write represented captured cells. Closure bodies, resolved
result summaries and nonreturning behavior remain unknown. This support does not
establish deferred, exceptional or concurrent execution semantics.

## Nilerr paths

Nilerr uses the shared operation flow for guard reachability and normal
continuations. Source statement order does not supply a separate path proof.
Gotos, breaks and terminal calls use the same successors as other consumers.
Canonical write symbols and operand positions also select copy transfers and
return values, including named-result reads. Missing or conflicting operands
remain unknown. Weak alias writes cannot establish a definite replacement.
Expression/predicate dependencies also use canonical operators and operands.
Compiler constants, literal shapes and types supply boundary facts. Nil guards
follow actual comparison and transparent-wrapper edges to read operations.
Storage paths now use canonical symbols, compiler field indexes and actual base
operands. Handling follows graph value inputs and writes. Allocation identities
distinguish fresh construction from uncertain storage mutation. Compiler types
and literal shapes remain boundary facts. Heap aliases and opaque effects remain
conservative. Legacy partial flow and unrepresented predicates cannot establish
a finding. This does not establish exceptional or deferred flow.
