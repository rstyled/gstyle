package callflow

type captureEffects [5]string

func completedModes() [5]Outcomes {
	return [5]Outcomes{Returns, RecoversInherited, Panics, PanicDuringGoexit, ExitsGoroutine}
}

func completionIndex(mode Outcomes) int {
	for i, candidate := range completedModes() {
		if mode == candidate {
			return i
		}
	}
	return -1
}

func (pass exitPass) captureEffects(old, next captureEffects) captureEffects {
	for i := range old {
		old[i] = pass.returned(old[i], next[i])
	}
	return old
}
