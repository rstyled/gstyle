package quality

import "slices"

// ContainerMutation identifies standard calls whose only write target is their
// first argument and which cannot invoke user callbacks or interface methods.
func ContainerMutation(callee string) bool {
	switch NormalizeCallee(callee) {
	case "maps.Copy", "slices.Sort", "slices.Compact", "slices.Reverse",
		"slices.Delete", "slices.Insert", "slices.Replace",
		"sort.Ints", "sort.Float64s", "sort.Strings",
		"encoding/binary.littleEndian.PutUint16", "encoding/binary.littleEndian.PutUint32", "encoding/binary.littleEndian.PutUint64",
		"encoding/binary.littleEndian.AppendUint16", "encoding/binary.littleEndian.AppendUint32", "encoding/binary.littleEndian.AppendUint64",
		"encoding/binary.bigEndian.PutUint16", "encoding/binary.bigEndian.PutUint32", "encoding/binary.bigEndian.PutUint64",
		"encoding/binary.bigEndian.AppendUint16", "encoding/binary.bigEndian.AppendUint32", "encoding/binary.bigEndian.AppendUint64":
		return true
	default:
		return false
	}
}

// ContainerClone identifies shallow copies with fresh container storage and no
// user callbacks. Element references in the copy can still be shared.
func ContainerClone(callee string) bool {
	name := NormalizeCallee(callee)
	return name == "slices.Clone" || name == "maps.Clone"
}

// HasLocalContainerTarget proves only that this modeled call writes local
// container storage. It does not override a configured effect classification.
func (call Call) HasLocalContainerTarget() bool {
	return call.MutationTarget == Local && !call.Dynamic && ContainerMutation(call.Callee)
}

func (call Call) validMutationTarget() bool {
	if call.MutationTarget == "" {
		return true
	}
	return !call.Dynamic && ContainerMutation(call.Callee) && slices.Contains([]Provenance{Local, External, Unknown}, call.MutationTarget)
}
