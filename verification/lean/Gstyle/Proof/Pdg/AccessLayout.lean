module
public import Gstyle.Proof.PdgPackage
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI
namespace github_com.rstyled.gstyle.domain.pdg
namespace Access
section def_

variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi] [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics]
variable [Access.TypeAssumptions] [Position.TypeAssumptions] [query.Assumptions]

instance Access_typed_pointsto :
    TypedPointsto (GF := GF) github_com.rstyled.gstyle.domain.pdg.Access where
  typedPointstoDef l v dq := iprop(
    "Symbol" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Access go!"Symbol" l) v.Symbol' dq ∗
    "Location" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Access go!"Location" l) v.Location' dq ∗
    "UnresolvedReason" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Access go!"UnresolvedReason" l) v.UnresolvedReason' dq ∗
    "Position" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Access go!"Position" l) v.Position' dq ∗
    "_" ∷ True)
  typedPointstoDef_dfractional := by solve_typed_pointsto_dfractional
  typedPointstoDef_timeless := by solve_typed_pointsto_timeless
  typedPointsto_agree := by solve_typed_pointsto_agree

instance Access_into_val_typed :
    IntoValTypedUnderlying (GF := GF) github_com.rstyled.gstyle.domain.pdg.Access github_com.rstyled.gstyle.domain.pdg.Access.underlying := by
  solve_into_val_typed_struct

instance Access_access_load_Symbol (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Access) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Access go!"Symbol" l) v.Symbol' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Access go!"Symbol" l) v.Symbol' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) := by
  solve_pointsto_access_struct

instance Access_access_load_Location (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Access) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Access go!"Location" l) v.Location' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Access go!"Location" l) v.Location' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) := by
  solve_pointsto_access_struct

end def_
end Access
end github_com.rstyled.gstyle.domain.pdg
end Perennial
