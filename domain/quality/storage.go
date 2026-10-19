package quality

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Storage records call-site ownership and callback-free JSON type evidence.
// Parameter identifies an
// enclosing formal (-1 for its receiver); nil means no formal binding is known.
type Storage struct {
	PlainJSON  bool       `json:"plain_json,omitempty"`
	Provenance Provenance `json:"provenance"`
	Parameter  *int       `json:"parameter,omitempty"`
}

func (call Call) storage(parameter int) Storage {
	if parameter == -1 {
		if call.Receiver != nil {
			return *call.Receiver
		}
		return Storage{Provenance: Unknown}
	}
	if parameter < 0 || parameter >= len(call.StorageArgs) {
		return Storage{Provenance: Unknown}
	}
	return call.StorageArgs[parameter]
}

func (call Call) resolveStorage(storage Storage) Storage {
	if storage.Parameter == nil {
		return storage
	}
	actual := call.storage(*storage.Parameter)
	return Storage{Provenance: actual.Provenance, PlainJSON: storage.PlainJSON || actual.PlainJSON}
}

func (call Call) bindHelper(helper Helper) Helper {
	mutations := slices.Clone(helper.Mutations)
	for i, mutation := range mutations {
		if mutation.Parameter != nil {
			mutations[i].Provenance = call.storage(*mutation.Parameter).Provenance
		}
	}
	calls := slices.Clone(helper.Calls)
	for i, nested := range calls {
		calls[i] = call.bindCall(nested)
	}
	helper.Mutations = mutations
	helper.Calls = calls
	return helper
}

func (call Call) bindCall(nested Call) Call {
	arguments := slices.Clone(nested.StorageArgs)
	for i, storage := range arguments {
		arguments[i] = call.resolveStorage(storage)
	}
	nested.StorageArgs = arguments
	if nested.Receiver != nil {
		resolved := call.resolveStorage(*nested.Receiver)
		nested.Receiver = &resolved
	}
	if nested.MutationTarget != "" && len(nested.StorageArgs) > 0 {
		nested.MutationTarget = nested.StorageArgs[0].Provenance
	}
	return nested
}

// Helper results depend on ownership and typed evidence, never formal identity.
func (call Call) storageKey(name string) string {
	var key strings.Builder
	key.WriteString(name)
	key.WriteByte('|')
	key.WriteString(string(call.storage(-1).Provenance))
	key.WriteString(strconv.FormatBool(call.storage(-1).PlainJSON))
	for _, storage := range call.StorageArgs {
		key.WriteByte('|')
		key.WriteString(string(storage.Provenance))
		key.WriteString(strconv.FormatBool(storage.PlainJSON))
	}
	return key.String()
}

func validParameter(parameter *int, f Function) bool {
	if parameter == nil {
		return true
	}
	if *parameter == -1 {
		return f.HasSelf
	}
	return *parameter >= 0 && *parameter < f.Params
}

func validateStorage(f Function) error {
	for _, mutation := range f.Mutations {
		if !validMutationParameter(mutation, f) {
			return fmt.Errorf("invalid mutation parameter in %s", f.Name)
		}
	}
	for _, call := range f.Calls {
		if err := validateCallStorage(call, f); err != nil {
			return err
		}
	}
	return nil
}

func validateCallStorage(call Call, f Function) error {
	values := call.StorageArgs
	if call.Receiver != nil {
		values = append(slices.Clone(values), *call.Receiver)
	}
	for _, storage := range values {
		if !slices.Contains([]Provenance{Local, External, Unknown}, storage.Provenance) || !validStorageParameter(storage, f) {
			return fmt.Errorf("invalid call storage in %s", f.Name)
		}
	}
	return nil
}

func validMutationParameter(mutation Mutation, f Function) bool {
	if mutation.Site < 0 {
		return false
	}
	if mutation.Parameter == nil {
		return true
	}
	return mutation.Provenance == External && validParameter(mutation.Parameter, f)
}

func validStorageParameter(storage Storage, f Function) bool {
	if storage.Parameter == nil {
		return true
	}
	return storage.Provenance == External && validParameter(storage.Parameter, f)
}
