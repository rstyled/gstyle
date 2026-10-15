# Research Papers Behind Gstyle

The static analysis research that gstyle implements, with the key insight
from each paper, what gstyle took from it, and how faithfully it was implemented.
These are the "buried gold" papers — old work that modern compute (and gstyle's
PDG extractor) unlocks.

## Summary

| Paper | Year | Venue | Pattern kind | Fidelity |
|---|---|---|---|---|
| Zou et al. (CCGraph) | 2020 | ASE | `ccgraph_clone` | Adapted |
| Gabel, Jiang, Su | 2008 | ICSE | Removed LSH adaptation | Not implemented |
| Li et al. (CP-Miner) | 2004 | OSDI | `inconsistent_clone` | Adapted |
| Li & Zhou (PR-Miner) | 2005 | FSE | `mined_rule` | Adapted |
| Engler et al. | 2001 | SOSP | `deviant_behavior` | Inspired by |
| Thummalapenta & Xie (Alattin) | 2009 | ASE | `alattin_rule` | Adapted |
| Krinke (Barrier slicing) | 2004 | SQJ | library only | Adapted |
| Weimer & Necula | 2004 | OOPSLA | `obligation` | Adapted |
| Jackson & Rollins (Chopping) | 1994 | CMU | library only | Adapted |
| Sridharan, Fink & Bodik (Thin slicing) | 2007 | PLDI | library only | Adapted |
| Binkley & Harman (Dependence clusters) | 2015 | ICSME | `dependence_cluster` | Adapted |
| Horwitz (Semantic diff) | 1990 | PLDI | library only | Inspired by |
| Bulychev & Minea (Anti-unification) | 2008 | — | `parameterize` fixer | Exact |

**Fidelity levels:**

- **Exact:** Implemented as the paper describes, with paper's thresholds/parameters.
- **Adapted:** Core idea from the paper, modified for gstyle's architecture or Go.
- **Inspired by:** Paper motivated the approach; implementation differs significantly.
- **Not implemented:** No current implementation of the paper; any removed adaptation is documented below.

---

## Implemented

### CCGraph — Zou, Ban, Xue & Xu, ASE 2020

**Paper:** "CCGraph: a PDG-based code clone detector with approximate graph matching"
**DOI:** 10.1145/3324884.3416541

**The insight:** PDG-based clone detection was stuck on exact subgraph isomorphism
(NP-hard, slow, misses clones). CCGraph replaces exact matching with approximate
graph matching via the Weisfeiler-Lehman (WL) graph kernel, plus a two-stage
filtering strategy using characteristic vectors to prune candidates before the
expensive kernel computation.

**The current pipeline in gstyle (adapted from the paper):**

1. Build nine numerical PDG features: declaration, assignment, control, call and other node counts; control, data and execution edge counts; and reference-binding count.
2. Admit a pair when feature cosine similarity is ≥ **0.9**, or short function-name Jaro-Winkler similarity is ≥ **0.5**. Missing source features or names permit conservative admission. The report route also admits pairs through its AST similarity bypass.
3. Build light WL summaries for functions in admitted pairs. Retain histograms and node counts, without refinement history or graph adjacency.
4. Verify **every admitted pair** with the WL kernel at ≥ **900/1000**, then group accepted pairs by connected components. There is no LSH stage or 512-dimensional retrieval vector.

**Fidelity:** Adapted. Admission uses numerical match OR name fallback. Full
parameter strings, size filters, and the paper's exact matching policies are not
yet reproduced. Go reference bindings and approximate must-precede execution
edges supply two of the numerical features. The shared contract is tracked in [compat PR #8](https://github.com/rstyled/compat/pull/8). These local policies do not
establish compat conformance or reproduction of the original paper. The separately named
`match-graphs` command now implements imported projected-stage matching against
`compat.ccgraph.strict/0.1.0` at revision `04d9aa0`; its shared stage fixtures test
projection, features, interfaces, directed WL, scoring and boundaries. It
also accepts native source through the bounded Go contract below. It does not
reproduce the original CCGraph tool or establish detector quality.

### Gstyle Go scalar normal

The explicit conformance path `gstyle match-graphs --source ROOT` registers
`gstyle.go.scalar-normal/0.2.0` in its run manifest. Admission follows
[compat's validation and projection contract](https://github.com/rstyled/compat/blob/56edd6bb804b10edaf9bbc2343d69e5e298d8c1a/schemas/ccgraph-profile.md#1-boundary-versions-and-admitted-evidence).
This is a Go extraction policy; it does not claim Rust HIR extraction.

- Admit acyclic integer/bool local and parameter operations, variable declarations,
  single ordinary assignments, increment/decrement, `if`, explicit returns,
  single named-result bare returns, unary expressions, and short-circuit logic.
  Check the complete body, including unreachable syntax. Reject calls, memory
  operations, loops, transfers, closures, concurrency, generic functions/receivers,
  and variadic inputs.
- Reject independently evaluated sibling operands, compound assignments,
  multiple initializer/return expressions, and multiple result slots. A selected
  source order is not a Go execution proof. This follows
  [compat's unspecified-order policy](https://github.com/rstyled/compat/blob/56edd6bb804b10edaf9bbc2343d69e5e298d8c1a/schemas/ccgraph-decisions.md)
  and [Go's evaluation-order rules](https://go.dev/ref/spec#Order_of_evaluation).
- Preserve every operation occurrence, including literal and variable reads.
  Parenthesized values have explicit identity computations; write-target
  parentheses do not create value operations. Logical tests are source-backed
  lowered predicates. Reaching-definition joins are nonexecuting values.
  Actual formal inputs are retained even when unused. Meaningful formal outputs
  represent result slots, including named-result initial zero definitions;
  unreachable returns cannot transfer into live outputs. Void fallthrough has
  no fabricated value operation.
- Count distinct local/parameter symbols read on reachable normal flow. Receivers
  use parameter binding roles; named results use local roles. Interface receiver
  and return roles remain distinct. Resolve integer/bool aliases and named types
  from compiler facts, preserving original type spellings. Transport primitives
  in `gstyle.go.scalar-normal:resolved-type` on symbols/interface elements;
  native integer width comes from the actual loaded target.
- A separate validator checks source inventory/kinds/operators/literals,
  scopes/roles/reads/writes/types, CFG entry/transitions/true-false outcomes,
  sparse dominator execution and post-dominator control edges, scalar definition
  paths/joins, operand/return edges, interfaces, and projected reference counts.
  Normal CFGs exclude nonexecuting entities. Optional context output uses one
  operation per CFG node, source spans and operation IDs shared with the PDG,
  synthetic administrative entry/exit, and explicit return transfers.
- Preserve panic/unwinding omissions: paired CFG evidence stays `partial`.
  Emit source content identities, Go toolchain/flags/OS/architecture/target width,
  root profile registration, and every projection decision. No producer JSON
  claim alone passes native admission. Repeated primitive evidence lists share
  storage; source bytes/compiler objects are not retained in routine PDGs.

The imported matcher fixtures remain pinned to `04d9aa0`; native admission targets
`56edd6b`, whose matching-stage conventions are unchanged. The
[independently labeled development/held-out study](https://github.com/rstyled/compat/blob/56edd6bb804b10edaf9bbc2343d69e5e298d8c1a/schemas/ccgraph-profile.md#10-fixtures-quality-and-implementation-gate)
remains separate and pending. Do not use this implementation to claim complete
Go coverage, original-tool reproduction, detector quality, or safe refactoring.

**What the advisory `patterns` matcher adds:** AST bypass, parallel pair checks, shared IR label hashes,
cached vector norms, a safe WL upper bound, and bounded match batches. These
additions are separate from the paper's algorithm. Removing LSH retains these
filters; it does not enable exhaustive matching over all usable functions.

**In gstyle:** `ccgraph_clone` pattern kind. `domain/patterns/ccgraph.go`, `domain/patterns/wl.go`.

**Note:** CCGraph replaced gstyle's earlier `semantic_clone` implementation (Komondoor & Horwitz style BFS subgraph enumeration), which caused 6-minute/4GB hangs on large codebases. The old code is deleted.

---

### CP-Miner — Li, Lu, Myagmar & Zhou, OSDI 2004

**Paper:** "CP-Miner: Finding Copy-Paste and Related Bugs in Large-Scale Software Code"

**The insight:** Find copy-pasted code, then check whether the copies were
*consistently* modified. Inconsistent edits to clones are bugs — e.g., someone
fixed a nil check in one copy but forgot the pasted copies.

**How gstyle implements it:** Rebuilt on CCGraph groups (the original implementation
depended on the deleted BFS enumeration). Within each CCGraph clone group, compute
pairwise WL similarity. If a pair scores in [900, 950) — above the clone threshold
but below the "identical" range — it's flagged as suspiciously diverged.

**Fidelity:** Adapted. The core "inconsistent edits to clones = bugs" idea is
CP-Miner's; the divergence detection via WL similarity bands is gstyle's.

**In gstyle:** `inconsistent_clone` pattern kind (detection-only). `domain/patterns/inconsistent.go`.

---

### PR-Miner — Li & Zhou, FSE 2005

**Paper:** "PR-Miner: Automatically Extracting Implicit Programming Rules and Detecting Violations in Large Software Code"

**The insight:** Frequent itemset mining over function-call co-occurrences extracts
*implicit* programming rules with zero annotations. "In this repo, `sql.Query`
results are always closed." Found 16 confirmed bugs in Linux. The self-supervised
version of every hand-written linter rule.

**How gstyle implements it:** Mine rules of form `{A} → B` (if you call A, you
usually also call B). Support ≥10%, confidence ≥90%. Violations are flagged as
bugs.

**Fidelity:** Adapted. Core frequent-itemset mining approach; thresholds tuned for
gstyle's use case.

**In gstyle:** `mined_rule` pattern kind. `domain/patterns/pr_miner.go`.

---

### Bugs as Deviant Behavior — Engler, Chen, Hallem, Chou & Chelf, SOSP 2001

**Paper:** "Bugs as Deviant Behavior: A General Approach to Inferring Errors in Systems Code"

**The insight:** Extract programmer *beliefs* from code, then find contradictions.
MUST beliefs (always true — violations are always bugs) vs. MAY beliefs
(statistically true — violations ranked by deviation). You don't need to know the
truth; a contradiction IS the bug. Engler commercialized this as Coverity.

**How gstyle implements it:** Mine beliefs about error handling: if 95%+ of callers
check a function's error return, the sites that don't are flagged. Uses a
parent-tracking AST walker to detect `_` assignments and ignored results.

**Fidelity:** Inspired by. The MUST/MAY belief distinction is Engler's; gstyle's
implementation focuses specifically on error-handling beliefs.

**In gstyle:** `deviant_behavior` pattern kind. `domain/patterns/deviant.go`, `adapters/gopatterns/errorcheck.go`.

---

### Alattin — Thummalapenta & Xie, ASE 2009

**Paper:** "Alattin: Mining Alternative Patterns for Detecting Neglected Conditions"

**The insight:** Classic belief miners (PR-Miner, Engler) find X→Y and flag ¬Y as
bugs. But API preconditions are usually disjunctive: "check arg for null OR check
return for error" — not "check both." Alattin mines *alternative* (OR) patterns
and only flags code that violates *all* alternatives. Cut false positives ~28%.

**How gstyle implements it:** Disjunctive rules over call sets. Where PR-Miner mines
"if you call A, you also call B", Alattin mines "if you call A, you call B OR C".
The OR rule only fires when neither single rule meets the confidence threshold.

**Fidelity:** Adapted. Core disjunctive mining idea; implemented on gstyle's PDG
call facts.

**In gstyle:** `alattin_rule` pattern kind (detection-only). `domain/patterns/alattin.go`.

---

### Barrier Slicing — Krinke, SQJ 2004

**Paper:** "Barrier Slicing and Chopping"

**The insight:** Declare parts of the dependence graph off-limits, and the slice
can't transitively pass through them. Unifies two ad-hoc practices: removing
dependences from the graph, and dicing (removing known-good code from a slice).

**How gstyle implements it:** `BarrierSlice(pdg, seed, barriers)` computes the
backward slice from a seed without traversing through barrier nodes. Use case:
the agent marks reviewed code as a barrier, and the fault area shrinks each
iteration.

**Fidelity:** Adapted. Core barrier concept; implemented on gstyle's PDG.

**In gstyle:** Library only (`domain/patterns/barrier.go`). Not yet wired as a CLI pattern kind.

---

### Error Handling Obligations — Weimer & Necula, OOPSLA 2004

**Paper:** "Finding and Preventing Run-Time Error Handling Mistakes"

**The insight:** Flow-sensitive dataflow tracks outstanding *obligations* —
resources that must be released on *every* path, including error paths. Error
paths are where resource discipline breaks. The paper found 800+ error-handling
mistakes in ~4M LOC of Java from pure static analysis.

**How gstyle implements it:** Links acquisitions to local deferred releases.
For typed `os` file acquisitions, it also follows normal control flow through
direct closes and local single-argument helpers that always close their parameter.
The acquisition's error arm is excluded only while its original error binding is
unchanged. Conditional cleanup, early returns, cycles before cleanup, rebinding,
address escapes, goroutines, closures, and unavailable helper bodies retain the
warning. Helper chains and exceptional termination are outside this analysis.
These are advisory findings; an unrecognized cleanup path is not proof of a leak.

**Fidelity:** Adapted. Core obligation-tracking idea; adapted for Go's error-handling
idioms.

**In gstyle:** `obligation` pattern kind (detection-only). `domain/patterns/obligation.go`.

---

### Chopping — Jackson & Rollins, 1994

**Paper:** "Chopping: A Generalization of Slicing" (CMU-CS-94-169)

**The insight:** A *chop* between a source and a sink is the intersection of the
forward slice of the source with the backward slice of the sink. Answers "how does
THIS input reach THAT output?" rather than "what affects X?"

**How gstyle implements it:** `Chop(pdg, source, sink)`. For `check --changed`:
given changed lines (sources) and sinks (DB writes, network calls), show exactly
the statements on the path between them.

**Fidelity:** Adapted. Core chop definition; implemented on gstyle's PDG.

**In gstyle:** Library only (`domain/patterns/chop.go`).

---

### Thin Slicing — Sridharan, Fink & Bodik, PLDI 2007

**Paper:** "Thin Slicing"

**The insight:** Traditional slices are too big because they include everything
that MIGHT affect a value — control dependences, heap plumbing. A *thin slice*
includes only PRODUCER statements: the chain of assignments that compute and copy
the value. 3-9x fewer statements; the buggy statement is still included.

**How gstyle implements it:** `ThinSlice(pdg, seed)` follows only Data edges
backward. When explaining a candidate, show the thin slice (5 lines) instead of
the full dependence cone (50).

**Fidelity:** Adapted. The paper has a more precise definition; gstyle uses a
heuristic (Data edges only).

**In gstyle:** Library only (`domain/patterns/thin_slice.go`).

---

### Dependence Clusters — Binkley & Harman, ICSME 2015

**Papers:** "Coherent Dependence Clusters" (JSS); "Dependence Cluster Visualization" (ICSME 2015)

**The insight:** A *dependence cluster* is a set of statements that are mutually
dependent (strongly connected in the dependence graph). Large programs contain
clusters spanning 10%+ of the codebase. Big clusters = "everything affects
everything" spaghetti. *Linchpin functions* are the ones whose removal collapses
a cluster — highest-leverage refactoring targets.

**How gstyle implements it:** Tarjan's SCC algorithm finds mutually-dependent
regions. Flags functions where 10+ statements are tangled, or 25%+ of the
function is one cluster.

**Fidelity:** Adapted. Core SCC-based cluster detection; thresholds chosen for
gstyle.

**In gstyle:** `dependence_cluster` pattern kind (detection-only). `domain/patterns/dep_cluster.go`.

---

### Semantic Diff — Horwitz, PLDI 1990

**Paper:** "Identifying the Semantic and Textual Differences Between Two Versions of a Program"

**The insight:** Compare versions by their *dependence graphs*, not their text.
A refactoring that moves code produces zero semantic diff; a one-line change
that alters dependences is flagged as significant.

**How gstyle implements it:** `SemanticDiff(old, new)` compares PDG structure
hashes ignoring line numbers. For `check --changed`: pure refactors get a free
pass; behavior changes get scrutiny.

**Fidelity:** Inspired by. The "compare PDGs not text" idea; gstyle's implementation
is simpler than Horwitz's full partitioning.

**In gstyle:** Library only (`domain/patterns/semantic_diff.go`).

---

### Anti-Unification — Bulychev & Minea, 2008

**Paper:** "Duplicate Code Detection Using Anti-Unification"

**The insight:** Anti-unification is the mathematical dual of unification. Given
two similar fragments, formally derive the template + minimal parameter list.
The *provably correct* way to extract a helper — not heuristics, but the
most-specific generalization.

**How gstyle implements it:** Drives the `parameterize` fixer. Instead of
heuristic diffing, anti-unification derives the exact template and holes, which
become the extracted function's parameters. Respects variable renames.

**Fidelity:** Exact. Standard anti-unification algorithm.

**In gstyle:** `domain/patterns/antiunify.go`. Wired into the `parameterize` fixer.

---

## Taint Analysis (SAST literature)

**Pattern kind:** `taint_flow` (detection-only)

Taint analysis doesn't come from a single paper — it's the standard
source→propagator→sink framework from security-focused static analysis (SAST)
engineering. Gstyle's implementation:

- **Sources:** `r.URL.Query().Get()`, `r.FormValue()`, `os.Getenv()`, `io.ReadAll()`, etc.
- **Propagators:** String concatenation, assignment, function returns (conservative)
- **Sanitizers:** `template.HTMLEscapeString()`, `url.QueryEscape()`, `strconv` numeric parsing
- **Sinks:** `db.Exec()`/`Query()` (arg 1 only — parameterized queries are safe), `exec.Command()`, `template.Execute()`, `os.Open()` (path arg)

**Key design decision:** Only flags tainted data in the *query string* argument
(arg 1) of SQL calls. `db.Exec("... WHERE id = ?", userID)` is the safe pattern
and is NOT flagged. `db.Exec("... WHERE id = '" + userID + "'")` IS flagged.

**Algorithm:** Intraprocedural forward dataflow along PDG data edges. Seed sources
as tainted, propagate to fixpoint, flag sinks with tainted dangerous args.

**In gstyle:** `domain/patterns/taint.go`. Wired into `gstyle patterns` CLI.

---

## Removed

### LSH Adaptation — Gabel, Jiang & Su, ICSE 2008

**Paper:** "Scalable detection of semantic clones"
**DOI:** 10.1145/1368088.1368132

**The insight:** Select meaningful PDG subgraphs, map them to AST forests, and
use DECKARD vectors and size-partitioned LSH near-neighbor search. This is not
whole-function WL vectorization. Sources: [Gabel §2.2–3.3](https://web.cs.ucdavis.edu/~su/publications/icse08-clone.pdf)
and [DECKARD §3.3](https://web.cs.ucdavis.edu/~su/publications/icse07.pdf).

**Status in gstyle:** Removed. The former whole-function WL/sign-hash adaptation
expanded collisions into connected components and provided no pruning on the
measured corpora. Clone detection now verifies every characteristic/name/AST-admitted
pair with WL, without LSH. This is not a Gabel implementation.
See the CCGraph section above for the current matching boundary.

---

### Semantic Clones via BFS Enumeration — Komondoor & Horwitz, SAS 2001

**Paper:** "Using Slicing to Identify Duplication in Source Code"

**What it was:** Found isomorphic PDG subgraphs via breadth-first search
enumeration of 10-12 node subgraphs, with WL hashing + isomorphism verification.

**Why it was removed:** The BFS enumeration caused 6-minute/4GB hangs on
codebases with 2,500+ functions. CCGraph (above) replaces it with better
recall, better precision, and orders of magnitude better performance.

**Status:** Deleted. `ccgraph_clone` is the only clone detector.

---

## Not Yet Implemented

### The PDG Foundation — Ferrante, Ottenstein & Warren, TOPLAS 1987

"The Program Dependence Graph and Its Use in Optimization." The paper that
invented the PDG. Everything above builds on this.

### Interprocedural Slicing — Horwitz, Reps & Binkley, 1988-1990

System dependence graphs, precise interprocedural slicing. Gstyle's PDG is
currently intraprocedural; this is the path to cross-function analysis.

---

## The Thesis

Gstyle is the AI's guide for attention and static analysis. These papers are the
buried gold — old ideas that were right but too slow, too manual, or too
academic to productize. Modern compute + gstyle's PDG extractor unlocks them.

The pattern: **static analysis tells the agent where to look; the agent decides
what to do.** Detection is deterministic. Fixes are conservative. Attention is
the scarce resource.
