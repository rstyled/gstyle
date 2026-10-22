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

// ContainerClone identifies standard calls that return fresh container storage.
// This proves storage ownership only: elements can be shared and callbacks
// can have effects of their own.
func ContainerClone(callee string) bool {
	switch NormalizeCallee(callee) {
	case "slices.Clone", "maps.Clone", "slices.Concat", "slices.Repeat", "slices.Collect",
		"strings.Split", "strings.SplitN", "strings.SplitAfter", "strings.SplitAfterN", "strings.Fields",
		"bytes.Clone", "bytes.Repeat", "bytes.Split", "bytes.SplitN", "bytes.SplitAfter", "bytes.SplitAfterN", "bytes.Fields":
		return true
	default:
		return false
	}
}

// HasLocalContainerTarget proves only that this modeled call writes local
// container storage. It does not override a configured effect classification.
func (call Call) HasLocalContainerTarget() bool {
	return call.MutationTarget == Local && !call.Dynamic && call.modeledContainerMutation()
}

func (call Call) validMutationTarget() bool {
	if call.MutationTarget == "" {
		return call.validCallbackExpansion()
	}
	return call.validCallbackExpansion() && !call.Dynamic && call.modeledContainerMutation() && slices.Contains([]Provenance{Local, External, Unknown}, call.MutationTarget)
}

// CallbackContainer identifies sorts whose sole callback is the second argument.
func CallbackContainer(callee string) bool {
	return slices.Contains([]string{"slices.SortFunc", "slices.SortStableFunc", "sort.Slice", "sort.SliceStable"}, NormalizeCallee(callee))
}

func (call Call) modeledContainerMutation() bool {
	return ContainerMutation(call.Callee) || (call.CallbackExpanded && CallbackContainer(call.Callee))
}

func (call Call) validCallbackExpansion() bool {
	return !call.CallbackExpanded || (!call.Dynamic && CallbackContainer(call.Callee))
}
