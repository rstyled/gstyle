package cpp

import (
	"github.com/rstyled/gstyle/domain/quality"
)

func (p *projection) parameterIndex(declaration *vertex) *int {
	index := 0
	for _, input := range p.inputs {
		if receiverParameter(input) {
			if input == declaration {
				return new(-1)
			}
			continue
		}
		if input == declaration {
			return new(index)
		}
		index++
	}
	return nil
}

func (p *projection) qualityArguments(call *quality.Call, node *vertex) {
	args, storageArgs, receiver := call.Args, call.StorageArgs, call.Receiver
	for _, child := range node.Children {
		index := child.number("ARGUMENT_INDEX")
		if index < 0 {
			continue
		}
		arg, storage := p.qualityArgument(child)
		if index == 0 {
			receiver = &storage
			continue
		}
		for len(args) < index {
			args = append(args, quality.ArgSource{Kind: quality.ArgOther})
			storageArgs = append(storageArgs, quality.Storage{Provenance: quality.Unknown})
		}
		args[index-1], storageArgs[index-1] = arg, storage
	}
	call.Args, call.StorageArgs, call.Receiver = args, storageArgs, receiver
}

func (p *projection) qualityArgument(node *vertex) (quality.ArgSource, quality.Storage) {
	arg := quality.ArgSource{Kind: quality.ArgOther}
	storage := quality.Storage{Provenance: quality.Unknown}
	if node.Label == "LITERAL" {
		return quality.ArgSource{Kind: quality.ArgConst}, storage
	}
	root := storageRoot(node)
	if root == nil || len(root.Out["REF"]) != 1 {
		return arg, storage
	}
	storage.Parameter = p.parameterIndex(root.Out["REF"][0])
	if storage.Parameter != nil {
		arg.Kind, arg.Param = quality.ArgParam, *storage.Parameter
		storage.Provenance = quality.External
	}
	if fieldOperation(node) {
		arg.Kind = quality.ArgField
	}
	return arg, storage
}
