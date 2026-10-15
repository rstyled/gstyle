module
public import Gstyle.Proof.PdgPackage
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.pdg

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [query.Assumptions]
variable (T : go.GoType) [FuncUnfold listAt [T] (listAt.impl T)]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.pdg

theorem wp_listAt_present (table : GoSlice) (rows : List GoSlice) (dq : DFrac)
    (ref : w32) (row : GoSlice)
    (hv : 0 < uint.Z ref ∧ uint.Z ref ≤ rows.length)
    (hlookup : rows[(uint.Z ref - 1).toNat]? = some row) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ table ↦*{dq} rows }}
      (App (App (Val #(functions listAt [T])) (Val #table)) (Val #ref))
    {{ RET #row; table ↦*{dq} rows }} := by
  wp_start as Hrows
  ihave %Hlen := ownSlice_len $$ Hrows
  try wp_auto
  repeat (wp_if_destruct <;> try (exfalso; obtain ⟨_, _⟩ := hv; word))
  have Hi : sint.Z (W64 (uint.Z (ref - W32 1))) = uint.Z ref - 1 := by obtain ⟨_, _⟩ := hv; word
  rw [Hi]
  rw [ite_eq_left (by obtain ⟨_, _⟩ := hv; constructor <;> word : 0 ≤ uint.Z ref - 1 ∧ uint.Z ref - 1 < sint.Z table.len)]
  wp_apply wp_load_slice_index table (uint.Z ref - 1) rows dq row (by obtain ⟨_, _⟩ := hv; word) $$ [Hrows] with Hrows
  · iframe; ipureintro; exact hlookup
  wp_end

theorem wp_listAt_absent (table : GoSlice) (rows : List GoSlice) (dq : DFrac)
    (ref : w32) (hv : ¬ (0 < uint.Z ref ∧ uint.Z ref ≤ rows.length)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ table ↦*{dq} rows }}
      (App (App (Val #(functions listAt [T])) (Val #table)) (Val #ref))
    {{ RET #slice.nil; table ↦*{dq} rows }} := by
  wp_start as Hrows
  ihave %Hlen := ownSlice_len $$ Hrows
  try wp_auto
  wp_if_destruct
  · wp_end
  · wp_if_destruct
    · wp_end
    · exfalso
      apply hv
      constructor <;> word

def listRow (rows : List GoSlice) (ref : w32) : GoSlice :=
  if 0 < uint.Z ref ∧ uint.Z ref ≤ rows.length then
    (rows[(uint.Z ref - 1).toNat]?).getD slice.nil
  else slice.nil

theorem wp_listAt (table : GoSlice) (rows : List GoSlice) (dq : DFrac) (ref : w32) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ table ↦*{dq} rows }}
      (App (App (Val #(functions listAt [T])) (Val #table)) (Val #ref))
    {{ RET #(listRow rows ref); table ↦*{dq} rows }} := by
  wp_start_folded as Hrows
  by_cases hv : 0 < uint.Z ref ∧ uint.Z ref ≤ rows.length
  · have hi : (uint.Z ref - 1).toNat < rows.length := by omega
    have hl : rows[(uint.Z ref - 1).toNat]? = some rows[(uint.Z ref - 1).toNat] := by simp
    have hrow : listRow rows ref = rows[(uint.Z ref - 1).toNat] := by
      rw [listRow, ite_eq_left hv, hl]
      rfl
    rw [hrow]
    wp_apply wp_listAt_present T table rows dq ref rows[(uint.Z ref - 1).toNat] hv hl $$ [Hrows] with Hrows
    · iframe
    wp_end
  · have hrow : listRow rows ref = slice.nil := by rw [listRow, ite_eq_right hv]
    rw [hrow]
    wp_apply wp_listAt_absent T table rows dq ref hv $$ [Hrows] with Hrows
    · iframe
    wp_end

end proof
end github_com.rstyled.gstyle.domain.pdg
end Perennial
