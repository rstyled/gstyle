# Result-controlled normal branches

Ordinary exit evaluation joins condition observations over the same caller
states as call continuation observations. It records both outcomes when a
condition is unknown. Only a unanimous Boolean observation can prune a branch.
Missing observations establish nothing. Loop visits and continuations after an
unknown call participate in the join.

The binding query removes canonical truth wrappers to retain source
correspondence. The adapter uses those node references when it rebuilds the
shared normal CFG. It then recomputes execution, control and reaching-definition
edges. Source operations remain in the graph, including unreachable ones.
Compiler-owned value facts remain separate from these ordinary observations.

The graph retains its original ExitContext for deferred/inherited exit queries.
An ordinary observation must not replace the semantics of recover during unwind.
The result and exit solver continues to use that original context.

`TestResultConditionsPruneSharedGraph` covers both arms, scalar and tuple copies,
nil results, argument contexts, mixed results, overwrites, loop joins, unknown
prefixes and short circuits. It checks reachability and the absence of execution
and control edges to unreachable source operations. This is finite test evidence.
General call-result payloads, heap execution and deferred state execution remain
outside this contract. Direct result facts for other consumer engines remain
separate work.
