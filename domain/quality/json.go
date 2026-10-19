package quality

func (call Call) HasPlainJSON() bool {
	return !call.Dynamic && NormalizeCallee(call.Callee) == "encoding/json.Marshal" && call.storage(0).PlainJSON
}

func (call Call) policyKind(prefixes []Prefix) (Kind, bool) {
	prefix, configured := selectedPrefix(NormalizeCallee(call.Callee), prefixes)
	if call.HasPlainJSON() && prefix.plainJSON && prefix.Kind == UnknownEffect {
		return None, true
	}
	return prefix.Kind, configured
}
