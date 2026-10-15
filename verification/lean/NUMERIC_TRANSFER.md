# Numeric constraint transfer

`make verify-numeric-transfer` checks the production `graphView.constrain` body
for bounds whose `Nil` field is false. It also proves the bodies of
`trackedBinding`, `lengthConstraint`, `exactInteger`, `emptyRange`,
`impossibleSignedComparison` and `infeasibleConstraint`. It calls the existing
proved binding and integer-constraint contracts.

## Execution contract

`wp_constrain_numeric` owns a non-nil state map, the graph view, and its kind,
instability and tracking slices. The slices have equal lengths. The supplied
key is a valid one-based reference, and the supplied kind, instability and
tracking values agree with the corresponding slice entries. The usual Go
semantics, layout, function-unfolding and package-initialization hypotheses
remain explicit. The machine model uses 64-bit Go ints.

The theorem returns all metadata unchanged and gives the exact state contents
and return location:

- An untracked binding returns the original state without changes.
- A tracked binding starts with the actual effective binding lookup. Unstable
  bindings use the graph base value, even when a cached record exists.
- Length bounds with `<` or `<=` set `UpperChecked`. Strict `<` also sets
  `BelowLength`; an existing `BelowLength` flag is retained. Other record
  fields are preserved by this step.
- A bound is exact only when both flags are set and its endpoints are equal.
  Exact bounds use the existing integer interval constraint. Nonexact bounds
  do not refine the interval.
- An empty interval or an impossible strict signed comparison returns nil
  without writing to the input map. The caller still owns the original map.
- Otherwise, only the target key is inserted or replaced. Other keys retain
  their full records. The result is the original map location.

`constraint_preserves_other_keys` states the map preservation property for
all other keys. Present zero records, absent target keys and unknown raw bounds
are distinguished by the existing binding contract.

## Conditional interval soundness

`constraint_state_sound` and `wp_constrain_numeric_sound` show that a represented
integer survives the transfer and its branch is not rejected when:

1. The input effective binding interval contains the integer.
2. The integer is in the signed 64-bit range.
3. If the bound is exact, the integer satisfies the selected comparison with
   the bound's signed value.

The execution theorem covers all operator strings. The existing interval
relation imposes no predicate for unsupported operators, including `!=`,
which conservatively leaves interval bounds unchanged. The proof does not
claim a precise interval exclusion for inequality.

Length flags have exact execution contracts. Their semantic connection to an
actual collection length still requires sound input facts. The interval theorem
does not prove that flags alone establish safe indexing. Instability is covered:
the output effective lookup still uses the graph base value.

## Safeguards

The profile audits fourteen named theorems against the existing standard-axiom
allowlist and rejects `sorryAx` and opaque implementation axioms transitively.
It regenerates and byte-compares runtimeguard, interval, PDG and query code.
Twenty-one Go mutations check exact/empty predicates, length flags, signed
limits, tracking, the chosen bound, rejection and the map key. Mutations must
fail in the expected proof file with the existing recognized proof diagnostics;
setup and translation errors do not count as rejection.

CI uses three shards and the existing private incremental build copies and two
mutation workers. Existing profile shard counts and safeguards remain unchanged.
Go regressions check explicit outcomes and sampled preservation across signed
extremes, unknown and contradictory bounds, instability and other map entries.

## Separate stages

Nil/pointer constraints, pointer/alias semantics, invalid references, unequal
metadata slice lengths and concurrent mutation are outside this contract.
Assignments, comparison normalization, recursive branch composition,
PDG extraction and concrete program soundness remain unproved here. Fixpoint
convergence and package initialization require their own contracts and stages.
This PR adds no production Go annotations or replacement model implementation.

Signed increments and decrements have a separate [transfer proof](SIGNED_STEP.md).
