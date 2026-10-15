# Logical branch composition

[Shared snapshot contracts](SHARED_FLOW.md) now let the logical layer use the same
read-only PDG snapshot as value and comparison queries. Child execution and truth
premises remain explicit.

`make verify-branch` checks the production `graphView.logical` and
`graphView.operand` bodies. The logical execution theorem is modular: it assumes
explicit execution contracts for the two child `assume` calls. The semantic
wrapper additionally assumes child soundness. Those assumptions are recorded
proof obligations, not verified callee implementations.

## Confirmed bug fix

An alternative branch used to pass two nil child states into `join`. The join's
existing copy contract deliberately returns an allocated empty map for two nil
arguments. In logical composition this resurrected an unreachable branch.

For example, inside `if i >= 0`, the condition `i < 0 || i < 0` has no feasible
true alternative. The old composition nevertheless analyzed its body from an
empty state and could report a false positive at a later guarded index access.
The false branch of `i >= 0 && i >= 0` had the same problem. Both regressions
fail before this change and pass after it. Logical composition now returns nil
when both alternatives are nil. The join's standalone nil/copy behavior stays
unchanged.

## Actual operand lookup

`wp_operand` proves the actual nested slice/array read for a valid one-based
operation reference and positions zero or one. Both array elements, the entire
operand slice and the graph view are preserved. `wp_operand_zero` and
`wp_operand_outside` prove the sentinel cases without receiver ownership: a zero
reference or a position at least two returns zero before reading the view.
Negative positions on nonzero references and out-of-range operation references
remain outside these contracts. The model uses 32-bit PDG references and 64-bit
Go ints.

## Composition contract

`wp_logical` consumes a non-nil owned input map and preserves the graph view,
operand, kind and instability slices plus an arbitrary additional resource
predicate used by the children. The row and both operand slots must agree with
the supplied child references. Kind and instability functions agree with the
metadata slices for cached keys; their lengths agree. Cached references are
valid, and child contracts preserve that validity.

For true conjunction and false disjunction, composition is sequential. The left
child refines the input. A nil result returns nil without calling the right
child; otherwise the right child receives that refined state. The callee owns
the consumed state, so the original map is not promised unchanged in this case.

For true disjunction and false conjunction, both children receive separate
copies of the original input. The original map is returned unchanged. Two nil
children return nil. A single surviving child is copied exactly. Two surviving
children use the existing actual state-map join contract, including exact key
coverage and conditional interval soundness. Output ownership on forks prevents
aliasing with the preserved original map. Allocated empty maps remain feasible
states and are never confused with nil.

The execution theorem covers every operator string according to the production
dispatch; the Boolean soundness theorem restricts it to `&&` and `||`.

## Semantic contract and explicit assumptions

`branch_result_sound` composes two refiners over an abstract integer environment.
If the incoming abstract state denotes that environment, the composite Boolean
condition has the selected truth value, and each child soundly retains matching
environments, the result still denotes the environment. The sequential cases
require both child predicates; fork cases require either. The existing join
preserves environments represented by either child.

`wp_logical_sound` connects that relation to execution of the actual logical
body. Its child execution hypotheses are full Hoare contracts: they consume an
owned input map, return the matching optional state and preserve the metadata
resources. Its child semantic hypotheses describe condition truth in the same
abstract environment. This stage does not prove either child contract.

The abstract environment and kind/instability functions must describe the
queried numeric bindings to obtain a concrete interpretation. Agreement for
uncached queries, expression evaluation, effects in conditions and PDG truth
facts are not established here. Child predicates over a fixed environment do
not by themselves model Go condition side effects. Normal Go semantics,
type/layout, function-unfolding and package-initialization hypotheses remain
explicit.

## Checks and next obligations

Nine named theorems have a fail-closed standard-axiom audit. Fifteen source
mutations cover sequential/fork selection, missing short-circuiting, stale
sequential input, copy aliasing, child and truth selection, both-nil resurrection,
dropped live alternatives, missing joins and operand boundaries. Regenerated
runtimeguard, interval, PDG and query code is byte-compared. Existing diagnostic
and setup-error safeguards remain unchanged. CI adds two shards; pinned
checks, private incremental build copies, targeted modules, the two-worker
limit and all prior shard counts remain in place.

Go tests cover all Boolean truth combinations, allocated-empty states, original
map isolation, sequential stopping, operand sentinels and the full detector
regressions. The modified production function remains within complexity ceilings.

The next PDG/condition stage must discharge the child execution and semantic
contracts, establish metadata and truth-fact provenance, and connect the
assignment and complete transfer paths. Convergence, pointer/alias semantics
and package initialization remain separate contracts and stages.
