module
public import Gstyle.Proof.RuntimePackage
public import Gstyle.Proof.Interval
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [FfiSyntax]

def bounds (v : Value) : interval.Range :=
  { Lower' := v.Lower', Upper' := v.Upper', HasLower' := v.HasLower', HasUpper' := v.HasUpper' }

def constrainedValue (v : Value) (operator : GoString) (bound : w64) : Value :=
  let r := interval.constrained (bounds v) operator bound
  { v with Lower' := r.Lower', Upper' := r.Upper', HasLower' := r.HasLower', HasUpper' := r.HasUpper' }

def mergedBounds (v other : Value) : Value :=
  let r := interval.joined (bounds v) (bounds other)
  { Identity' := 0, Lower' := r.Lower', Upper' := r.Upper', HasLower' := r.HasLower', HasUpper' := r.HasUpper',
    NonNil' := false, Nil' := false, Guarded' := false, UpperChecked' := false, Integer' := false,
    Length' := false, BelowLength' := false, Extent' := 0, HasExtent' := false }

def mergedPointers (v other out : Value) : Value :=
  { out with
    NonNil' := Bool.and v.NonNil' other.NonNil', Nil' := Bool.and v.Nil' other.Nil',
    Guarded' := Bool.and v.Guarded' other.Guarded',
    Identity' := if v.Identity' = other.Identity' then v.Identity' else out.Identity' }

def merged (v other : Value) : Value :=
  mergedPointers v other { (mergedBounds v other) with
    UpperChecked' := Bool.or v.UpperChecked' other.UpperChecked',
    Length' := v.Length' && other.Length', BelowLength' := v.BelowLength' && other.BelowLength' }

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
