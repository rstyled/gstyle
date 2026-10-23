module
public import Gstyle.Proof.RuntimePackage
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI
namespace github_com.rstyled.gstyle.domain.runtimeguard
namespace analysis
section def_

variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi] [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics]
variable [analysis.TypeAssumptions]

instance analysis_typed_pointsto :
    TypedPointsto (GF := GF) github_com.rstyled.gstyle.domain.runtimeguard.analysis where
  typedPointstoDef l v dq := iprop(
    "view" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.analysis go!"view" l) v.view' dq ∗
    "inputs" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.analysis go!"inputs" l) v.inputs' dq ∗
    "queued" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.analysis go!"queued" l) v.queued' dq ∗
    "queue" ∷ typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.analysis go!"queue" l) v.queue' dq ∗
    "_" ∷ True)
  typedPointstoDef_dfractional := by solve_typed_pointsto_dfractional
  typedPointstoDef_timeless := by solve_typed_pointsto_timeless
  typedPointsto_agree := by solve_typed_pointsto_agree

instance analysis_into_val_typed :
    IntoValTypedUnderlying (GF := GF) github_com.rstyled.gstyle.domain.runtimeguard.analysis github_com.rstyled.gstyle.domain.runtimeguard.analysis.underlying := by
  solve_into_val_typed_struct

instance analysis_access_load_view (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.analysis) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.analysis go!"view" l) v.view' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.analysis go!"view" l) v.view' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance analysis_access_store_view (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.analysis) (view' : Loc) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.analysis go!"view" l) v.view' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.analysis go!"view" l) view' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with view' := view' } : github_com.rstyled.gstyle.domain.runtimeguard.analysis) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance analysis_access_load_inputs (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.analysis) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.analysis go!"inputs" l) v.inputs' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.analysis go!"inputs" l) v.inputs' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance analysis_access_store_inputs (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.analysis) (inputs' : GoSlice) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.analysis go!"inputs" l) v.inputs' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.analysis go!"inputs" l) inputs' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with inputs' := inputs' } : github_com.rstyled.gstyle.domain.runtimeguard.analysis) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance analysis_access_load_queued (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.analysis) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.analysis go!"queued" l) v.queued' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.analysis go!"queued" l) v.queued' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance analysis_access_store_queued (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.analysis) (queued' : GoSlice) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.analysis go!"queued" l) v.queued' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.analysis go!"queued" l) queued' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with queued' := queued' } : github_com.rstyled.gstyle.domain.runtimeguard.analysis) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

instance analysis_access_load_queue (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.analysis) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.analysis go!"queue" l) v.queue' dq)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.analysis go!"queue" l) v.queue' dq)
      (typedPointsto l v dq) (typedPointsto l v dq) :=
 by
  solve_pointsto_access_struct

instance analysis_access_store_queue (l : Loc) (v : github_com.rstyled.gstyle.domain.runtimeguard.analysis) (queue' : GoSlice) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.analysis go!"queue" l) v.queue' (DFrac.own 1))
      (typedPointsto (structFieldRef github_com.rstyled.gstyle.domain.runtimeguard.analysis go!"queue" l) queue' (DFrac.own 1))
      (typedPointsto l v (DFrac.own 1)) (typedPointsto l ({ v with queue' := queue' } : github_com.rstyled.gstyle.domain.runtimeguard.analysis) (DFrac.own 1)) :=
 by
  solve_pointsto_access_struct

end def_
end analysis
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
