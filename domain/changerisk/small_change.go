package changerisk

import (
	"fmt"
	"slices"
)

func smallChangeReasons(reasons []Reason, changed, maximum int) []Reason {
	if maximum <= 0 || changed > maximum {
		return reasons
	}
	result := slices.Clone(reasons)
	for index, reason := range result {
		if reason.Rule == "pdg_change" && reason.Level == Medium {
			result[index].Level = Low
			result[index].Message = fmt.Sprintf("PDG change is within the low-risk limit of %d affected functions", maximum)
		}
	}
	return result
}
