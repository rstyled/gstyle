module
public import Gstyle.Proof.Pdg.List
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.pdg
section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics]
variable {V : Type} [ZeroVal V] [TypedPointsto (GF := GF) V]

def readonlyRows : List GoSlice → List (List V) → IProp GF
  | [], [] => iprop(True)
  | row :: rows, values :: contents => iprop(row ↦*□ values ∗ readonlyRows rows contents)
  | _, _ => iprop(⌜False⌝)

instance readonlyRows_persistent (rows : List GoSlice) (contents : List (List V)) :
    Persistent (readonlyRows (GF := GF) rows contents) := by
  induction rows generalizing contents with
  | nil => cases contents <;> unfold readonlyRows <;> infer_instance
  | cons row rows ih =>
    cases contents with
    | nil => unfold readonlyRows; infer_instance
    | cons values contents =>
      haveI := ih contents
      unfold readonlyRows
      infer_instance

theorem readonlyRows_length (rows : List GoSlice) (contents : List (List V)) :
    readonlyRows (GF := GF) rows contents ⊢ ⌜rows.length = contents.length⌝ := by
  induction rows generalizing contents with
  | nil =>
    cases contents with
    | nil => unfold readonlyRows; iintro _; ipureintro; rfl
    | cons values contents => unfold readonlyRows; iintro %h; exact h.elim
  | cons row rows ih =>
    cases contents with
    | nil => unfold readonlyRows; iintro %h; exact h.elim
    | cons values contents =>
      unfold readonlyRows
      iintro ⟨_, Htail⟩
      ihave %Hlen := ih contents $$ Htail
      ipureintro
      simpa using congrArg Nat.succ Hlen

theorem readonlyRows_at (rows : List GoSlice) (contents : List (List V)) (i : Nat) :
    readonlyRows (GF := GF) rows contents ⊢
      (rows[i]?.getD slice.nil) ↦*□ (contents[i]?.getD []) := by
  induction rows generalizing contents i with
  | nil =>
    cases contents with
    | nil =>
      simp only [List.getElem?_nil, Option.getD_none]
      iintro _
      iapply ownSlice_nil
    | cons values contents => unfold readonlyRows; iintro %h; exact h.elim
  | cons row rows ih =>
    cases contents with
    | nil => unfold readonlyRows; iintro %h; exact h.elim
    | cons values contents =>
      cases i with
      | zero =>
        simp only [List.getElem?_cons_zero, Option.getD_some]
        unfold readonlyRows
        iintro ⟨Hrow, _⟩
        iexact Hrow
      | succ i =>
        simp only [List.getElem?_cons_succ]
        unfold readonlyRows
        iintro ⟨_, Htail⟩
        iapply ih contents i $$ Htail

def rowContents (contents : List (List V)) (ref : w32) : List V :=
  if 0 < uint.Z ref ∧ uint.Z ref ≤ contents.length then
    contents[(uint.Z ref - 1).toNat]?.getD [] else []

theorem readonlyRows_at_ref (rows : List GoSlice) (contents : List (List V)) (ref : w32) :
    readonlyRows (GF := GF) rows contents ⊢
      listRow rows ref ↦*□ rowContents contents ref := by
  iintro #Hrows
  ihave %Hlen := readonlyRows_length rows contents $$ Hrows
  unfold listRow rowContents
  rw [← Hlen]
  by_cases hv : 0 < uint.Z ref ∧ uint.Z ref ≤ rows.length
  · simp only [ite_eq_left hv]
    iapply readonlyRows_at rows contents (uint.Z ref - 1).toNat $$ Hrows
  · simp only [ite_eq_right hv]
    iapply ownSlice_nil

end proof
end github_com.rstyled.gstyle.domain.pdg
end Perennial
