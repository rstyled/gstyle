module
public import Gstyle.Proof.Transfer.ComparisonModel
public import Gstyle.Proof.Transfer.Numeric
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [graphView.TypeAssumptions] [pdg.query.Assumptions]
variable [pdg.Graph.TypeAssumptions] [pdg.Node.TypeAssumptions] [pdg.Attributes.TypeAssumptions]
variable [pdg.AttributeSet.TypeAssumptions] [pdg.AttributeTables.TypeAssumptions]
variable [pdg.Producer.TypeAssumptions] [pdg.Function.TypeAssumptions] [pdg.Span.TypeAssumptions]
variable [pdg.CountFact.TypeAssumptions] [pdg.Conformance.TypeAssumptions] [pdg.Fact.TypeAssumptions]
variable [pdg.Position.TypeAssumptions] [pdg.CallForm.TypeAssumptions] [pdg.ExitKind.TypeAssumptions]
variable [pdg.ConformanceStatus.TypeAssumptions] [pdg.Category.TypeAssumptions]
variable [FuncUnfold pdg.listAt [pdg.Ref.ty] (pdg.listAt.impl pdg.Ref.ty)]
variable [FuncUnfold pdg.listAt [pdg.Access.ty] (pdg.listAt.impl pdg.Access.ty)]
variable [FuncUnfold pdg.listAt [pdg.Extension.ty] (pdg.listAt.impl pdg.Extension.ty)]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

variable [pdg.Access.TypeAssumptions] [pdg.Tables.TypeAssumptions] [pdg.Extension.TypeAssumptions]
variable [FuncUnfold directBindingRead [] directBindingRead.impl]
variable [FuncUnfold operatorText [] operatorText.impl]


variable [FuncUnfold extension [] extension.impl]

variable [Value.TypeAssumptions] [State.TypeAssumptions]

variable [FuncUnfold integerConstraint [] integerConstraint.impl]
variable [FuncUnfold lengthConstraint [] lengthConstraint.impl]
variable [FuncUnfold exactInteger [] exactInteger.impl]
variable [FuncUnfold emptyRange [] emptyRange.impl]
variable [interval.Assumptions]

theorem wp_constrain_snapshot_numeric (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (key : w32) (kind op : GoString) (changing active : Bool) (bound : Value)
    (hk : 0 < uint.Z key ∧ uint.Z key ≤ s.kinds.length)
    (hkind : s.kinds[(uint.Z key - 1).toNat]? = some kind)
    (hu : s.unstable.length = s.kinds.length)
    (ht : s.tracked.length = s.kinds.length)
    (hunstable : s.unstable[(uint.Z key - 1).toNat]? = some changing)
    (htracked : s.tracked[(uint.Z key - 1).toNat]? = some active)
    (hnil : bound.Nil' = false) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
       (querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"constrain" #receiver)) (Val #key)) (Val #op)) (Val #bound)) (Val #state))
    {{ RET #(if active && constraintRejected kind changing key m op bound then map.nil else state);
       querySnapshotOwn receiver s ∗ state ↦$ (constraintState active kind changing key m op bound) }} := by
  wp_start_folded as ⟨#Hsnapshot, Hstate⟩
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, _, _, _, Hkinds, Hunstable, Htracked, _, _, _, _, _, _, _, _, _, _⟩
  wp_apply wp_constrain_numeric receiver s.view s.kinds s.unstable s.tracked state m key kind op changing active bound
    hk hkind hu ht hunstable htracked hnil DFrac.discard
    $$ [Hview Hkinds Hunstable Htracked Hstate] with ⟨Hview, Hkinds, Hunstable, Htracked, Hstate⟩
  · iframe Hview Hkinds Hunstable Htracked Hstate
  wp_end
  all_goals iframe Hsnapshot Hstate

theorem wp_constrain_snapshot_invalid (receiver : Loc) (s : QuerySnapshot) (state : Loc)
    (key : w32) (op : GoString) (bound : Value)
    (hkey : key = W32 0 ∨ s.tracked.length < uint.Z key) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ querySnapshotOwn receiver s }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"constrain" #receiver)) (Val #key)) (Val #op)) (Val #bound)) (Val #state))
    {{ RET #state; querySnapshotOwn receiver s }} := by
  wp_start as #Hsnapshot
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, _, _, _, _, _, Htracked, _, _, _, _, _, _, _, _, _, _⟩
  try wp_auto
  wp_apply wp_trackedBinding_invalid receiver s.view s.tracked key DFrac.discard hkey
    $$ [Hview Htracked] with ⟨Hview, Htracked⟩
  · iframe Hview Htracked
  try wp_auto
  wp_end
  all_goals iframe Hsnapshot

theorem wp_constrain_snapshot_numeric_any (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (key : w32) (kind op : GoString) (changing active : Bool) (bound : Value)
    (hu : s.unstable.length = s.kinds.length) (ht : s.tracked.length = s.kinds.length)
    (metadata : ConstraintMetadata s key kind changing active) (hnil : bound.Nil' = false) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
       (querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"constrain" #receiver)) (Val #key)) (Val #op)) (Val #bound)) (Val #state))
    {{ RET #(if active && constraintRejected kind changing key m op bound then map.nil else state);
       querySnapshotOwn receiver s ∗ state ↦$ (constraintState active kind changing key m op bound) }} := by
  cases metadata with
  | present hk hkind hunstable htracked =>
    exact wp_constrain_snapshot_numeric receiver s state m key kind op changing active bound hk hkind hu ht hunstable htracked hnil
  | absent hout ha =>
    subst active
    simp only [Bool.false_and, Bool.false_eq_true, ite_false, constraintState]
    have htrackedOut : key = W32 0 ∨ s.tracked.length < uint.Z key := by
      rw [ht]
      try exact hout
    wp_start_folded as ⟨Hsnapshot, Hstate⟩
    wp_apply wp_constrain_snapshot_invalid receiver s state key op bound htrackedOut
      $$ [Hsnapshot] with Hsnapshot
    · iexact Hsnapshot
    wp_end

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
