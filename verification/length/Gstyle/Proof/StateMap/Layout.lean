module
public import Gstyle.Proof.RuntimeValueModel
public import Gstyle.GeneratedProof.github_com.rstyled.gstyle.domain.pdg.query
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI
namespace github_com.rstyled.gstyle.domain.runtimeguard
namespace graphView
section def_

variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi] [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics]
variable [view_sem : graphView.TypeAssumptions]

instance graphView_typed_pointsto :
    TypedPointsto (GF := GF) github_com.rstyled.gstyle.domain.runtimeguard.graphView where
  typedPointstoDef l v dq := iprop(
    "graph" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"graph" l) v.graph' dq ∗
    "operands" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"operands" l) v.operands' dq ∗
    "unstable" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"unstable" l) v.unstable' dq ∗
    "kinds" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"kinds" l) v.kinds' dq ∗
    "tracked" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"tracked" l) v.tracked' dq ∗
    "fields" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"fields" l) v.fields' dq ∗
    "paths" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"paths" l) v.paths' dq ∗
    "fieldRoots" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"fieldRoots" l) v.fieldRoots' dq ∗
    "dereferences" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"dereferences" l) v.dereferences' dq ∗
    "_" ∷ True)
  typedPointstoDef_dfractional := by solve_typed_pointsto_dfractional
  typedPointstoDef_timeless := by solve_typed_pointsto_timeless
  typedPointsto_agree := by solve_typed_pointsto_agree

instance graphView_into_val_typed :
    IntoValTypedUnderlying (GF := GF) github_com.rstyled.gstyle.domain.runtimeguard.graphView github_com.rstyled.gstyle.domain.runtimeguard.graphView.underlying := by
  solve_into_val_typed_struct

instance graphView_access_load_graph (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"graph" l) v.graph' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"graph" l) v.graph' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance graphView_access_store_graph (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (graph' : Loc) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"graph" l) v.graph' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"graph" l) graph' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with graph' := graph' } : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance graphView_access_load_operands (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"operands" l) v.operands' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"operands" l) v.operands' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance graphView_access_store_operands (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (operands' : GoSlice) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"operands" l) v.operands' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"operands" l) operands' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with operands' := operands' } : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance graphView_access_load_unstable (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"unstable" l) v.unstable' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"unstable" l) v.unstable' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance graphView_access_store_unstable (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (unstable' : GoSlice) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"unstable" l) v.unstable' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"unstable" l) unstable' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with unstable' := unstable' } : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance graphView_access_load_kinds (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"kinds" l) v.kinds' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"kinds" l) v.kinds' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance graphView_access_store_kinds (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (kinds' : GoSlice) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"kinds" l) v.kinds' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"kinds" l) kinds' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with kinds' := kinds' } : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance graphView_access_load_tracked (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"tracked" l) v.tracked' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"tracked" l) v.tracked' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance graphView_access_store_tracked (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (tracked' : GoSlice) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"tracked" l) v.tracked' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"tracked" l) tracked' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with tracked' := tracked' } : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance graphView_access_load_fields (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"fields" l) v.fields' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"fields" l) v.fields' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance graphView_access_store_fields (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (fields' : GoSlice) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"fields" l) v.fields' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"fields" l) fields' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with fields' := fields' } : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance graphView_access_load_paths (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"paths" l) v.paths' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"paths" l) v.paths' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance graphView_access_store_paths (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (paths' : GoMap) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"paths" l) v.paths' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"paths" l) paths' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with paths' := paths' } : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance graphView_access_load_fieldRoots (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"fieldRoots" l) v.fieldRoots' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"fieldRoots" l) v.fieldRoots' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance graphView_access_store_fieldRoots (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (fieldRoots' : GoMap) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"fieldRoots" l) v.fieldRoots' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"fieldRoots" l) fieldRoots' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with fieldRoots' := fieldRoots' } : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance graphView_access_load_dereferences (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"dereferences" l) v.dereferences' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"dereferences" l) v.dereferences' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance graphView_access_store_dereferences (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (dereferences' : GoMap) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"dereferences" l) v.dereferences' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.graphView go!"dereferences" l) dereferences' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with dereferences' := dereferences' } : github_com.rstyled.gstyle.domain.runtimeguard.graphView) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

end def_
end graphView
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
