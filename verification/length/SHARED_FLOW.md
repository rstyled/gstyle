# Shared metadata for logical composition

`make verify-shared-flow` connects production join and logical execution to the shared PDG snapshot used by value and comparison queries. It generalizes metadata permissions in `wp_operand`, `wp_join`, `wp_join_sound`, `wp_logical`, and `wp_logical_sound`. Existing full-ownership callers remain supported. Join and operand permissions default to full ownership; logical permissions follow the supplied child contracts.

The fraction applies only to the graph view, operand rows, kind slice, and instability slice. State-map ownership stays exclusive. The join still requires separately owned, disjoint non-nil input maps. This stage does not add support for aliased join inputs. Existing nil join contracts are unchanged.

`querySnapshot_branchResources` derives the logical layer's read-only resources from `querySnapshotOwn`. `snapshot_child_to_branch` adapts child execution contracts that preserve that snapshot. `wp_join_snapshot` preserves both input maps and the complete snapshot, produces a separate output map, and establishes exact key union and interval preservation for either input.

`wp_logical_snapshot` covers sequential and alternative composition, nil rejection, copying, and joins with the shared snapshot intact. Its map result and metadata validity follow the existing logical execution model. `wp_logical_snapshot_sound` also preserves represented numeric environments under the existing child truth and soundness assumptions. Semantic composition is restricted to `&&` and `||`; the execution contract follows production dispatch for every operator string.

Input assumptions remain explicit:

- The input state map is allocated, non-nil, and exclusively owned. Cached keys have valid one-based metadata references and agree with the supplied kind and instability interpretations.
- The operation has a valid operand row and two supplied operand slots. Kind and instability slices have equal length.
- Both child `assume` execution contracts preserve the snapshot and metadata validity. Their implementations are not discharged by this stage.
- The semantic wrapper requires the selected Boolean condition and the child soundness premises in the same fixed numeric environment.
- The existing Go semantics, layout, helper-unfolding, package-initialization, and 64-bit execution assumptions hold.

The profile audits ten named theorems and checks six mutations of the actual production Go bodies. Controls attempt writes to kinds, instability, operand rows, and the graph view during join, logical, or operand execution. They must fail in the designated logical proof module; setup failures do not count. Existing state-map and branch mutation suites remain enabled. CI adds one shard and retains pinned dependencies, exact canonical translations, source hashes checked before and after verification, private build copies, and the two-worker limit.

Production Go behavior does not change. Remaining work includes metadata-validity preservation for numeric constraints, finite composition of the actual `assume` dispatcher, complete transfer/CFG integration, producer and parser truth, and query termination. Pointer/alias semantics, package initialization, and convergence remain separate stages.
