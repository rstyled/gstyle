package gopatterns

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"

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
	return apply(spec, src)
}

// parseSpec parses src and returns the file set and AST.
func parseSpec(spec *patterns.FixSpec, src []byte) (*token.FileSet, *ast.File, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, spec.File, src, parser.ParseComments)
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", spec.File, err)
	}
	return fset, f, nil
}
