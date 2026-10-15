module
public import Gstyle.Proof.PdgPackage
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI
namespace github_com.rstyled.gstyle.domain.pdg
namespace Fact
section def_

variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi] [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics]
variable [fact_sem : Fact.TypeAssumptions] [query.Assumptions]

instance Fact_typed_pointsto :
    TypedPointsto (GF := GF) github_com.rstyled.gstyle.domain.pdg.Fact where
  typedPointstoDef l v dq := iprop(
    "Value" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Fact go!"Value" l) v.Value' dq ∗
    "Status" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Fact go!"Status" l) v.Status' dq ∗
    "_" ∷ True)
  typedPointstoDef_dfractional := by solve_typed_pointsto_dfractional
  typedPointstoDef_timeless := by solve_typed_pointsto_timeless
  typedPointsto_agree := by solve_typed_pointsto_agree

instance Fact_into_val_typed :
    IntoValTypedUnderlying (GF := GF) github_com.rstyled.gstyle.domain.pdg.Fact github_com.rstyled.gstyle.domain.pdg.Fact.underlying := by
  solve_into_val_typed_struct

instance Fact_access_load_Value (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Fact) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Fact go!"Value" l) v.Value' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Fact go!"Value" l) v.Value' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Fact_access_store_Value (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Fact) (Value' : github_com.rstyled.gstyle.domain.pdg.query.Text) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Fact go!"Value" l) v.Value' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Fact go!"Value" l) Value' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with Value' := Value' } : github_com.rstyled.gstyle.domain.pdg.Fact) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance Fact_access_load_Status (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Fact) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Fact go!"Status" l) v.Status' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Fact go!"Status" l) v.Status' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Fact_access_store_Status (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Fact) (Status' : github_com.rstyled.gstyle.domain.pdg.query.Status) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Fact go!"Status" l) v.Status' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Fact go!"Status" l) Status' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with Status' := Status' } : github_com.rstyled.gstyle.domain.pdg.Fact) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

end def_
end Fact
end github_com.rstyled.gstyle.domain.pdg
end Perennial
