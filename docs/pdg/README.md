# Compact PDG IR and compat export

Cyclo extracts one native `domain/pdg.Graph` per function. It represents the
fields in [compat PR #8](https://github.com/rstyled/compat/pull/8), draft 0.1.0,
pinned at `e1d08c2500e4184444fed70c9d9d1df206ac6962` in
[compat-0.1.0.schema.json](compat-0.1.0.schema.json). Structural validity does
not establish shared extraction or matching conformance.

## Export and validation

```sh
cyclo patterns --format pdg-json . > pdg.json
python3 docs/pdg/validate-export.py pdg.json
```

Export writes a JSON array with one compat document per function. It includes
sources, interfaces, symbols, definitions, memory locations, operations,
dependencies, provenance, capability evidence, diagnostics and extensions.
Unknown facts retain reasons; absent optional facts stay absent. Required empty
collections are arrays. Normal `--format json` still writes the pattern report.

Native validation runs before export. The separate schema check requires Python
`jsonschema`; it is not a Cyclo runtime dependency. Export skips mining, matching
and its cache; `--cache` is rejected. Expanded records exist for one function at
a time, while extraction retains the compact function graphs. JSON repeats shared
evidence as required by the schema. Discard partial output if the command fails.

## Storage and extraction

Strings, full types, labels, policies, evidence and common provenance share
interned storage across functions. Sparse attributes and lists are packed and
deduplicated. Zero references mean absent; known empty strings have a text ID.
Measured amd64 hot records are 44 bytes per node and 56 per edge. `Compact`
removes excess capacity. Construction maps, ASTs and compiler objects are not
retained. Builders have one owner; shared tables and lists are immutable by convention.

The producer records source byte spans, typed interfaces, distinct syntactic
definitions, reads/writes and call targets. Unresolved memory targets remain
unknown; shared storage is not invented. Data edges use approximate first-binding
or syntactic dependencies, and control edges use approximate lexical nesting.
They must not be treated as precise reaching-definition or post-dominator facts.

`cyclo.go-must-precede/1` adds a sparse, approximate precedence basis from reachable
statement order and dominance. CFG successor links are inputs, not dependencies.
`cyclo.go-reference-bindings/1` counts distinct typed Go reference-bearing bindings;
generic reference kinds remain unknown. `cyclo.go-return-values/1` links explicit
returns to formal output slots, including tuple calls. Bare returns record reads
of named results. Nested closure returns do not connect to the outer function.
Deferred result writes and implicit/exceptional exits remain incomplete.

## Matching boundary and remaining work

Pattern mining derives a temporary `MiningGraph`; no legacy graph copy is stored
in the IR. Labels are separate from base categories and IDs. The report route
admits pairs by nine-feature cosine ≥ 0.9 OR short-name Jaro–Winkler ≥ 0.5, with
an AST admission bypass and conservative handling of missing source facts.
Every admitted pair receives WL verification at ≥ 900/1000, then accepted pairs
form connected groups. LSH is removed. This remains a Cyclo adaptation of CCGraph.

The numerical order is declarations, assignments, controls, calls, other nodes,
control edges, data edges, execution edges and reference bindings. Formal output
nodes stay outside the WL view; canonical data-edge counts include return links.
Light WL summaries retain histograms and node counts. Shared label hashes, cached
norms, a safe similarity bound and bounded match batches limit repeated work.

`cyclo.ccgraph-pairs/1` explicitly streams sorted unordered pair evidence without
AST bypass or grouping; it requires known nine-feature facts and names.
`compat.ccgraph-base/draft` is unavailable. Shared projection, parameter strings,
size/admission gates, label/depth/weight/score policies and independent Rust/Go
stage fixtures still need agreement. Reaching definitions, branch/loop merges,
post-dominator control, captures, aliases and exceptional/concurrent dependencies
also remain incomplete. Graph evidence states these limits.

## Measurements and checks

Earlier extraction measurements show retained graph heap +33% and full extraction
retained heap +17%, with substantially higher construction allocation volume.
Interning limits retained duplicates; it does not make construction free.

The latest Prometheus return-link comparison uses three fresh runs per mode and
route, alternating order on commit `2213c3c192574b8a339acd336ea6385974a7567b`.
The comparison disables only the return-link pass; both builds have no LSH and
use the host Go cache, with output sent to `/dev/null`.

| Mode | Without return links | With return links | Peak RSS before → after |
| --- | ---: | ---: | ---: |
| Pattern analysis | 3.922 s | 4.010 s | 589.0 → 623.6 MiB |
| PDG JSON export | 6.660 s | 6.686 s | 669.4 → 675.6 MiB |

These are medians, not general performance guarantees. Small timing differences
can be run variation. Peak RSS is not retained IR storage. A failed cold-cache
attempt that hit disk quota is excluded. Raw data and extended notes stay local.

Run `make check`, `make cognitive` and `make quality-gate`. Extraction, dependency,
packing, export and profile tests cover the new boundaries. Optional memory and
profiling harnesses stay in tests. Schema tests use the separate Python validator.
