# Finite exits and possible divergence

The exit solver now uses the admitted body evaluator in two passes. It no longer
seeds both invocation contexts from a separate static recursive-call graph.
That graph includes calls whose guards are false and saved deferred calls that
an abort never invokes.

First, finite completion outcomes grow from bottom to a least fixed point.
These include normal return, recovery, panic, suspended Goexit, goroutine exit,
process abort and explicit unknown execution. The divergence bit is excluded
from this pass. Missing bodies or flow retain unknown execution.

Second, every supported invocation context starts with possible divergence.
The solver holds the finite outcomes fixed and removes divergence until it
reaches a greatest fixed point. Local CFG cycles remain possible divergence.
Interprocedural cycles remain only when the actual evaluator can reach them in
that context. Ordinary and deferred-panic contexts remain separate.

The order matters. A finite prefix must be known to return before its recursive
continuation can contribute divergence. Deferred registration alone is not an
invocation: abort exits without running the saved stack. A recovery-guarded
recursive call can terminate in one context and diverge in another. Unknown
calls do not establish that a later continuation runs. Absence of a divergence
bit is not a termination proof when unknown execution remains.

Source cases cover these distinctions, true ordinary/deferred recursion, local
loops, finite prefixes, panic and Goexit. Native Go checks a recovery-guarded
recursive handler that finishes after two calls. Resource cleanup uses the same
summaries: an older close can run after a terminating handler, while ordinary
recursion can still prevent Goexit cleanup.

A separate finite graph oracle enumerates all directed graphs of one through
three functions, with independent return choices. Transitive closure identifies
reachable cycles without calling the production solver's reachability code.
The 3,140 ordinary/deferred observations validate this bounded model, not full
Go lowering or general termination. The obsolete static call-graph engine and
its narrower reachability-only test are removed.

The solver still depends on the evaluator's admitted paths and unknown effects.
General argument/result specialization beyond the nilness/Boolean contexts in
`pdg-call-arguments.md`, recovered payloads, deferred heap/result bodies,
complete exceptional PDG edges, callbacks and concurrency remain required work.
No new formal proof is claimed.
