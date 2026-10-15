module
public import Gstyle.Proof.Transfer.Tracking
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

theorem wp_exactInteger (v : Value) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (Val (@! exactInteger)) (Val #v))
    {{ RET #(isExact v); True }} := by
  wp_start
  cases hl : v.HasLower' <;> cases hu : v.HasUpper' <;> by_cases he : v.Lower' = v.Upper'
  all_goals iterate 6 (try simp [isExact, hl, hu, he]; try wp_auto)
  all_goals wp_end

theorem wp_emptyRange (v : Value) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (Val (@! emptyRange)) (Val #v))
    {{ RET #(isEmpty v); True }} := by
  wp_start
  cases hl : v.HasLower' <;> cases hu : v.HasUpper' <;> by_cases he : sint.Z v.Lower' > sint.Z v.Upper'
  all_goals iterate 6 (try simp [isEmpty, hl, hu, he]; try wp_auto)
  all_goals wp_end

theorem wp_lengthConstraint (v : Value) (op : GoString) (b : Value) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (App (Val (@! lengthConstraint)) (Val #v)) (Val #op)) (Val #b))
    {{ RET #(lengthConstrained v op b); True }} := by
  wp_start
  cases hb : b.Length' <;> cases hv : v.BelowLength' <;>
    by_cases hl : op = go!"<" <;> by_cases he : op = go!"<="
  all_goals iterate 8 (try simp [lengthConstrained, hb, hv, hl, he]; try wp_auto)
  all_goals wp_end

theorem wp_impossibleSignedComparison (receiver : Loc) (view : graphView) (kinds : List GoString)
    (key : w32) (kind op : GoString) (b : Value)
    (hk : 0 < uint.Z key ∧ uint.Z key ≤ kinds.length)
    (hkind : kinds[(uint.Z key - 1).toNat]? = some kind) (dq : DFrac := DFrac.own 1) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds) }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"impossibleSignedComparison" #receiver)) (Val #key)) (Val #op)) (Val #b))
    {{ RET #(impossibleComparison kind op b); receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds }} := by
  wp_start as ⟨Hview, Hkinds⟩
  ihave %Hlen := ownSlice_len $$ Hkinds
  wp_auto
  have Hi : sint.Z (W64 (uint.Z (key - W32 1))) = uint.Z key - 1 := by obtain ⟨_, _⟩ := hk; word
  rw [Hi]
  rw [ite_eq_left (by obtain ⟨_, _⟩ := hk; constructor <;> word : 0 ≤ uint.Z key - 1 ∧ uint.Z key - 1 < sint.Z view.kinds'.len)]
  wp_apply wp_load_slice_index view.kinds' (uint.Z key - 1) kinds dq kind (by obtain ⟨_, _⟩ := hk; word) $$ [Hkinds] with Hkinds
  · iframe; ipureintro; exact hkind
  by_cases hs : kind = go!"\"signed\""
  · simp [hs]
    wp_auto
    wp_apply wp_exactInteger b
    cases he : isExact b <;> by_cases hl : op = go!"<" <;> by_cases hg : op = go!">"
    all_goals iterate 8 (try simp [impossibleComparison, hs, he, hl, hg]; try wp_auto)
    all_goals wp_end
  · iterate 5 (try simp [impossibleComparison, hs]; try wp_auto)
    wp_end

theorem wp_infeasibleConstraint (receiver : Loc) (view : graphView) (kinds : List GoString)
    (key : w32) (kind op : GoString) (b v : Value)
    (hk : 0 < uint.Z key ∧ uint.Z key ≤ kinds.length)
    (hkind : kinds[(uint.Z key - 1).toNat]? = some kind) (dq : DFrac := DFrac.own 1) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"infeasibleConstraint" #receiver)) (Val #key)) (Val #op)) (Val #b)) (Val #v))
    {{ RET #(isEmpty v || impossibleComparison kind op b); receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds }} := by
  wp_start as ⟨Hview, Hkinds⟩
  wp_auto
  wp_apply wp_emptyRange v
  cases he : isEmpty v
  · simp [he]
    wp_auto
    wp_apply wp_impossibleSignedComparison receiver view kinds key kind op b hk hkind dq $$ [Hview Hkinds] with ⟨Hview, Hkinds⟩
    · iframe
    wp_end
  · simp [he]
    wp_auto
    wp_end


end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
