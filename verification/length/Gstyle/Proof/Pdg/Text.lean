module
public import Gstyle.Proof.Pdg.TablesLayout
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.pdg

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [Tables.TypeAssumptions] [query.Assumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.pdg

theorem wp_Text_present (receiver : Loc) (table : Tables) (strings : List GoString)
    (dq : DFrac) (id : w32) (value : GoString)
    (hv : 0 < uint.Z id ∧ uint.Z id ≤ strings.length)
    (hlookup : strings[(uint.Z id - 1).toNat]? = some value) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{dq} table ∗ table.Strings' ↦*{dq} strings) }}
      (App (Val #(methods (go.PointerType Tables.ty) go!"Text" #receiver)) (Val #id))
    {{ RET #value; receiver ↦{dq} table ∗ table.Strings' ↦*{dq} strings }} := by
  wp_start as ⟨Htable, Hstrings⟩
  ihave %Hlen := ownSlice_len $$ Hstrings
  wp_auto
  wp_apply query.wp_ReferenceIndex id table.Strings'.len (by unfold query.validReference; obtain ⟨_, _⟩ := hv; constructor <;> word) with %index %Hindex
  rw [ite_eq_left Hindex.2]
  wp_apply wp_load_slice_index table.Strings' (sint.Z index) strings dq value Hindex.2.1 $$ [Hstrings] with Hstrings
  · iframe; ipureintro; simpa [Hindex.1] using hlookup
  wp_end

theorem wp_Text_absent (receiver : Loc) (table : Tables) (strings : List GoString)
    (dq : DFrac) (id : w32) (hv : ¬ (0 < uint.Z id ∧ uint.Z id ≤ strings.length)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{dq} table ∗ table.Strings' ↦*{dq} strings) }}
      (App (Val #(methods (go.PointerType Tables.ty) go!"Text" #receiver)) (Val #id))
    {{ RET #(go!""); receiver ↦{dq} table ∗ table.Strings' ↦*{dq} strings }} := by
  wp_start as ⟨Htable, Hstrings⟩
  ihave %Hlen := ownSlice_len $$ Hstrings
  wp_auto
  wp_apply query.wp_ReferenceReject id table.Strings'.len (by
    intro h
    apply hv
    unfold query.validReference at h
    obtain ⟨_, _⟩ := h
    constructor <;> word)
  wp_end

def tableText (strings : List GoString) (id : w32) : GoString :=
  if 0 < uint.Z id ∧ uint.Z id ≤ strings.length then
    (strings[(uint.Z id - 1).toNat]?).getD (go!"")
  else go!""

theorem wp_Text (receiver : Loc) (table : Tables) (strings : List GoString)
    (dq : DFrac) (id : w32) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{dq} table ∗ table.Strings' ↦*{dq} strings) }}
      (App (Val #(methods (go.PointerType Tables.ty) go!"Text" #receiver)) (Val #id))
    {{ RET #(tableText strings id); receiver ↦{dq} table ∗ table.Strings' ↦*{dq} strings }} := by
  wp_start_folded as ⟨Htable, Hstrings⟩
  by_cases hv : 0 < uint.Z id ∧ uint.Z id ≤ strings.length
  · have hi : (uint.Z id - 1).toNat < strings.length := by omega
    have hl : strings[(uint.Z id - 1).toNat]? = some strings[(uint.Z id - 1).toNat] := by simp [hi]
    have htext : tableText strings id = strings[(uint.Z id - 1).toNat] := by
      rw [tableText, if_pos hv, hl]
      rfl
    rw [htext]
    wp_apply wp_Text_present receiver table strings dq id strings[(uint.Z id - 1).toNat] hv hl $$ [Htable Hstrings] with ⟨Htable, Hstrings⟩
    · iframe
    wp_end
  · have htext : tableText strings id = go!"" := by rw [tableText, if_neg hv]
    rw [htext]
    wp_apply wp_Text_absent receiver table strings dq id hv $$ [Htable Hstrings] with ⟨Htable, Hstrings⟩
    · iframe
    wp_end

end proof
end github_com.rstyled.gstyle.domain.pdg
end Perennial
