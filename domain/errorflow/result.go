package errorflow

// All combines independently queried error-result slots. An ignored slot is
// enough to disprove complete use; unknown evidence cannot disprove or prove it.
func (result Result) All(other Result) Result {
	if result.Unused() {
		return result
	}
	if other.Unused() {
		return other
	}
	return Result{Used: result.Used && other.Used, Known: result.Known && other.Known}
}
func (result Result) Unused() bool { return result.Known && !result.Used }
