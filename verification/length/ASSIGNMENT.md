# Assignment selection and state-map writes

`make verify-assignment` checks the production bodies of `singleValueWrite`,
`graphView.assignmentValue` and the non-pointer path of `graphView.assign`.
It regenerates the actual PDG category type and declaration constant as well as
runtimeguard, interval and query code. The declaration tag is not an opaque axiom.

## Execution contracts

`wp_singleValueWrite` covers every category byte, form string and machine-int
write count. A single `=` or `:=` selects the RHS. An empty form selects it only
for a declaration. All other cases reject it. Negative, zero and multiple write
counts reject it. The machine model uses 64-bit Go ints.

`wp_singleValueWrite_compound` covers all eleven compound operators: `+=`, `-=`,
`*=`, `/=`, `%=`, `&=`, `|=`, `^=`, `<<=`, `>>=` and `&^=`. None selects the
unmodified RHS as the result of the compound operation. Increment and decrement
are also ineligible here; their separate production step helper has its own proof.

`wp_assignmentValue` covers actual base-value normalization for every kind.
Unsigned bindings acquire lower bound zero only when the input record has no
known lower bound. Existing lower bounds and all other fields are preserved.
Signed and other kinds retain the entire input record.

`wp_assign_nonpointer` covers the actual state-map insertion. It requires full
ownership of the non-nil map, graph view and kind slice, a valid one-based target
reference, agreement with the kind slice and a non-pointer target kind. The
operation reference is arbitrary because this path does not use pointer identity
allocation. The output map is exactly the old map with the normalized record
inserted at the target key. Every other key remains unchanged. The graph view
and kind slice are returned unchanged. The Value argument is passed by value.

The existing Go-semantics, type/layout, method-unfolding and initialization
hypotheses remain explicit. Invalid targets, nil maps and concurrent writes are
outside these execution contracts. Pointer assignment and alias invalidation
remain a separate stage.

## Interval meaning and compound fallback

`assignment_interval_sound` and `wp_assign_interval_sound` preserve every
integer represented by the RHS interval. For unsigned targets the represented
integer must additionally be nonnegative. This is an input-value assumption;
the proof does not claim that RHS expression evaluation or conversion is sound.
The theorem quantifies mathematical integers, so it does not silently truncate
unsigned values to signed 64-bit words.

The current production transfer uses an unknown Value for compound assignments.
`wp_assign_unknown_signed` proves that storing this fallback at a signed target
preserves every possible integer result and clears the entire old record. This
is conservative soundness, not precise compound arithmetic. Operations that
panic produce no normal result; their panic behavior is not proved here.

The eligibility and map-write contracts are separate pieces of the full transfer.
This stage does not formally compose the `analysis.transfer` loop, RHS reads,
tracking gate, guard-marker clearing, field invalidation or range writes. Those
remain integration obligations for the PDG/control-flow stages. Go regressions
execute the production transfer for every compound form and all three plain
assignment forms, including guard clearing and RHS preservation.

## Checks

Eight named theorems have a fail-closed standard-axiom audit. Fifteen Go mutations
cover write counts, declaration tags, compound eligibility, unsigned defaults,
record clobbering, missing writes, wrong keys and skipped normalization. Mutations
must fail in the expected proof file; setup failures do not count. CI adds two
shards and keeps private incremental build copies, pinned dependency checks,
targeted modules, the two-worker limit and prior profile shard counts.

Go regressions check raw metadata, replacement and insertion, preservation of
other records and caller values, unsigned defaults and the full production
fallback path. Production Go behavior does not change in this stage.

## Remaining work

Branch composition is next. PDG extraction, RHS evaluation and the full transfer
integration still need contracts and proofs. The earlier length-flag provenance
and type/operator dispatch gaps remain. Convergence, pointer/alias semantics
and package initialization retain separate contracts and stages.
