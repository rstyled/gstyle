package quality

func MemoryWriterCall(callee string) bool {
	switch NormalizeCallee(callee) {
	case "fmt.Fprint", "fmt.Fprintf", "fmt.Fprintln", "io.WriteString":
		return true
	case "fmt.Sscan", "fmt.Sscanf", "fmt.Sscanln":
		return true
	case "encoding/xml.Decoder.Decode", "encoding/json.Decoder.Decode":
		return true
	}
	return false
}

func (call Call) validMemoryWriter() bool {
	return !call.MemoryWriter || (!call.Dynamic && MemoryWriterCall(call.Callee))
}

func (prefix Prefix) refinesMemoryWriter(call Call) bool {
	return prefix.memoryWriter && call.MemoryWriter && (prefix.Kind == IO || (prefix.Kind == UnknownEffect && NormalizeCallee(call.Callee) == "encoding/json.Decoder.Decode"))
}
