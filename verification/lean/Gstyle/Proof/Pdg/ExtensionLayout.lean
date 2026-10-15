module
public import Gstyle.Proof.PdgPackage
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI
namespace github_com.rstyled.gstyle.domain.pdg
namespace Extension
section def_

variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi] [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics]
variable [Extension.TypeAssumptions] [query.Assumptions]

instance Extension_typed_pointsto :
    TypedPointsto (GF := GF) github_com.rstyled.gstyle.domain.pdg.Extension where
  typedPointstoDef l v dq := iprop(
    "Name" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Extension go!"Name" l) v.Name' dq ∗
    "Value" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Extension go!"Value" l) v.Value' dq ∗
    "_" ∷ True)
  typedPointstoDef_dfractional := by solve_typed_pointsto_dfractional
  typedPointstoDef_timeless := by solve_typed_pointsto_timeless
  typedPointsto_agree := by solve_typed_pointsto_agree

instance Extension_into_val_typed :
    IntoValTypedUnderlying (GF := GF) github_com.rstyled.gstyle.domain.pdg.Extension github_com.rstyled.gstyle.domain.pdg.Extension.underlying := by
  solve_into_val_typed_struct

instance Extension_access_load_Name (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Extension) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Extension go!"Name" l) v.Name' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Extension go!"Name" l) v.Name' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Extension_access_load_Value (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Extension) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Extension go!"Value" l) v.Value' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Extension go!"Value" l) v.Value' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

end def_
end Extension
end github_com.rstyled.gstyle.domain.pdg
end Perennial
