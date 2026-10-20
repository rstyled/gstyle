package quality

func (call Call) HasPlainJSON() bool {
	if call.Dynamic {
		return false
	}
	switch NormalizeCallee(call.Callee) {
	case "encoding/json.Marshal", "encoding/json.MarshalIndent":
		return call.storage(0).PlainJSON
	case "encoding/json.Unmarshal":
		return call.LocalJSONDecode
	}
	return false
}

func (call Call) policyKind(prefixes []Prefix) (Kind, bool) {
	prefix, configured := selectedPrefix(NormalizeCallee(call.Callee), prefixes)
	if prefix.refinesMemoryRead(call) {
		return None, true
	}
	if prefix.refinesMemoryWriter(call) {
		return MutationEffect, true
	}
	if call.HasPlainJSON() && prefix.plainJSON && prefix.Kind == UnknownEffect {
		return None, true
	}
	return prefix.Kind, configured
}

func (call Call) validJSONDecode() bool {
	return !call.LocalJSONDecode || (!call.Dynamic && NormalizeCallee(call.Callee) == "encoding/json.Unmarshal")
}
