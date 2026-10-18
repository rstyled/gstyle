# Values during deferred execution

On the parent revision, a deferred closure reads a changed Boolean capture as
unknown. `flag:=false; defer func(){if flag {panic("yes")}}(); flag=true`
reports return-or-panic, although native Go always panics. A named result changed
by a deferred closure also stays unknown. The exit solver now carries live cell
values through deferred execution and publishes results after unwind completes.

Each registration saves its target and explicit argument evaluation context.
The remainder of the body produces suffix completions. The registered call then
runs against each completion, with live lexical cells and saved explicit
arguments. Older registrations run after it. A completion keeps the exit mode,
cell state, saved return tuple and owner recovery marker together.

Named result slots receive explicit return values before unwind and are read
after unwind. Unnamed results retain the return-expression snapshot. A recovered
return without a return expression uses compiler type facts for Boolean false
and nilable zero values. Missing or malformed facts do not establish a zero.

Capture summaries now distinguish normal return, inherited-panic recovery,
panic, panic during Goexit and goroutine exit. Ordinary exceptional calls apply
their captured postconditions before the caller unwinds. A nested callee that
recovers its own panic does not recover an ancestor's pending panic. Goexit runs
older defers and does not itself provide a recoverable panic.

## Registration loops

The analysis does not store a bounded dynamic defer stack. A local least fixed
point solves suffix completion sets at control/value states. A cycle contributes
possible divergence and the finite completions found so far. Later rounds add
new finite completions until the sets stop changing. Registration count is not
part of the key. This retains both Boolean parities for an unknown number of
registered flips, without an arbitrary count limit or an eager Cartesian
transfer table. Return and capture summaries publish only after stabilization.

## Evidence and scope

Twenty-six source cases cover live captures, saved scalar/tuple arguments, LIFO
stores, exceptional postconditions, nested recovery, Goexit, named/unnamed
results, an unknown registration count, and 257 registrations. Native Go runs
the same traces and agrees with the predicted outcome sets. An independent
explicit-stack oracle checks all 510 combinations of up to three registered
Boolean operations, initial values and return/panic/Goexit contexts.

Nine compiling fault controls fail their semantic checks: frozen live cells,
live explicit arguments, missing named assignment, lost unnamed snapshots,
lost exceptional writes, discarded finite loop completions, dropped defers,
lost owner recovery and skipped Goexit unwind.

This extends the existing admitted Boolean/nil and frame-confined cell model.
Repeated allocation families still use conservative histories. Missing bodies,
unconfined stores and opaque effects retain unknown behavior. Full heap state,
callbacks, concurrent/blocking flow, all implicit faults and full exceptional
PDG dependencies remain open. These finite checks add no full-language proof.
