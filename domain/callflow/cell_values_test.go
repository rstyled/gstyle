package callflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestCellHistoryJoinAgainstSequenceOracle(t *testing.T) {
	for size := 1; size <= 4; size++ {
		combinations := 1
		for range size {
			combinations *= 5
		}
		for encoded := 0; encoded < combinations; encoded++ {
			history := map[pdg.Ref]exitValue{}
			remaining := encoded
			first := scalarValues()[remaining%5]
			uniform := true
			for range size {
				value := scalarValues()[remaining%5]
				remaining /= 5
				uniform = uniform && value == first
				history[1] = joinCellValue(history, 1, value)
			}
			want := unknownValue
			if uniform {
				want = first
			}
			if history[1] != want {
				t.Fatalf("sequence size=%d code=%d joined=%v want %v", size, encoded, history[1], want)
			}
		}
	}
}
