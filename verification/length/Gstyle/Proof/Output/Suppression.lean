module
public import Gstyle.Proof.Output.Base
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

end proof
end github_com.rstyled.gstyle.domain.patterns.outputcheck
end Perennial
