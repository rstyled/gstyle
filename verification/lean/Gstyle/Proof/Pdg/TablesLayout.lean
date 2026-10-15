module
public import Gstyle.Proof.PdgPackage
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI
namespace github_com.rstyled.gstyle.domain.pdg
namespace Tables
section def_

variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi] [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics]
variable [table_sem : Tables.TypeAssumptions]

instance Tables_typed_pointsto :
    TypedPointsto (GF := GF) github_com.rstyled.gstyle.domain.pdg.Tables where
  typedPointstoDef l v dq := iprop(
    "Sources" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"Sources" l) v.Sources' dq ∗
    "MatchLabels" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"MatchLabels" l) v.MatchLabels' dq ∗
    "Strings" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"Strings" l) v.Strings' dq ∗
    "Policies" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"Policies" l) v.Policies' dq ∗
    "Evidence" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"Evidence" l) v.Evidence' dq ∗
    "Provenance" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"Provenance" l) v.Provenance' dq ∗
    "_" ∷ True)
  typedPointstoDef_dfractional := by solve_typed_pointsto_dfractional
  typedPointstoDef_timeless := by solve_typed_pointsto_timeless
  typedPointsto_agree := by solve_typed_pointsto_agree

instance Tables_into_val_typed :
    IntoValTypedUnderlying (GF := GF) github_com.rstyled.gstyle.domain.pdg.Tables github_com.rstyled.gstyle.domain.pdg.Tables.underlying := by
  solve_into_val_typed_struct

instance Tables_access_load_Sources (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Tables) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"Sources" l) v.Sources' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"Sources" l) v.Sources' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Tables_access_store_Sources (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Tables) (Sources' : Loc) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"Sources" l) v.Sources' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"Sources" l) Sources' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with Sources' := Sources' } : github_com.rstyled.gstyle.domain.pdg.Tables) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance Tables_access_load_MatchLabels (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Tables) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"MatchLabels" l) v.MatchLabels' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"MatchLabels" l) v.MatchLabels' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Tables_access_store_MatchLabels (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Tables) (MatchLabels' : GoSlice) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"MatchLabels" l) v.MatchLabels' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"MatchLabels" l) MatchLabels' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with MatchLabels' := MatchLabels' } : github_com.rstyled.gstyle.domain.pdg.Tables) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance Tables_access_load_Strings (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Tables) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"Strings" l) v.Strings' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"Strings" l) v.Strings' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Tables_access_store_Strings (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Tables) (Strings' : GoSlice) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"Strings" l) v.Strings' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"Strings" l) Strings' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with Strings' := Strings' } : github_com.rstyled.gstyle.domain.pdg.Tables) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance Tables_access_load_Policies (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Tables) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"Policies" l) v.Policies' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"Policies" l) v.Policies' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Tables_access_store_Policies (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Tables) (Policies' : GoSlice) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"Policies" l) v.Policies' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"Policies" l) Policies' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with Policies' := Policies' } : github_com.rstyled.gstyle.domain.pdg.Tables) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance Tables_access_load_Evidence (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Tables) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"Evidence" l) v.Evidence' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"Evidence" l) v.Evidence' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Tables_access_store_Evidence (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Tables) (Evidence' : GoSlice) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"Evidence" l) v.Evidence' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"Evidence" l) Evidence' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with Evidence' := Evidence' } : github_com.rstyled.gstyle.domain.pdg.Tables) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance Tables_access_load_Provenance (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Tables) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"Provenance" l) v.Provenance' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"Provenance" l) v.Provenance' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance Tables_access_store_Provenance (l : Loc) (v : github_com.rstyled.gstyle.domain.pdg.Tables) (Provenance' : GoSlice) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"Provenance" l) v.Provenance' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.pdg.Tables go!"Provenance" l) Provenance' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with Provenance' := Provenance' } : github_com.rstyled.gstyle.domain.pdg.Tables) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

end def_
end Tables
end github_com.rstyled.gstyle.domain.pdg
end Perennial
