package changerisk

func unknownReasons(reasons []Reason, rule string) int {
	count := 0
	for _, reason := range reasons {
		if reason.Rule == rule && reason.Level == Unknown {
			count++
		}
	}
	return count
}
