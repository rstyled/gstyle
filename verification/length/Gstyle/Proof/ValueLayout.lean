module
public import Gstyle.Code.github_com.rstyled.gstyle.domain.runtimeguard
public import Perennial.Proof.ProofPrelude
public import Perennial.Golang.Theory
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI
namespace github_com.rstyled.gstyle.domain.runtimeguard
namespace Value
section def_

variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi] [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics]
variable [value_sem : Value.TypeAssumptions]

instance Value_typed_pointsto :
    TypedPointsto (GF := GF) github_com.rstyled.gstyle.domain.runtimeguard.Value where
  typedPointstoDef l v dq := iprop(
    "Identity" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Identity" l) v.Identity' dq ∗
    "Lower" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Lower" l) v.Lower' dq ∗
    "Upper" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Upper" l) v.Upper' dq ∗
    "HasLower" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"HasLower" l) v.HasLower' dq ∗
    "HasUpper" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"HasUpper" l) v.HasUpper' dq ∗
    "NonNil" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"NonNil" l) v.NonNil' dq ∗
    "Nil" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Nil" l) v.Nil' dq ∗
    "Guarded" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Guarded" l) v.Guarded' dq ∗
    "UpperChecked" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"UpperChecked" l) v.UpperChecked' dq ∗
    "Integer" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Integer" l) v.Integer' dq ∗
    "Length" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Length" l) v.Length' dq ∗
    "BelowLength" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"BelowLength" l) v.BelowLength' dq ∗
    "Extent" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Extent" l) v.Extent' dq ∗
    "HasExtent" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"HasExtent" l) v.HasExtent' dq ∗
    "_" ∷ True)
  typedPointstoDef_dfractional := by solve_typed_pointsto_dfractional
  typedPointstoDef_timeless := by solve_typed_pointsto_timeless
  typedPointsto_agree := by solve_typed_pointsto_agree

instance Value_into_val_typed :
    IntoValTypedUnderlying (GF := GF) github_com.rstyled.gstyle.domain.runtimeguard.Value github_com.rstyled.gstyle.domain.runtimeguard.Value.underlying := by
  solve_into_val_typed_struct

instance Value_access_load_Identity (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Identity" l) v.Identity' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Identity" l) v.Identity' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Value_access_store_Identity (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (Identity' : w64) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Identity" l) v.Identity' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Identity" l) Identity' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with Identity' := Identity' } : github_com.rstyled.gstyle.domain.runtimeguard.Value) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance Value_access_load_Lower (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Lower" l) v.Lower' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Lower" l) v.Lower' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Value_access_store_Lower (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (Lower' : w64) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Lower" l) v.Lower' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Lower" l) Lower' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with Lower' := Lower' } : github_com.rstyled.gstyle.domain.runtimeguard.Value) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance Value_access_load_Upper (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Upper" l) v.Upper' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Upper" l) v.Upper' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Value_access_store_Upper (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (Upper' : w64) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Upper" l) v.Upper' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Upper" l) Upper' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with Upper' := Upper' } : github_com.rstyled.gstyle.domain.runtimeguard.Value) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance Value_access_load_HasLower (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"HasLower" l) v.HasLower' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"HasLower" l) v.HasLower' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Value_access_store_HasLower (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (HasLower' : Bool) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"HasLower" l) v.HasLower' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"HasLower" l) HasLower' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with HasLower' := HasLower' } : github_com.rstyled.gstyle.domain.runtimeguard.Value) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance Value_access_load_HasUpper (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"HasUpper" l) v.HasUpper' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"HasUpper" l) v.HasUpper' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Value_access_store_HasUpper (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (HasUpper' : Bool) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"HasUpper" l) v.HasUpper' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"HasUpper" l) HasUpper' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with HasUpper' := HasUpper' } : github_com.rstyled.gstyle.domain.runtimeguard.Value) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance Value_access_load_NonNil (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"NonNil" l) v.NonNil' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"NonNil" l) v.NonNil' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Value_access_store_NonNil (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (NonNil' : Bool) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"NonNil" l) v.NonNil' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"NonNil" l) NonNil' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with NonNil' := NonNil' } : github_com.rstyled.gstyle.domain.runtimeguard.Value) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance Value_access_load_Nil (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Nil" l) v.Nil' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Nil" l) v.Nil' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Value_access_store_Nil (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (Nil' : Bool) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Nil" l) v.Nil' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Nil" l) Nil' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with Nil' := Nil' } : github_com.rstyled.gstyle.domain.runtimeguard.Value) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance Value_access_load_Guarded (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Guarded" l) v.Guarded' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Guarded" l) v.Guarded' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Value_access_store_Guarded (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (Guarded' : Bool) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Guarded" l) v.Guarded' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Guarded" l) Guarded' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with Guarded' := Guarded' } : github_com.rstyled.gstyle.domain.runtimeguard.Value) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance Value_access_load_UpperChecked (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"UpperChecked" l) v.UpperChecked' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"UpperChecked" l) v.UpperChecked' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Value_access_store_UpperChecked (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (UpperChecked' : Bool) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"UpperChecked" l) v.UpperChecked' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"UpperChecked" l) UpperChecked' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with UpperChecked' := UpperChecked' } : github_com.rstyled.gstyle.domain.runtimeguard.Value) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance Value_access_load_Integer (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Integer" l) v.Integer' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Integer" l) v.Integer' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Value_access_store_Integer (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (Integer' : Bool) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Integer" l) v.Integer' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Integer" l) Integer' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with Integer' := Integer' } : github_com.rstyled.gstyle.domain.runtimeguard.Value) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance Value_access_load_Length (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Length" l) v.Length' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Length" l) v.Length' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Value_access_store_Length (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (Length' : Bool) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Length" l) v.Length' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Length" l) Length' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with Length' := Length' } : github_com.rstyled.gstyle.domain.runtimeguard.Value) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance Value_access_load_BelowLength (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"BelowLength" l) v.BelowLength' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"BelowLength" l) v.BelowLength' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Value_access_store_BelowLength (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (BelowLength' : Bool) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"BelowLength" l) v.BelowLength' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"BelowLength" l) BelowLength' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with BelowLength' := BelowLength' } : github_com.rstyled.gstyle.domain.runtimeguard.Value) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance Value_access_load_Extent (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Extent" l) v.Extent' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Extent" l) v.Extent' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Value_access_store_Extent (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (Extent' : w64) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Extent" l) v.Extent' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"Extent" l) Extent' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with Extent' := Extent' } : github_com.rstyled.gstyle.domain.runtimeguard.Value) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance Value_access_load_HasExtent (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"HasExtent" l) v.HasExtent' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"HasExtent" l) v.HasExtent' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Value_access_store_HasExtent (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.Value) (HasExtent' : Bool) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"HasExtent" l) v.HasExtent' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.Value go!"HasExtent" l) HasExtent' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with HasExtent' := HasExtent' } : github_com.rstyled.gstyle.domain.runtimeguard.Value) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

end def_
end Value
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
