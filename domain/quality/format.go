package quality

func MemoryWriterCall(callee string) bool {
	switch NormalizeCallee(callee) {
	case "fmt.Fprint", "fmt.Fprintf", "fmt.Fprintln", "io.WriteString":
		return true
	case "fmt.Sscan", "fmt.Sscanf", "fmt.Sscanln":
		return true
	case "encoding/xml.Decoder.Decode", "encoding/json.Decoder.Decode", "encoding/json.Unmarshal":
		return true
	}
	return false
}

func (call Call) validMemoryWriter() bool {
	if call.PrivateBuilder && !call.validPrivateBuilder() {
		return false
	}
	return !call.MemoryWriter || (!call.Dynamic && MemoryWriterCall(call.Callee))
}

func (call Call) validPrivateBuilder() bool {
	return call.MemoryWriter && BuilderFormatCall(call.Callee) && len(call.Args) > 0
}

func BuilderFormatCall(callee string) bool {
	switch NormalizeCallee(callee) {
	case "fmt.Fprint", "fmt.Fprintf", "fmt.Fprintln", "io.WriteString":
		return true
	default:
		return false
	}
}

func (prefix Prefix) refinesMemoryWriter(call Call) bool {
	return prefix.memoryWriter && call.MemoryWriter && (prefix.Kind == IO || (prefix.Kind == UnknownEffect && (prefix.plainJSON || NormalizeCallee(call.Callee) == "encoding/json.Decoder.Decode")))
}

// Private builders and scalar scans have separate ownership certificates.
func (call Call) memoryWriteKind() Kind {
	if call.PrivateBuilder {
		return None
	}
	switch NormalizeCallee(call.Callee) {
	case "fmt.Sscan", "fmt.Sscanln":
		return call.scalarScanKind(1)
	case "fmt.Sscanf":
		return call.scalarScanKind(2)
	default:
		return MutationEffect
	}
}

func (call Call) scalarScanKind(offset int) Kind {
	// Legacy or partial storage facts cannot prove every destination local.
	if len(call.StorageArgs) <= offset || len(call.StorageArgs) != len(call.Args) {
		return MutationEffect
	}
	for _, destination := range call.StorageArgs[offset:] {
		if destination.Provenance != Local {
			return MutationEffect
		}
	}
	return None
}
