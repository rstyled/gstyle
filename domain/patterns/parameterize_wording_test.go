package patterns

import (
	"strings"
	"testing"
)

func TestPartialAlignmentDoesNotClaimOnlyDifferences(t *testing.T) {
	cluster := &Cluster{CoverageMilli: []uint32{1000, 600}}
	summary := summaryOf(&candidateIndex{}, []*MiningFacts{{}, {}}, cluster)
	if strings.Contains(summary, "differ only") || !strings.Contains(summary, "80% mean coverage") {
		t.Fatalf("overclaims partial alignment: %s", summary)
	}
}
