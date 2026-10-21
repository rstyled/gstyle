# Resource lifecycle findings

Resource obligations are advisory. Missing cleanup evidence does not establish a
leak. Returning a resource moves the obligation to its caller; it does not prove
that the caller fulfills it. Cleanup advice therefore refers to the owner's
lifetime. A `defer` in a constructor can close or stop a resource before use.

The existing file analysis retains its factory and cleanup contracts. Typed
protocols also cover `time.NewTicker`, `time.NewTimer`, `net.Dial`,
`net.DialTimeout`, `net.Listen`, `database/sql.Open`, and `database/sql.OpenDB`.
Each protocol uses exact acquisition and release declarations. Same-package
factory and cleanup summaries use the existing canonical resource-flow engine.
Aliases, conditional paths, and the identity of the released resource matter.
Unknown calls, dynamic targets, and missing graph evidence remain conservative.

Supported sequential cases distinguish local release, direct return to a
caller, and a resource carried in a returned composite object. Returned-object
cleanup stays unresolved: finding a resource in an object does not establish
that the object's lifecycle releases it. Acquisitions inside a returned composite
literal are also recorded as ownership observations, with no cleanup proof.
Arbitrary field updates and cross-package
ownership protocols are not inferred.

JSON obligation candidates may contain `resource_lifecycle` with a state,
resource identity, release method, reason, and worker observations. `released`
and `returned` facts do not produce a local obligation candidate. `returned_owner`,
`worker`, and `unresolved` still require review. There is no automatic fix.

## Worker observations

For named same-package goroutine calls, worker evidence separates:

- `HandoffProven`: canonical value flow establishes the acquisition argument.
- `CleanupProvenOnExit`: cleanup is recognized on modeled worker exits. This is
  protocol evidence, not a proof of termination or external method behavior.
- `CleanupDeferObserved`: a typed defer names the selected parameter binding.
  This observation alone does not prove registration, capture identity, or execution.
- `Shutdown`: a typed context `Done` receive with a direct return is observed,
  an explicit background context is supplied, or shutdown remains unresolved.

Goroutine and select bodies currently have partial sequential-flow coverage.
Source observations never upgrade that coverage or discharge an obligation.
A parameter rebind can make the observed defer refer to a different value;
conditional registrations may never execute. Such cases remain unresolved.
An observed cancellation receive does not prove that a caller cancels the
context. Stopping a ticker does not terminate its receiving goroutine.

Anonymous workers, arbitrary context wrappers, variadic/tuple-packed worker
calls, and aliases without canonical value evidence remain unresolved. The
analysis does not infer application lifetime intent or verify whole-program
shutdown. Go 1.23 and later can collect unreferenced timers and tickers without
an explicit Stop; this does not collect objects retained by live worker goroutines.
