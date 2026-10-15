module
public import Gstyle.Proof.StateMap.Model
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [go.Semantics] [view_sem : graphView.TypeAssumptions]
variable [Value.TypeAssumptions] [State.TypeAssumptions] [pdg.query.Assumptions]

theorem wp_base (receiver : Loc) (view : graphView) (kinds : List GoString)
    (key : w32) (kind : GoString)
    (hk : 0 < uint.Z key ∧ uint.Z key ≤ kinds.length)
    (hkind : kinds[(uint.Z key - 1).toNat]? = some kind) (dq : DFrac := DFrac.own 1) :
    {{ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard ∗ (receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds) }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"base" #receiver)) (Val #key))
    {{ RET #(baseKind kind); receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds }} := by
  wp_start as ⟨Hview, Hkinds⟩
  ihave %Hlen := ownSlice_len $$ Hkinds
  wp_auto
  obtain ⟨hk0, hk1⟩ := hk
  repeat (wp_if_destruct <;> try (exfalso; word))
  have Hi : sint.Z (W64 (uint.Z (key - W32 1))) = uint.Z key - 1 := by word
  rw [Hi]
  rw [ite_eq_left (by constructor <;> word : 0 ≤ uint.Z key - 1 ∧ uint.Z key - 1 < sint.Z view.kinds'.len)]
  wp_apply wp_load_slice_index view.kinds' (uint.Z key - 1) kinds dq kind (by word) $$ [Hkinds] with Hkinds
  · iframe; ipureintro; exact hkind
  by_cases hu : kind = go!"\"unsigned\""
  all_goals iterate 5 (try simp [baseKind, hu]; try wp_pures; try wp_auto)
  all_goals wp_end

theorem wp_binding (receiver : Loc) (view : graphView) (kinds : List GoString)
    (unstable : List Bool) (state : Loc) (m : StateData)
    (key : w32) (kind : GoString) (changing : Bool)
    (hk : 0 < uint.Z key ∧ uint.Z key ≤ kinds.length)
    (hkind : kinds[(uint.Z key - 1).toNat]? = some kind)
    (hlen : unstable.length = kinds.length)
    (hunstable : unstable[(uint.Z key - 1).toNat]? = some changing) (dq : DFrac := DFrac.own 1) :
    {{ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard ∗
      (receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds ∗ view.unstable' ↦*{dq} unstable ∗ state ↦$ m) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"binding" #receiver)) (Val #key)) (Val #state))
    {{ RET #(bindingValue kind changing key m);
      receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds ∗ view.unstable' ↦*{dq} unstable ∗ state ↦$ m }} := by
  wp_start as ⟨Hview, Hkinds, Hunstable, Hstate⟩
  ihave %Hklen := ownSlice_len $$ Hkinds
  ihave %Hulen := ownSlice_len $$ Hunstable
  wp_auto
  obtain ⟨hk0, hk1⟩ := hk
  repeat (wp_if_destruct <;> try (exfalso; word))
  have Hi : sint.Z (W64 (uint.Z (key - W32 1))) = uint.Z key - 1 := by word
  rw [Hi]
  rw [ite_eq_left (by constructor <;> word : 0 ≤ uint.Z key - 1 ∧ uint.Z key - 1 < sint.Z view.unstable'.len)]
  wp_apply wp_load_slice_index view.unstable' (uint.Z key - 1) unstable dq changing (by word) $$ [Hunstable] with Hunstable
  · iframe; ipureintro; exact hunstable
  cases changing
  · wp_auto
    wp_apply wp_map_lookup2 pdg.Ref.ty Value.ty state m key (DFrac.own 1) $$ [Hstate] with Hstate
    · iframe
    cases hm : m !! key with
    | some found =>
      simp [hm, bindingValue]
      wp_auto
      wp_end
    | none =>
      simp [hm]
      wp_auto
      wp_apply wp_base receiver view kinds key kind ⟨hk0, hk1⟩ hkind dq $$ [Hview Hkinds] with ⟨Hview, Hkinds⟩
      · iframe
      have Hi2 : sint.Z (W64 (uint.Z (key - 1#32))) = uint.Z key - 1 := by word
      rw [Hi2]
      rw [ite_eq_left (by constructor <;> word : 0 ≤ uint.Z key - 1 ∧ uint.Z key - 1 < sint.Z view.kinds'.len)]
      wp_apply wp_load_slice_index view.kinds' (uint.Z key - 1) kinds dq kind (by word) $$ [Hkinds] with Hkinds
      · iframe; ipureintro; exact hkind
      by_cases hp : kind = go!"\"pointer\""
      all_goals iterate 5 (try simp [bindingValue, defaultKind, hm, hp]; try wp_pures; try wp_auto)
      all_goals simp [baseKind]
      all_goals wp_end
  · wp_auto
    wp_apply wp_base receiver view kinds key kind ⟨hk0, hk1⟩ hkind dq $$ [Hview Hkinds] with ⟨Hview, Hkinds⟩
    · iframe
    simp [bindingValue]
    wp_end


end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
