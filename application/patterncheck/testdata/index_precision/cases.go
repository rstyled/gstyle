package precision

func IndexNegativeBad() int  { i := -1; return []int{7, 8}[i] }
func IndexNegativeSafe() int { i := 1; return []int{7, 8}[i] }
func IndexHighBad() int      { i := 2; return []int{7, 8}[i] }
func IndexHighSafe() int     { i := 1; return []int{7, 8}[i] }
func IndexBranchBad(i int) int {
	if i >= 2 {
		return []int{7, 8}[i]
	}
	return 0
}
func IndexBranchSafe(i int) int {
	if i < 0 || i >= 2 {
		return 0
	}
	return []int{7, 8}[i]
}
func IndexSliceBad() int  { s := []int{7, 8}; i := len(s); return s[i] }
func IndexSliceSafe() int { s := []int{7, 8}; i := len(s) - 1; return s[i] }

func IndexConstantBad() int   { s := []int{7, 8}; return s[2] }
func IndexConstantSafe() int  { s := []int{7, 8}; return s[1] }
func IndexEmptyBad(i int) int { s := []int{}; return s[i] }
func IndexEmptySafe(i int) int {
	s := []int{}
	if len(s) == 0 {
		return 0
	}
	return s[i]
}

func IndexEmptyNegatedSafe(i int) int {
	s := []int{}
	if !(len(s) != 0) {
		return 0
	}
	return s[i]
}
func IndexEmptyConjunctionSafe(i int) int {
	s := []int{}
	if len(s) > 0 && i >= 0 {
		return s[i]
	}
	return 0
}
func IndexEmptyDisjunctionSafe(i int) int {
	s := []int{}
	if len(s) == 0 || i < 0 {
		return 0
	}
	return s[i]
}

func IndexGenericBad[T ~int]() int  { var i T = 2; return []int{7, 8}[i] }
func IndexGenericSafe[T ~int]() int { var i T = 1; return []int{7, 8}[i] }
