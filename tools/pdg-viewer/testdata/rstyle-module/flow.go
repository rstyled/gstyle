package flow

func Forward(x int) int {
	if x < 100 {
		y := 2 * x
		y++
		y++
		return y
	}
	return 0
}

func Other(x int) int { return Forward(x) }
