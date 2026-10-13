package gopatterns

import (
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"

	"github.com/rstyled/gstyle/domain/patterns"
)

// fixApplier is a function that applies a FixSpec.
type fixApplier func(*patterns.FixSpec, []byte) ([]byte, error)

// appliers maps each fixable kind to its applier.
var appliers = map[patterns.CandidateKind]fixApplier{
	patterns.DeclarationOrder:   applyDeclarationOrderFix,
	patterns.GuardClause:        applyGuardFix,
	patterns.OffsetFallback:     applyOffsetFallbackFix,
	patterns.Parameterize:       applyParameterizeFix,
	patterns.TraitMethod:        applyInterfaceFix,
	patterns.CapabilitySet:      applyInterfaceFix,
	patterns.EnumDispatch:       applyEnumDispatchFix,
	patterns.GenericFn:          applyGenericFnFix,
	patterns.PrimitiveObsession: applyPrimitiveObsessionFix,
	patterns.TypeSwitch:         applyTypeSwitchFix,
	patterns.EntityIdentity:     applyEntityIdentityFix,
	patterns.Factory:            applyFactoryFix,
	patterns.Specification:      applySpecificationFix,
	// MutableIdentity is detection-only: no safe mechanical fix exists.
	// Aggregate is detection-only: choosing the root is design.
	// Repository is detection-only: extraction is design.
	// MissingIdentity is detection-only first gate: entity_identity fixes.
}

// ApplyFix applies the transform for a candidate's FixSpec to src.
// This is a single-file code generator. Signature fixes use PlanSignatureFix
// so typed callers cannot be dropped. The IO boundary validates every result.
func ApplyFix(spec *patterns.FixSpec, src []byte) ([]byte, error) {
	if spec == nil {
		return nil, fmt.Errorf("nil FixSpec")
	}
	apply, ok := appliers[spec.Kind]
	if !ok {
		return nil, fmt.Errorf("no fixer for kind %q", spec.Kind)
	}
	if spec.Context == nil {
		return nil, fmt.Errorf("fix requires extraction evidence")
	}
	path := fixSourcePath(spec.Context.Root, spec.File)
	if err := spec.Context.Check(path, src); err != nil {
		return nil, err
	}
	return apply(spec, src)
}

// parseSpec reconstructs a disposable rendering tree from this extraction's
// shared source syntax. It never parses input bytes or reads source files.
func parseSpec(spec *patterns.FixSpec, src []byte) (*token.FileSet, *ast.File, error) {
	if spec.Context == nil {
		return nil, nil, fmt.Errorf("fix requires extraction evidence")
	}
	path := fixSourcePath(spec.Context.Root, spec.File)
	if err := spec.Context.Check(path, src); err != nil {
		return nil, nil, err
	}
	return sourceSyntaxTree(path, spec.Context)
}

func fixSourcePath(root, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(root, path)
}
