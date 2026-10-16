package patterns

import (
	"fmt"
	"slices"
	"strings"
)

// Resource-obligation tracking (Weimer & Necula, OOPSLA 2004):
// "Finding and Preventing Run-Time Error Handling Mistakes".
//
// Flow-sensitive dataflow tracks outstanding *obligations* — resources that
// must be released on *every* path, including error paths. Error paths are
// where resource discipline breaks: the paper found 800+ error-handling
// mistakes in ~4M LOC of Java from pure static analysis.
//
// Go's `defer` is a gift: the common case is syntactic. An acquired resource
// with no `defer release()` is the highest-value signal — it catches the
// most common bug pattern (a forgotten Close/Unlock before an early return).
//
// Local deferred releases are linked through the PDG. The typed Go extractor
// also records bounded normal-path cleanup for os file acquisitions, including
// local single-argument helpers that close their parameter on every path.
// Returning the acquired file transfers its close obligation to the caller.
// Missing evidence remains advisory; it does not prove a leak.

// obligationRule pairs a resource-acquiring callee with its release method.
type obligationRule struct {
	// acquire is a CalleeID suffix, e.g. "os.Open", "Mutex).Lock".
	acquire string
	// release is the release method name, e.g. "Close".
	release string
	// resource is the human name, e.g. "file".
	resource string
	// receiverIsResource is true when the receiver IS the resource
	// (e.g. mu.Lock()), rather than the call returning it (os.Open).
	receiverIsResource bool
}

// obligationRules is the hardcoded acquisition→release map.
var obligationRules = []obligationRule{
	// Files.
	{"os.Open", "Close", "file", false},
	{"os.Create", "Close", "file", false},
	{"os.OpenFile", "Close", "file", false},
	{"os.CreateTemp", "Close", "file", false},
	{"ioutil.TempFile", "Close", "file", false},
	{"os.Pipe", "Close", "pipe", false},
	// Databases.
	{"sql.Open", "Close", "database", false},
	{"sql.OpenDB", "Close", "database", false},
	// Network.
	{"net.Dial", "Close", "connection", false},
	{"net.Listen", "Close", "listener", false},
	// Locks: the receiver is the resource.
	{"Mutex).Lock", "Unlock", "mutex", true},
	{"RWMutex).Lock", "Unlock", "rwmutex", true},
	{"RWMutex).RLock", "RUnlock", "rwmutex", true},
	// Timers.
	{"time.NewTicker", "Stop", "ticker", false},
	{"time.NewTimer", "Stop", "timer", false},
	// Compressed I/O.
	{"gzip.NewReader", "Close", "gzip reader", false},
	{"zip.OpenReader", "Close", "zip reader", false},
	// HTTP: the release is resp.Body.Close(), linked through the Body field.
	{"http.Get", "Close", "http response body", false},
	{"http.Post", "Close", "http response body", false},
	{"http.Client).Do", "Close", "http response body", false},
}

// ResourceAcquisition is one acquired resource carrying a release obligation.
type ResourceAcquisition struct {
	FuncID string
	Line   int
	// Resource is the human name, e.g. "file".
	Resource string
	// AcquireCall is the acquiring callee suffix, e.g. "os.Open".
	AcquireCall string
}

// ObligationViolation is an acquired resource with no deferred release.
type ObligationViolation struct {
	FuncID string
	Line   int
	// Resource is the human name, e.g. "file".
	Resource string
	// AcquireCall is the acquiring callee suffix, e.g. "os.Open".
	AcquireCall string
	// Reason explains the missing release.
	Reason string
}

// acquisition is a ResourceAcquisition plus its PDG node and release method.
type acquisition struct {
	ResourceAcquisition
	node          int
	release       string
	receiverIsRes bool
}

// isReleaseCall reports whether a Call node invokes the release method.
func (n PdgNode) isReleaseCall(release string) bool {
	return n.Kind == Call && strings.HasSuffix(n.CalleeID, "."+release)
}

// FindObligationViolations finds acquisitions without recognized cleanup.
// Each violation is an advisory candidate, not proof of a leak.
func FindObligationViolations(facts []*MiningFacts) []ObligationViolation {
	var out []ObligationViolation
	for _, f := range facts {
		if f.Pdg == nil {
			continue
		}
		out = append(out, violationsInFunc(f)...)
		out = append(out, returnedFileViolations(f)...)
	}
	return out
}

func returnedFileViolations(f *MiningFacts) []ObligationViolation {
	var out []ObligationViolation
	for _, site := range f.FileObligationSites {
		out = append(out, ObligationViolation{FuncID: f.ID, Line: site.Line, Resource: site.Resource, AcquireCall: site.AcquireCall, Reason: "returned file has no recognized close or further ownership transfer"})
	}
	return out
}

// violationsInFunc checks one function's acquisitions for missing defers.
func violationsInFunc(f *MiningFacts) []ObligationViolation {
	pdg := f.Pdg
	deferred := deferredCallNodes(pdg)
	fwd := forwardDataAdj(pdg)
	var out []ObligationViolation
	acquisitions := findAcquisitions(pdg, f.ID)
	for _, acq := range acquisitions {
		if dischargedFile(f.ReleasedFileLines, acquisitions, acq) {
			continue
		}
		if hasDeferredRelease(pdg, acq, deferred, fwd) {
			continue
		}
		out = append(out, ObligationViolation{
			FuncID:      acq.FuncID,
			Line:        acq.Line,
			Resource:    acq.Resource,
			AcquireCall: acq.AcquireCall,
			Reason: fmt.Sprintf("%s acquired via %s has no deferred %s",
				acq.Resource, acq.AcquireCall, acq.release),
		})
	}
	return out
}

// Line-only evidence must never discharge a second acquisition on the same line.
func dischargedFile(lines []int, acquisitions []acquisition, target acquisition) bool {
	if target.Resource != "file" || !slices.Contains(lines, target.Line) {
		return false
	}
	count := 0
	for _, acq := range acquisitions {
		if acq.Line == target.Line {
			count++
		}
	}
	return count == 1
}

// findAcquisitions returns all resource acquisitions in a PDG.
func findAcquisitions(pdg *MiningGraph, funcID string) []acquisition {
	var out []acquisition
	for i, n := range pdg.Nodes {
		if n.Kind != Call {
			continue
		}
		rule, ok := matchAcquisition(n.CalleeID)
		if !ok {
			continue
		}
		out = append(out, acquisition{
			ResourceAcquisition: ResourceAcquisition{
				FuncID:      funcID,
				Line:        n.Line,
				Resource:    rule.resource,
				AcquireCall: rule.acquire,
			},
			node:          i,
			release:       rule.release,
			receiverIsRes: rule.receiverIsResource,
		})
	}
	return out
}

// matchAcquisition reports the rule for a callee, if it acquires a resource.
func matchAcquisition(calleeID string) (obligationRule, bool) {
	for _, r := range obligationRules {
		if strings.HasSuffix(calleeID, r.acquire) {
			return r, true
		}
	}
	return obligationRule{}, false
}

// deferredCallNodes returns the Call node indices wrapped by a Defer node.
// `defer f()` lowers to a Call node with a Data edge into a Defer node.
func deferredCallNodes(pdg *MiningGraph) map[int]bool {
	deferNodes := collectDeferNodes(pdg)
	return collectDeferredCalls(pdg, deferNodes)
}

// collectDeferNodes returns the indices of all Defer nodes.
func collectDeferNodes(pdg *MiningGraph) map[int]bool {
	out := map[int]bool{}
	for i, n := range pdg.Nodes {
		if n.Kind == Defer {
			out[i] = true
		}
	}
	return out
}

// collectDeferredCalls returns Call nodes with a Data edge into a Defer node.
func collectDeferredCalls(pdg *MiningGraph, deferNodes map[int]bool) map[int]bool {
	out := map[int]bool{}
	for _, e := range pdg.Edges {
		if e.Kind == Data && deferNodes[e.To] && pdg.Nodes[e.From].Kind == Call {
			out[e.From] = true
		}
	}
	return out
}

// forwardDataAdj builds forward adjacency over Data edges.
func forwardDataAdj(pdg *MiningGraph) map[int][]int {
	adj := map[int][]int{}
	for _, e := range pdg.Edges {
		if e.Kind == Data {
			adj[e.From] = append(adj[e.From], e.To)
		}
	}
	return adj
}

// hasDeferredRelease reports whether a deferred release call is data-linked
// to the acquisition — e.g. `defer f.Close()` where f came from `os.Open`,
// or `defer mu.Unlock()` where mu was the `mu.Lock()` receiver.
func hasDeferredRelease(pdg *MiningGraph, acq acquisition, deferred map[int]bool, fwd map[int][]int) bool {
	root := acq.node
	if acq.receiverIsRes {
		if recv := dataSourceAt(pdg, acq.node, 0); recv >= 0 {
			root = recv
		}
	}
	reached := reachableFrom(fwd, root, len(pdg.Nodes))
	for callIdx := range deferred {
		if !pdg.Nodes[callIdx].isReleaseCall(acq.release) {
			continue
		}
		if anyDataSourceReached(pdg, callIdx, reached) {
			return true
		}
	}
	return false
}

// dataSourceAt returns the source of the Data edge into node at argPos,
// or -1 if none.
func dataSourceAt(pdg *MiningGraph, node, argPos int) int {
	for _, e := range pdg.Edges {
		if e.To == node && e.Kind == Data && e.ArgPos == argPos {
			return e.From
		}
	}
	return -1
}

// anyDataSourceReached reports whether any Data source of node was reached.
func anyDataSourceReached(pdg *MiningGraph, node int, reached map[int]bool) bool {
	for _, e := range pdg.Edges {
		if e.To == node && e.Kind == Data && reached[e.From] {
			return true
		}
	}
	return false
}

// reachableFrom returns nodes reachable via forward edges within maxDepth.
// Covers Let bindings (`f, _ := os.Open()`) and field hops (resp.Body).
func reachableFrom(fwd map[int][]int, start, maxDepth int) map[int]bool {
	seen := map[int]bool{start: true}
	frontier := []int{start}
	for d := 0; d < maxDepth && len(frontier) > 0; d++ {
		var next []int
		for _, n := range frontier {
			for _, m := range fwd[n] {
				if !seen[m] {
					seen[m] = true
					next = append(next, m)
				}
			}
		}
		frontier = next
	}
	return seen
}

// ObligationCandidates converts obligation violations to candidates.
func ObligationCandidates(violations []ObligationViolation, facts []*MiningFacts) []Candidate {
	factByID := map[string]*MiningFacts{}
	for _, f := range facts {
		factByID[f.ID] = f
	}
	var out []Candidate
	for _, v := range violations {
		f := factByID[v.FuncID]
		if f == nil {
			continue
		}
		out = append(out, Candidate{
			Kind:        Obligation,
			ScoreMilli:  800, // High: a leaked resource is a bug.
			Observation: fmt.Sprintf("%s acquired via `%s` has no deferred release", v.Resource, v.AcquireCall),
			Inference:   v.Reason,
			PossibleRefactor: fmt.Sprintf(
				"add `defer` for the release right after acquiring the %s", v.Resource,
			),
			Sites: []Site{
				{Path: f.Path, Line: v.Line, Name: f.Name},
			},
		})
	}
	return out
}
