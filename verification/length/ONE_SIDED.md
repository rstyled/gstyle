# One-sided numeric comparison soundness

`make verify-one-sided` connects the four production one-sided comparison execution contracts to all-key numeric interval preservation. It covers left or right bindings on true or false branches. The exact map update comes from the selected production comparison body and its existing execution proofs.

The other operand has an explicit mathematical value and a value record that represents it. It need not have a symbol. The proof does not interpret an absent symbol as binding zero. Right-side constraints reverse the normalized operator; false branches invert before reversal.

`numeric_bound_state_sound` proves that an exact bound and the selected integer relation prevent rejection and preserve every binding valuation. Non-exact bounds impose no extra numeric constraint. The four execution wrappers preserve the original allocated state-map location, the PDG snapshot, and any borrowed global operator tables, and establish the resulting map's interval representation for every key. Untracked and out-of-range cached IDs remain conservative under the existing constraint metadata contract.

Assumptions remain explicit:

- The input map is allocated and non-nil and represents the supplied numeric valuation for every key.
- The updated binding's concrete value lies in the signed 64-bit range. Snapshot kind and instability entries agree with the supplied interpretation for that binding.
- Operand and symbol queries succeed with finite derivations against a valid shared PDG snapshot. The other operand's abstract interval contains its supplied concrete value.
- The selected branch condition holds. The selected operator-table entries implement the proved inversion and reversal functions.
- The bound has `Nil = false`, and the value evaluator retains its explicit `ParseInt` execution contract.

These contracts do not establish producer-fact truth, expression side effects, pointer/nil or alias semantics, package initialization, the parser implementation, convergence, or general query termination. They do not compose the full `assume` or analysis transfer loop. General nil-state operand evaluation and 32-bit execution remain outside this stage.

The profile audits five named theorems and applies fourteen semantic mutations. Controls cover operand direction, truth polarity, inversion/reversal, the represented bound value, input representation, signed word limits, rejection, the updated key, and the returned valuation. They must fail at a logical proof obligation in the designated module; client type errors and setup failures do not count. The existing comparison profile continues to check its forty Go/model mutations. No production Go behavior changes.

Go regressions run 2,400 concrete cases across all six operators, signed endpoints, both binding sides, bound-presence combinations, and tracking choices. They check feasible-branch preservation, an unrelated binding, and returned map identity. CI adds one shard and keeps pinned dependencies, exact canonical translations, source hashes checked before and after verification, private build copies, and the two-worker limit.
