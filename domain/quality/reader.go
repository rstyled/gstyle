package quality

// MemoryReadCall identifies operations with a typed closed-memory proof.
func MemoryReadCall(callee string) bool {
	switch NormalizeCallee(callee) {
	case "io.ReadAll", "io.Closer.Close", "encoding/json.Decoder.Token", "encoding/json.Decoder.More":
		return true
	}
	return false
}

func (call Call) validLocalMemoryRead() bool {
	if !call.LocalMemoryRead {
		return true
	}
	if !MemoryReadCall(call.Callee) {
		return false
	}
	return !call.Dynamic || NormalizeCallee(call.Callee) == "io.Closer.Close"
}

func (prefix Prefix) refinesMemoryRead(call Call) bool {
	return prefix.localMemoryRead && call.LocalMemoryRead && prefix.Kind == IO
}
