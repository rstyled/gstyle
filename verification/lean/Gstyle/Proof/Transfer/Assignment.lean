module
public import Gstyle.Proof.StateMap.Binding
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [FfiSyntax]

def assignedValue (kind : GoString) (v : Value) : Value :=
  if kind = go!"\"unsigned\"" ∧ v.HasLower' = false then
    { v with Lower' := 0, HasLower' := true }
  else v

def selectsRhs (category : w8) (form : GoString) (writes : w64) : Bool :=
  decide (writes = 1#64 ∧ (form = go!"=" ∨ form = go!":=" ∨ (form = go!"" ∧ category = 0#8)))

def compoundForms : List GoString :=
  [go!"+=", go!"-=", go!"*=", go!"/=", go!"%=", go!"&=", go!"|=", go!"^=", go!"<<=", go!">>=", go!"&^="]

theorem assignment_interval_sound (kind : GoString) (v : Value) (x : Int)
    (hx : interval.contains (bounds v) x) (hu : kind = go!"\"unsigned\"" → 0 ≤ x) :
    interval.contains (bounds (assignedValue kind v)) x := by
  unfold assignedValue
  split
  · rename_i h
    obtain ⟨hk, hl⟩ := h
    simp [bounds, interval.contains]
    exact ⟨hu hk, hx.2⟩
  · exact hx

theorem compound_discards_rhs (category : w8) (form : GoString) (writes : w64)
    (hform : form ∈ compoundForms) : selectsRhs category form writes = false := by
  simp [compoundForms] at hform
  rcases hform with h | h | h | h | h | h | h | h | h | h | h
  all_goals subst form
  all_goals simp [selectsRhs]

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [graphView.TypeAssumptions] [Value.TypeAssumptions] [State.TypeAssumptions] [pdg.query.Assumptions]
variable [pdg.Category.TypeAssumptions]
variable [FuncUnfold singleValueWrite [] singleValueWrite.impl]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

theorem wp_singleValueWrite (category : w8) (form : GoString) (writes : w64) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (App (Val (@! singleValueWrite)) (Val #category)) (Val #form)) (Val #writes))
    {{ RET #(selectsRhs category form writes); True }} := by
  wp_start
  by_cases hw : writes = 1#64 <;> by_cases ha : form = go!"=" <;>
    by_cases hd : form = go!":=" <;> by_cases he : form = go!"" <;> by_cases hc : category = 0#8
  all_goals iterate 8 (try simp [selectsRhs, hw, ha, hd, he, hc]; try wp_auto)
  all_goals wp_end

theorem wp_assignmentValue (receiver : Loc) (view : graphView) (kinds : List GoString)
    (key : w32) (kind : GoString) (v : Value)
    (hk : 0 < uint.Z key ∧ uint.Z key ≤ kinds.length)
    (hkind : kinds[(uint.Z key - 1).toNat]? = some kind) (dq : DFrac := DFrac.own 1) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"assignmentValue" #receiver)) (Val #key)) (Val #v))
    {{ RET #(assignedValue kind v); receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds }} := by
  wp_start as ⟨Hview, Hkinds⟩
  wp_auto
  wp_apply wp_base receiver view kinds key kind hk hkind dq $$ [Hview Hkinds] with ⟨Hview, Hkinds⟩
  · iframe
  by_cases hu : kind = go!"\"unsigned\"" <;> cases hl : v.HasLower'
  all_goals iterate 8 (try simp [assignedValue, baseKind, hu, hl]; try wp_auto)
  all_goals wp_end

theorem wp_assign_nonpointer (receiver : Loc) (view : graphView) (kinds : List GoString)
    (state : Loc) (m : StateData) (ref key : w32) (kind : GoString) (v : Value)
    (hk : 0 < uint.Z key ∧ uint.Z key ≤ kinds.length)
    (hkind : kinds[(uint.Z key - 1).toNat]? = some kind) (hn : kind ≠ go!"\"pointer\"") (dq : DFrac := DFrac.own 1) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"assign" #receiver)) (Val #ref)) (Val #key)) (Val #v)) (Val #state))
    {{ RET #(); receiver ↦{dq} view ∗ view.kinds' ↦*{dq} kinds ∗ state ↦$ (<[key := assignedValue kind v]>m) }} := by
  wp_start as ⟨Hview, Hkinds, Hstate⟩
  ihave %Hlen := ownSlice_len $$ Hkinds
  wp_auto
  wp_apply wp_assignmentValue receiver view kinds key kind v hk hkind dq $$ [Hview Hkinds] with ⟨Hview, Hkinds⟩
  · iframe
  have Hi : sint.Z (W64 (uint.Z (key - W32 1))) = uint.Z key - 1 := by obtain ⟨_, _⟩ := hk; word
  rw [Hi]
  rw [ite_eq_left (by obtain ⟨_, _⟩ := hk; constructor <;> word : 0 ≤ uint.Z key - 1 ∧ uint.Z key - 1 < sint.Z view.kinds'.len)]
  wp_apply wp_load_slice_index view.kinds' (uint.Z key - 1) kinds dq kind (by obtain ⟨_, _⟩ := hk; word) $$ [Hkinds] with Hkinds
  · iframe; ipureintro; exact hkind
  simp [hn]
  wp_auto
  wp_apply wp_mapInsert pdg.Ref.ty state m key (assignedValue kind v) $$ [Hstate] with Hstate
  · iframe
  wp_end

theorem wp_singleValueWrite_compound (category : w8) (form : GoString) (writes : w64)
    (hform : form ∈ compoundForms) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (App (Val (@! singleValueWrite)) (Val #category)) (Val #form)) (Val #writes))
    {{ RET #false; True }} := by
  wp_start_folded
  wp_apply wp_singleValueWrite category form writes
  rw [compound_discards_rhs category form writes hform]
  wp_end

theorem wp_assign_interval_sound (receiver : Loc) (view : graphView) (kinds : List GoString)
    (state : Loc) (m : StateData) (ref key : w32) (kind : GoString) (v : Value)
    (hk : 0 < uint.Z key ∧ uint.Z key ≤ kinds.length)
    (hkind : kinds[(uint.Z key - 1).toNat]? = some kind) (hn : kind ≠ go!"\"pointer\"")
    (x : Int) (hx : interval.contains (bounds v) x) (hu : kind = go!"\"unsigned\"" → 0 ≤ x) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦ view ∗ view.kinds' ↦* kinds ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"assign" #receiver)) (Val #ref)) (Val #key)) (Val #v)) (Val #state))
    {{ RET #(); receiver ↦ view ∗ view.kinds' ↦* kinds ∗ state ↦$ (<[key := assignedValue kind v]>m) ∗
      ⌜interval.contains (bounds (assignedValue kind v)) x ∧
        ∀ other, key ≠ other → (<[key := assignedValue kind v]>m) !! other = m !! other⌝ }} := by
  wp_start_folded as Hpre
  wp_apply wp_assign_nonpointer receiver view kinds state m ref key kind v hk hkind hn $$ [Hpre] with ⟨Hview, Hkinds, Hstate⟩
  · iexact Hpre
  wp_end
  iframe
  ipureintro
  constructor
  · exact assignment_interval_sound kind v x hx hu
  · intro other hne
    simp [hne]

theorem wp_assign_unknown_signed (receiver : Loc) (view : graphView) (kinds : List GoString)
    (state : Loc) (m : StateData) (ref key : w32)
    (hk : 0 < uint.Z key ∧ uint.Z key ≤ kinds.length)
    (hkind : kinds[(uint.Z key - 1).toNat]? = some go!"\"signed\"") :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦ view ∗ view.kinds' ↦* kinds ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"assign" #receiver)) (Val #ref)) (Val #key)) (Val #(zero_val Value))) (Val #state))
    {{ RET #(); receiver ↦ view ∗ view.kinds' ↦* kinds ∗ state ↦$ (<[key := (zero_val Value)]>m) ∗
      ⌜∀ x : Int, interval.contains (bounds (zero_val Value)) x⌝ }} := by
  wp_start_folded as Hpre
  wp_apply wp_assign_nonpointer receiver view kinds state m ref key go!"\"signed\"" (zero_val Value) hk hkind (by decide) $$ [Hpre] with ⟨Hview, Hkinds, Hstate⟩
  · iexact Hpre
  simp [assignedValue]
  wp_end
  iframe
  ipureintro
  intro x
  simp [bounds, interval.contains, zero_val]

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
