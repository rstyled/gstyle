package patterns

import (
	"strings"
	"testing"
)

func TestTextShowsWorkerEvidenceWithoutStrongerProof(t *testing.T) {
	for _, test := range []struct {
		name                       string
		worker                     ResourceWorker
		handoff, shutdown, cleanup string
	}{
		{"observed defer", ResourceWorker{CleanupDeferObserved: true, Shutdown: "non_cancellable_context"},
			"observed; resource value flow unproved", "non-cancellable context", "defer observed; execution unproved"},
		{"unverified cancellation", ResourceWorker{CleanupDeferObserved: true, Shutdown: "cancellation_receive; caller_cancellation_unverified"},
			"observed; resource value flow unproved", "cancellation receive observed; caller cancellation unverified", "defer observed; execution unproved"},
		{"missing cleanup evidence", ResourceWorker{Shutdown: "unresolved"},
			"observed; resource value flow unproved", "unresolved", "unproved; no cleanup defer observed"},
		{"recognized cleanup", ResourceWorker{CleanupProvenOnExit: true},
			"observed; resource value flow unproved", "unresolved", "recognized on modeled exits; worker termination unproved"},
		{"recognized cleanup with defer", ResourceWorker{CleanupProvenOnExit: true, CleanupDeferObserved: true},
			"observed; resource value flow unproved", "unresolved", "recognized on modeled exits; worker termination unproved"},
		{"recognized handoff", ResourceWorker{HandoffProven: true, CleanupDeferObserved: true, Shutdown: "unresolved"},
			"resource value flow recognized", "unresolved", "defer observed; execution unproved"},
		{"unknown shutdown", ResourceWorker{Shutdown: "future_status"},
			"observed; resource value flow unproved", "unresolved", "unproved; no cleanup defer observed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			worker := test.worker
			worker.Function = "pkg.owner.run"
			report := &PatternsReport{Candidates: []Candidate{{Kind: Obligation,
				ResourceLifecycle: &ResourceLifecycle{State: ResourceInWorker, Worker: &worker}}}}
			before, err := JSON(report)
			if err != nil {
				t.Fatal(err)
			}
			text := Text(report)
			want := "  worker: pkg.owner.run\n  handoff: " + test.handoff + "\n  shutdown: " + test.shutdown + "\n  cleanup: " + test.cleanup
			if !strings.Contains(text, want) {
				t.Errorf("missing worker evidence %q:\n%s", want, text)
			}
			after, err := JSON(report)
			if err != nil {
				t.Fatal(err)
			}
			if before != after {
				t.Fatal("text rendering changes structured analysis evidence")
			}
		})
	}
}

func TestTextOmitsAbsentWorkerEvidence(t *testing.T) {
	for _, lifecycle := range []*ResourceLifecycle{nil, {State: ResourceUnresolved}, {State: ResourceReturnedOwner}} {
		text := Text(&PatternsReport{Candidates: []Candidate{{Kind: Obligation, ResourceLifecycle: lifecycle}}})
		for _, label := range []string{"  worker:", "  handoff:", "  shutdown:", "  cleanup:"} {
			if strings.Contains(text, label) {
				t.Errorf("unexpected %q without worker evidence:\n%s", label, text)
			}
		}
	}
}
