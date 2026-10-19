package quality

func (call Call) HasPlainJSON() bool {
	if call.Dynamic {
		return false
	}
	name := NormalizeCallee(call.Callee)
	return (name == "encoding/json.Marshal" && call.storage(0).PlainJSON) || (name == "encoding/json.Unmarshal" && call.LocalJSONDecode)
}

func (call Call) policyKind(prefixes []Prefix) (Kind, bool) {
	prefix, configured := selectedPrefix(NormalizeCallee(call.Callee), prefixes)
	if call.HasPlainJSON() && prefix.plainJSON && prefix.Kind == UnknownEffect {
		return None, true
	}
	return prefix.Kind, configured
}

func (call Call) validJSONDecode() bool {
	return !call.LocalJSONDecode || (!call.Dynamic && NormalizeCallee(call.Callee) == "encoding/json.Unmarshal")
}
