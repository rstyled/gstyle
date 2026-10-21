package patterns

type ResourceState string

const (
	ResourceReleased      ResourceState = "released"
	ResourceReturned      ResourceState = "returned"
	ResourceReturnedOwner ResourceState = "returned_owner"
	ResourceUnresolved    ResourceState = "unresolved"
	ResourceInWorker      ResourceState = "worker"
)

// ResourceLifecycle keeps cleanup and transfer distinct. Neither transfer nor
// missing proof establishes whether a recipient eventually releases a resource.
type ResourceLifecycle struct {
	ResourceAcquisition
	SourceNode uint32 `json:"-"`
	State      ResourceState
	Release    string
	Reason     string
	Worker     *ResourceWorker `json:"worker,omitempty"`
}

// ResourceWorker separates cleanup on modeled exits from shutdown ownership.
// A cancellation receive does not prove that the caller ever cancels it.
type ResourceWorker struct {
	Function             string
	HandoffProven        bool
	CleanupProvenOnExit  bool
	CleanupDeferObserved bool
	Shutdown             string
}

func lifecycleViolations(f *MiningFacts) []ObligationViolation {
	var violations []ObligationViolation
	for _, evidence := range f.ResourceLifecycles {
		if evidence.State == ResourceReleased || evidence.State == ResourceReturned {
			continue
		}
		violations = append(violations, ObligationViolation{
			FuncID: f.ID, Line: evidence.Line, Resource: evidence.Resource,
			AcquireCall: evidence.AcquireCall, Reason: evidence.Reason, Lifecycle: &evidence, DeferredCleanup: evidence.DeferredCleanup,
		})
	}
	return violations
}

func hasLifecycle(f *MiningFacts, acq acquisition) bool {
	source := f.Pdg.Nodes[acq.node].SourceNode
	for _, evidence := range f.ResourceLifecycles {
		if source != 0 && evidence.SourceNode == source && evidence.Resource == acq.Resource {
			return true
		}
	}
	return false
}
