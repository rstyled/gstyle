package patterns

func workerLines(lifecycle *ResourceLifecycle) []string {
	if lifecycle == nil || lifecycle.Worker == nil {
		return nil
	}
	worker := lifecycle.Worker
	handoff := "observed; resource value flow unproved"
	if worker.HandoffProven {
		handoff = "resource value flow recognized"
	}
	return []string{
		"  worker: " + worker.Function,
		"  handoff: " + handoff,
		"  shutdown: " + workerShutdownText(worker.Shutdown),
		"  cleanup: " + workerCleanupText(worker),
	}
}

func workerShutdownText(shutdown string) string {
	switch shutdown {
	case "non_cancellable_context":
		return "non-cancellable context"
	case "cancellation_receive; caller_cancellation_unverified":
		return "cancellation receive observed; caller cancellation unverified"
	default:
		return "unresolved"
	}
}

func workerCleanupText(worker *ResourceWorker) string {
	if worker.CleanupProvenOnExit {
		return "recognized on modeled exits; worker termination unproved"
	}
	if worker.CleanupDeferObserved {
		return "defer observed; execution unproved"
	}
	return "unproved; no cleanup defer observed"
}
