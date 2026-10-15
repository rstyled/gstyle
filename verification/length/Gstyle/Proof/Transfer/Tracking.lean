module
public import Gstyle.Proof.StateMap.Layout
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
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

theorem wp_trackedBinding (receiver : Loc) (view : graphView) (tracked : List Bool)
    (key : w32) (active : Bool)
    (hk : 0 < uint.Z key ∧ uint.Z key ≤ tracked.length)
    (htracked : tracked[(uint.Z key - 1).toNat]? = some active) (dq : DFrac := DFrac.own 1) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{dq} view ∗ view.tracked' ↦*{dq} tracked) }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"trackedBinding" #receiver)) (Val #key))
    {{ RET #active; receiver ↦{dq} view ∗ view.tracked' ↦*{dq} tracked }} := by
  wp_start as ⟨Hview, Htracked⟩
  ihave %Hlen := ownSlice_len $$ Htracked
  wp_auto
  obtain ⟨hk0, hk1⟩ := hk
  repeat (wp_if_destruct <;> try (exfalso; word))
  have Hi : sint.Z (W64 (uint.Z (key - W32 1))) = uint.Z key - 1 := by word
  rw [Hi]
  rw [ite_eq_left (by constructor <;> word : 0 ≤ uint.Z key - 1 ∧ uint.Z key - 1 < sint.Z view.tracked'.len)]
  wp_apply wp_load_slice_index view.tracked' (uint.Z key - 1) tracked dq active (by word) $$ [Htracked] with Htracked
  · iframe; ipureintro; exact htracked
  wp_end

theorem wp_trackedBinding_invalid (receiver : Loc) (view : graphView) (tracked : List Bool)
    (key : w32) (dq : DFrac) (hkey : key = W32 0 ∨ tracked.length < uint.Z key) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{dq} view ∗ view.tracked' ↦*{dq} tracked) }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"trackedBinding" #receiver)) (Val #key))
    {{ RET #false; receiver ↦{dq} view ∗ view.tracked' ↦*{dq} tracked }} := by
  wp_start as ⟨Hview, Htracked⟩
  ihave %Hlen := ownSlice_len $$ Htracked
  try wp_auto
  rcases hkey with hz | hout
  · subst key
    try wp_auto
    wp_end
  · repeat (wp_if_destruct <;> try (exfalso; word))
    all_goals wp_end

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
