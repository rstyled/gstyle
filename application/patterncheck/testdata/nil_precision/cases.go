package precision

func NilAssignedBad() int  { p := new(int); p = nil; return *p }
func NilAssignedSafe() int { p := new(int); *p = 7; return *p }
func NilGuardBad(p *int) int {
	if p != nil {
		return 0
	}
	return *p
}
func NilGuardSafe(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
func NilAliasBad(p *int) int {
	q := p
	if p != nil {
		return 0
	}
	return *q
}
func NilAliasSafe(p *int) int {
	q := p
	if p == nil {
		return 0
	}
	return *q
}
func NilBranchBad(p *int) int {
	if p == nil {
		return *p
	}
	return 0
}
func NilBranchSafe(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
