module
public import Gstyle.Proof.Transfer.Tracking
public import Gstyle.Proof.Transfer.NumericPrimitives
public import Gstyle.Proof.Transfer.NumericModel
public import Gstyle.Proof.StateMap.Binding
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [FfiSyntax]

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [graphView.TypeAssumptions] [Value.TypeAssumptions] [State.TypeAssumptions] [pdg.query.Assumptions]
variable [FuncUnfold integerConstraint [] integerConstraint.impl]
variable [FuncUnfold lengthConstraint [] lengthConstraint.impl]
variable [FuncUnfold exactInteger [] exactInteger.impl]
variable [FuncUnfold emptyRange [] emptyRange.impl]
variable [interval.Assumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

def numericIntegerContract (v : Value) (operator : GoString) (bound : w64) : Prop :=
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval }}
      (App (App (App (Val (@! integerConstraint)) (Val #v)) (Val #operator)) (Val #bound))
    {{ RET #(constrainedValue v operator bound); True }}

theorem wp_constrain_numeric_query (receiver : Loc) (view : graphView)
    (kinds : List GoString) (unstable tracked : List Bool)
    (state : Loc) (m : StateData) (key : w32) (kind op : GoString) (changing active : Bool) (b : Value)
    (hk : 0 < uint.Z key ∧ uint.Z key ≤ kinds.length)
    (hkind : kinds[(uint.Z key - 1).toNat]? = some kind)
    (hu : unstable.length = kinds.length)
    (ht : tracked.length = kinds.length)
    (hunstable : unstable[(uint.Z key - 1).toNat]? = some changing)
    (htracked : tracked[(uint.Z key - 1).toNat]? = some active)
    (hnil : b.Nil' = false) (dq : DFrac)
    (hinteger : numericIntegerContract (GF := GF) (lengthConstrained (bindingValue kind changing key m) op b) op b.Lower') :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
      (receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds ∗ view.unstable' ↦*{dq} unstable ∗ view.tracked' ↦*{dq} tracked ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"constrain" #receiver)) (Val #key)) (Val #op)) (Val #b)) (Val #state))
    {{ RET #(if active && constraintRejected kind changing key m op b then map.nil else state);
      receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds ∗ view.unstable' ↦*{dq} unstable ∗ view.tracked' ↦*{dq} tracked ∗
      state ↦$ (constraintState active kind changing key m op b) }} := by
  unfold numericIntegerContract at hinteger
  wp_start as ⟨Hview, Hkinds, Hunstable, Htracked, Hstate⟩
  wp_auto
  wp_apply wp_trackedBinding receiver view tracked key active (by simpa [ht] using hk) htracked dq $$ [Hview Htracked] with ⟨Hview, Htracked⟩
  · iframe
  cases ha : active
  · simp [ha, constraintState]
    wp_auto
    wp_end
  · simp [ha]
    wp_auto
    wp_apply wp_binding receiver view kinds unstable state m key kind changing hk hkind hu hunstable dq $$ [Hview Hkinds Hunstable Hstate] with ⟨Hview, Hkinds, Hunstable, Hstate⟩
    · iframe
    simp [hnil]
    wp_auto
    wp_apply wp_lengthConstraint (bindingValue kind changing key m) op b
    wp_apply wp_exactInteger b
    cases he : isExact b
    · wp_auto
      wp_apply wp_infeasibleConstraint receiver view kinds key kind op b (lengthConstrained (bindingValue kind changing key m) op b) hk hkind dq $$ [Hview Hkinds] with ⟨Hview, Hkinds⟩
      · iframe
      have hv : constraintValue kind changing key m op b = lengthConstrained (bindingValue kind changing key m) op b := by simp [constraintValue, numericConstrained, he]
      rw [← hv]
      cases hr : constraintRejected kind changing key m op b
      all_goals unfold constraintRejected at hr
      all_goals simp [hr, constraintState, constraintRejected]
      all_goals try wp_auto
      · wp_apply wp_mapInsert pdg.Ref.ty state m key (constraintValue kind changing key m op b) $$ [Hstate] with Hstate
        · iframe
        try wp_auto
        wp_end
      · wp_end
    · wp_auto
      wp_apply hinteger $$ []
      · iframe; iassumption
      wp_apply wp_infeasibleConstraint receiver view kinds key kind op b (constrainedValue (lengthConstrained (bindingValue kind changing key m) op b) op b.Lower') hk hkind dq $$ [Hview Hkinds] with ⟨Hview, Hkinds⟩
      · iframe
      have hv : constraintValue kind changing key m op b = constrainedValue (lengthConstrained (bindingValue kind changing key m) op b) op b.Lower' := by simp [constraintValue, numericConstrained, he]
      rw [← hv]
      cases hr : constraintRejected kind changing key m op b
      all_goals unfold constraintRejected at hr
      all_goals simp [hr, constraintState, constraintRejected]
      all_goals try wp_auto
      · wp_apply wp_mapInsert pdg.Ref.ty state m key (constraintValue kind changing key m op b) $$ [Hstate] with Hstate
        · iframe
        try wp_auto
        wp_end
      · wp_end

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
