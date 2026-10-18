package callflow

// These narrow contracts exclude explicit frame inspection and user callbacks.
// They do not assert termination, allocation success or absence of implicit faults.
// Other external calls remain unknown. Reviewed Go source is recorded in docs.
func externalFrameEffect(name string) FrameEffect {
	if observesFrame(name) {
		return FrameEffect{Observes: true}
	}
	if neutralFrameContract(name) {
		return FrameEffect{}
	}
	return FrameEffect{Unknown: true}
}
func observesFrame(name string) bool {
	switch name {
	case "recover", "panic", "runtime.Caller", "runtime.Callers", "runtime.Stack", "runtime/debug.Stack", "runtime/debug.PrintStack":
		return true
	default:
		return false
	}
}
func neutralFrameContract(name string) bool {
	switch name {
	case "append", "cap", "clear", "close", "complex", "copy", "delete", "imag", "len", "make", "max", "min", "new", "print", "println", "real":
		return true
	case "errors.New", "strings.TrimSpace", "path.Clean":
		return true
	default:
		return false
	}
}
