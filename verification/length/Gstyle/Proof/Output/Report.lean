module
public import Gstyle.Proof.Output.Base
public import Gstyle.Proof.Output.Find
public import Gstyle.Proof.Output.Describe
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.patterns.outputcheck

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics] [package_sem : Assumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.patterns.outputcheck
theorem wp_Report (function : Function) (sites suppressions : GoSlice)
    (records : List Site) (names : List GoString) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ sites ↦* records ∗ suppressions ↦* names }}
      (App (App (App (Val (@! Report)) (Val #function)) (Val #sites)) (Val #suppressions))
    {{ (out : GoSlice), RET #out;
      sites ↦* records ∗ suppressions ↦* names ∗
        out ↦* ((if go!"unchecked_output" ∈ names then [] else records.filter Site.DiscardedOutput').map (advisory function)) }} := by
  wp_start as ⟨Hsites, Hnames⟩
  wp_auto
  wp_apply wp_Find sites suppressions records names $$ [Hsites Hnames] with %selected ⟨Hsites, Hnames, Hselected⟩
  · iframe
  try wp_auto
  let chosen := if go!"unchecked_output" ∈ names then [] else records.filter Site.DiscardedOutput'
  ihave %Hlen := ownSlice_len _ _ _ $$ Hselected
  have Hchosen : chosen.length = sint.nat selected.len := Hlen.1
  ihave HI : (∃ (iv : w64) (sl : GoSlice),
      "i" ∷ i_ptr ↦ iv ∗
      "out" ∷ out_ptr ↦ sl ∗
      "Hselected" ∷ selected ↦* chosen ∗
      "Hout" ∷ sl ↦* ((chosen.take (sint.nat iv)).map (advisory function)) ∗
      "Hcap" ∷ ownSliceCap Advisory sl (DFrac.own 1) ∗
      "%Hbounds" ∷ ⌜0 ≤ sint.Z iv ∧ sint.Z iv ≤ sint.Z selected.len⌝ : IProp GF)
      $$ [i out Hselected]
  · iexists (W64 0), slice.nil
    simp only [show sint.nat (W64 0) = 0 from rfl, List.take_zero, List.map_nil]
    iframe
    isplitr
    · iapply ownSlice_nil
    isplitr
    · iapply ownSliceCap_nil
    ipureintro
    constructor <;> word
  wp_for HI
  wp_if_destruct
  · try simp only [go.intoVal_inj.eq_iff, decide_eq_true_eq] at Hif
    have Hindex : sint.nat iv < chosen.length := by rw [Hchosen]; word
    list_elem chosen (sint.nat iv) as record
    rw [ite_eq_left (by constructor <;> word : 0 ≤ sint.Z iv ∧ sint.Z iv < sint.Z selected.len)]
    wp_apply wp_load_slice_index selected (sint.Z iv) chosen _ record (by word) $$ [Hselected] with Hselected
    · iframe
      ipureintro
      exact Hrecord_lookup
    wp_apply wp_describe function record
    wp_apply wp_slice_literal (V := Advisory) [advisory function record]
    isplitr
    · ipureintro; rfl
    iintro %one ⟨Hone, -⟩
    wp_auto
    wp_apply wp_slice_append $$ [Hout Hcap Hone] with %sl' ⟨Hout, Hcap, -⟩
    · iframe
    wp_for_post
    iframe
    iexists (iv + W64 1), sl'
    iframe
    have ht : chosen.take (sint.nat (iv + W64 1)) = chosen.take (sint.nat iv) ++ [record] := by
      have hi : sint.nat (iv + W64 1) = sint.nat iv + 1 := by word
      rw [hi, List.take_add_one]
      simp [Hrecord_lookup]
    rw [ht, List.map_append]
    simp only [List.map_cons, List.map_nil]
    iframe
    ipureintro
    constructor <;> word
  · try simp only [go.intoVal_inj.eq_iff, decide_eq_true_eq] at Hif
    try simp only [go.intoVal_inj.eq_iff, decide_eq_true_eq, decide_eq_false Hif, Bool.false_eq_true, ↓reduceIte]
    try wp_auto
    have hi : sint.nat iv = chosen.length := by rw [Hchosen]; word
    rw [hi, List.take_length]
    wp_end


end proof
end github_com.rstyled.gstyle.domain.patterns.outputcheck
end Perennial
