module
public import Gstyle.GeneratedProof.github_com.rstyled.gstyle.domain.patterns.outputcheck
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
instance isPkgInit_inst : IsPkgInit (IProp GF) pkg := define_is_pkg_init iprop(True)
instance get_isPkgInit_wf_inst : GetIsPkgInitWf (IProp GF) pkg := build_get_is_pkg_init_wf

theorem wp_Formatted (pkgName name : GoString) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! Formatted)) (Val #pkgName)) (Val #name))
    {{ RET #(decide (pkgName = go!"fmt" ∧
      (name = go!"Fprint" ∨ name = go!"Fprintf" ∨ name = go!"Fprintln"))); True }} := by
  wp_start
  wp_auto
  by_cases hp : pkgName = go!"fmt" <;>
    by_cases h0 : name = go!"Fprint" <;>
    by_cases h1 : name = go!"Fprintf" <;>
    by_cases h2 : name = go!"Fprintln"
  all_goals iterate 5 (try simp [hp, h0, h1, h2]; try wp_auto)
  all_goals wp_end

theorem wp_Memory (pkgName name : GoString) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! Memory)) (Val #pkgName)) (Val #name))
    {{ RET #(decide ((pkgName = go!"bytes" ∧ name = go!"Buffer") ∨
      (pkgName = go!"strings" ∧ name = go!"Builder"))); True }} := by
  wp_start
  wp_auto
  by_cases hp : pkgName = go!"bytes" <;>
    by_cases hq : pkgName = go!"strings" <;>
    by_cases h0 : name = go!"Buffer" <;>
    by_cases h1 : name = go!"Builder"
  all_goals iterate 5 (try simp [hp, hq, h0, h1]; try wp_auto)
  all_goals wp_end


def discarded (input : Decision) : Bool :=
  input.Formatted' && !input.Memory' &&
    (input.Standalone' || (decide (input.Values' = (1#64)) &&
      decide (input.Results' = (2#64)) && input.ErrorBlank'))

theorem wp_Discards (input : Decision) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (Val (@! Discards)) (Val #input))
    {{ RET #(discarded input); True }} := by
  wp_start
  wp_auto
  cases hf : input.Formatted' <;> cases hm : input.Memory' <;>
    cases hs : input.Standalone' <;> cases hb : input.ErrorBlank' <;>
    by_cases hv : input.Values' = (1#64) <;> by_cases hr : input.Results' = (2#64)
  all_goals iterate 5 (try simp [discarded, hf, hm, hs, hb, hv, hr]; try wp_auto)
  all_goals try simp only [hb]
  all_goals wp_end


theorem wp_Select (sites : GoSlice) (records : List Site) (suppressed : Bool) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ sites ↦* records }}
      (App (App (Val (@! Select)) (Val #sites)) (Val #suppressed))
    {{ (out : GoSlice), RET #out;
      sites ↦* records ∗ out ↦* (if suppressed then [] else records.filter Site.DiscardedOutput') }} := by
  wp_start as Hsites
  wp_auto
  cases suppressed
  · simp only [Bool.false_eq_true, ↓reduceIte]
    try wp_auto
    ihave %Hlen := ownSlice_len _ _ _ $$ Hsites
    ihave HI : (∃ (iv : w64) (sl : GoSlice),
        "i" ∷ i_ptr ↦ iv ∗
        "out" ∷ out_ptr ↦ sl ∗
        "Hsites" ∷ sites ↦* records ∗
        "Hout" ∷ sl ↦* ((records.take (sint.nat iv)).filter Site.DiscardedOutput') ∗
        "Hcap" ∷ ownSliceCap Site sl (DFrac.own 1) ∗
        "%Hbounds" ∷ ⌜0 ≤ sint.Z iv ∧ sint.Z iv ≤ sint.Z sites.len⌝ : IProp GF)
        $$ [i out Hsites]
    · iexists (W64 0), slice.nil
      simp only [show sint.nat (W64 0) = 0 from rfl, List.take_zero, List.filter_nil]
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
      list_elem records (sint.nat iv) as record
      rw [ite_eq_left (by constructor <;> word : 0 ≤ sint.Z iv ∧ sint.Z iv < sint.Z sites.len)]
      wp_apply wp_load_slice_index sites (sint.Z iv) records _ record (by word) $$ [Hsites] with Hsites
      · iframe
        ipureintro
        exact Hrecord_lookup
      try wp_auto
      cases hd : record.DiscardedOutput'
      · try simp only [hd, Bool.false_eq_true, ↓reduceIte]
        wp_auto
        wp_for_post
        iframe
        iexists (iv + W64 1), sl
        iframe
        have ht : records.take (sint.nat (iv + W64 1)) = records.take (sint.nat iv) ++ [record] := by
          have hi : sint.nat (iv + W64 1) = sint.nat iv + 1 := by word
          rw [hi, List.take_add_one]
          simp [Hrecord_lookup]
        rw [ht, List.filter_append]
        simp only [List.filter_cons, hd, Bool.false_eq_true, ↓reduceIte, List.filter_nil, List.append_nil]
        iframe
        ipureintro
        constructor <;> word
      · try simp only [hd, ↓reduceIte]
        wp_auto
        wp_apply wp_slice_literal (V := Site) [record]
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
        have ht : records.take (sint.nat (iv + W64 1)) = records.take (sint.nat iv) ++ [record] := by
          have hi : sint.nat (iv + W64 1) = sint.nat iv + 1 := by word
          rw [hi, List.take_add_one]
          simp [Hrecord_lookup]
        rw [ht, List.filter_append]
        simp only [List.filter_cons, hd, ↓reduceIte, List.filter_nil]
        iframe
        ipureintro
        constructor <;> word
    · try simp only [go.intoVal_inj.eq_iff, decide_eq_true_eq] at Hif
      try simp only [go.intoVal_inj.eq_iff, decide_eq_true_eq, decide_eq_false Hif, Bool.false_eq_true, ↓reduceIte]
      try wp_auto
      have hi : sint.nat iv = records.length := by word
      rw [hi, List.take_length]
      wp_end
  · simp only [↓reduceIte]
    try wp_auto
    wp_end
    iframe
    iapply ownSlice_nil


theorem wp_hasSuppression (kinds : GoSlice) (names : List GoString) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ kinds ↦* names }}
      (App (Val (@! hasSuppression)) (Val #kinds))
    {{ RET #(decide (go!"unchecked_output" ∈ names)); kinds ↦* names }} := by
  wp_start as Hkinds
  wp_auto
  ihave %Hlen := ownSlice_len _ _ _ $$ Hkinds
  ihave HI : (∃ (iv : w64),
      "i" ∷ i_ptr ↦ iv ∗
      "Hkinds" ∷ kinds ↦* names ∗
      "%Hbounds" ∷ ⌜0 ≤ sint.Z iv ∧ sint.Z iv ≤ sint.Z kinds.len⌝ ∗
      "%Hprefix" ∷ ⌜go!"unchecked_output" ∉ names.take (sint.nat iv)⌝ : IProp GF)
      $$ [i Hkinds]
  · iexists (W64 0)
    simp only [show sint.nat (W64 0) = 0 from rfl, List.take_zero, List.not_mem_nil]
    iframe
    ipureintro
    constructor <;> word
  wp_for HI
  wp_if_destruct
  · try simp only [go.intoVal_inj.eq_iff, decide_eq_true_eq] at Hif
    list_elem names (sint.nat iv) as name
    rw [ite_eq_left (by constructor <;> word : 0 ≤ sint.Z iv ∧ sint.Z iv < sint.Z kinds.len)]
    wp_apply wp_load_slice_index kinds (sint.Z iv) names _ name (by word) $$ [Hkinds] with Hkinds
    · iframe
      ipureintro
      exact Hname_lookup
    wp_if_destruct
    · try simp only [go.intoVal_inj.eq_iff, decide_eq_true_eq] at Hif
      have hm : go!"unchecked_output" ∈ names := by
        have hn := List.mem_of_getElem? Hname_lookup
        try simp_all
      simp only [decide_eq_true hm]
      wp_for_post
      wp_end
    · try simp only [go.intoVal_inj.eq_iff, decide_eq_true_eq] at Hif
      wp_for_post
      iframe
      iexists (iv + W64 1)
      iframe
      ipureintro
      constructor
      · constructor <;> word
      · have hi : sint.nat (iv + W64 1) = sint.nat iv + 1 := by word
        rw [hi, List.take_add_one]
        simp [Hname_lookup, Hprefix, Hif, eq_comm]
  · try simp only [go.intoVal_inj.eq_iff, decide_eq_true_eq] at Hif
    try simp only [go.intoVal_inj.eq_iff, decide_eq_true_eq, decide_eq_false Hif, Bool.false_eq_true, ↓reduceIte]
    try wp_auto
    have hi : sint.nat iv = names.length := by word
    rw [hi, List.take_length] at Hprefix
    simp only [decide_eq_false Hprefix]
    wp_end

theorem wp_Find (sites suppressions : GoSlice) (records : List Site) (names : List GoString) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ sites ↦* records ∗ suppressions ↦* names }}
      (App (App (Val (@! Find)) (Val #sites)) (Val #suppressions))
    {{ (out : GoSlice), RET #out;
      sites ↦* records ∗ suppressions ↦* names ∗
        out ↦* (if go!"unchecked_output" ∈ names then [] else records.filter Site.DiscardedOutput') }} := by
  wp_start as ⟨Hsites, Hnames⟩
  wp_auto
  wp_apply wp_hasSuppression suppressions names $$ [Hnames] with Hnames
  · iframe
  wp_apply wp_Select sites records (decide (go!"unchecked_output" ∈ names)) $$ [Hsites] with %out ⟨Hsites, Hout⟩
  · iframe
  wp_end
  simp only [decide_eq_true_eq]
  iframe

end proof
end github_com.rstyled.gstyle.domain.patterns.outputcheck
end Perennial
