module
public import Gstyle.Proof.Interval.ModelSelect
public import Gstyle.Proof.Interval.Risk
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard.interval

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics] [package_sem : Assumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval
set_option maxHeartbeats 2000000 in
theorem wp_Select (accesses : GoSlice) (records : List Access) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ accesses ↦* records }}
      (App (Val (@! Select)) (Val #accesses))
    {{ (out : GoSlice), RET #out; accesses ↦* records ∗ out ↦* selectFrom [] records }} := by
  wp_start as Haccesses
  wp_auto
  wp_apply (wp_map_make1 (K := w32) (V := Bool)) with %seenMap Hmap
  try wp_auto
  ihave %Hlen := ownSlice_len _ _ _ $$ Haccesses
  ihave HI : (∃ (iv : w64) (sl : GoSlice) (m : GMap w32 Bool) (keys : List w32) (picked : List Access),
      "i" ∷ i_ptr ↦ iv ∗
      "out" ∷ out_ptr ↦ sl ∗
      "Haccesses" ∷ accesses ↦* records ∗
      "Hmap" ∷ seenMap ↦$ m ∗
      "Hout" ∷ sl ↦* picked ∗
      "Hcap" ∷ ownSliceCap Access sl (DFrac.own 1) ∗
      "%Hbounds" ∷ ⌜0 ≤ sint.Z iv ∧ sint.Z iv ≤ sint.Z accesses.len⌝ ∗
      "%Hmem" ∷ ⌜∀ key, (m !! key).getD false = decide (key ∈ keys)⌝ ∗
      "%Hrest" ∷ ⌜picked ++ selectFrom keys (records.drop (sint.nat iv)) = selectFrom [] records⌝ : IProp GF)
      $$ [i out Haccesses Hmap]
  · iexists (W64 0), slice.nil, (∅ : GMap w32 Bool), ([] : List w32), ([] : List Access)
    simp only [show sint.nat (W64 0) = 0 from rfl, List.drop_zero, List.nil_append]
    iframe
    isplitr
    · iapply ownSlice_nil
    isplitr
    · iapply ownSliceCap_nil
    isplitr
    · ipureintro; constructor <;> word
    isplitr
    · ipureintro; intro key; simp
    ipureintro; trivial
  wp_for HI
  wp_if_destruct
  · try simp only [go.intoVal_inj.eq_iff, decide_eq_true_eq] at Hif
    list_elem records (sint.nat iv) as record
    have Hdrop : records.drop (sint.nat iv) = record :: records.drop (sint.nat iv + 1) := by
      rw [List.drop_eq_getElem?_toList_append, Hrecord_lookup]
      rfl
    rw [Hdrop] at Hrest
    rw [ite_eq_left (by constructor <;> word : 0 ≤ sint.Z iv ∧ sint.Z iv < sint.Z accesses.len)]
    wp_apply wp_load_slice_index accesses (sint.Z iv) records _ record (by word) $$ [Haccesses] with Haccesses
    · iframe; ipureintro; exact Hrecord_lookup
    wp_apply wp_NegativeIndex record.Index' record.UpperChecked' record.Bounds'
    cases hr : risky record
    · simp only [hr]
      try wp_auto
      wp_for_post
      iframe
      iexists (iv + W64 1), sl, m, keys, picked
      iframe
      isplitr
      · ipureintro; constructor <;> word
      isplitr
      · ipureintro; exact Hmem
      ipureintro
      have hi : sint.nat (iv + W64 1) = sint.nat iv + 1 := by word
      rw [hi]
      simpa [selectFrom, hr] using Hrest
    · simp only [hr]
      try wp_auto
      wp_apply wp_map_lookup1 go.uint32 go.bool seenMap m record.Symbol' (DFrac.own 1) $$ [Hmap] with Hmap
      · iframe
      simp only [show (zero_val Bool) = false from rfl]
      rw [Hmem record.Symbol']
      by_cases hs : record.Symbol' ∈ keys
      · simp only [decide_eq_true hs]
        try wp_auto
        wp_for_post
        iframe
        iexists (iv + W64 1), sl, m, keys, picked
        iframe
        isplitr
        · ipureintro; constructor <;> word
        isplitr
        · ipureintro; exact Hmem
        ipureintro
        have hi : sint.nat (iv + W64 1) = sint.nat iv + 1 := by word
        rw [hi]
        simpa [selectFrom, hs] using Hrest
      · simp only [decide_eq_false hs]
        try wp_auto
        wp_apply wp_mapInsert go.uint32 seenMap m record.Symbol' true $$ [Hmap] with Hmap
        · iframe
        wp_apply wp_slice_literal (V := Access) [record]
        isplitr
        · ipureintro; rfl
        iintro %one ⟨Hone, -⟩
        wp_auto
        wp_apply wp_slice_append $$ [Hout Hcap Hone] with %sl' ⟨Hout, Hcap, -⟩
        · iframe
        wp_for_post
        iframe
        iexists (iv + W64 1), sl', (<[record.Symbol' := true]> m), (record.Symbol' :: keys), (picked ++ [record])
        iframe
        isplitr
        · ipureintro; constructor <;> word
        isplitr
        · ipureintro
          intro key
          by_cases hk : record.Symbol' = key
          · subst key; simp [GMap.lookup_insert]
          · simp [GMap.lookup_insert_eq_iff, hk, Hmem key, Ne.symm hk]
        ipureintro
        have hi : sint.nat (iv + W64 1) = sint.nat iv + 1 := by word
        rw [hi]
        simpa [selectFrom, hr, hs, List.append_assoc] using Hrest
  · try simp only [go.intoVal_inj.eq_iff, decide_eq_true_eq] at Hif
    try simp only [go.intoVal_inj.eq_iff, decide_eq_true_eq, decide_eq_false Hif, Bool.false_eq_true, ↓reduceIte]
    try wp_auto
    have hi : sint.nat iv = records.length := by word
    simp only [hi, List.drop_length, selectFrom, List.append_nil] at Hrest
    rw [Hrest]
    wp_end


theorem wp_SelectProperties (accesses : GoSlice) (records : List Access) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ accesses ↦* records }}
      (App (Val (@! Select)) (Val #accesses))
    {{ (out : GoSlice), RET #out;
      accesses ↦* records ∗ out ↦* selectFrom [] records ∗
      ⌜((selectFrom [] records).map Access.Symbol').Nodup ∧
        (selectFrom [] records).Sublist records ∧
        (∀ site ∈ selectFrom [] records, risky site = true) ∧
        (∀ key, key ∈ (selectFrom [] records).map Access.Symbol' ↔
          ∃ site ∈ records, risky site = true ∧ site.Symbol' = key)⌝ }} := by
  wp_start_folded as Haccesses
  wp_apply wp_Select accesses records $$ [Haccesses] with %out ⟨Haccesses, Hout⟩
  · iframe
  wp_end
  iframe
  ipureintro
  refine ⟨selected_unique [] records, selected_sublist [] records, ?_, ?_⟩
  · intro site hs
    exact selected_risky [] records site hs
  · intro key
    simpa using selected_keys [] records key



theorem wp_SelectNegative (accesses : GoSlice) (records : List Access)
    (site : Access) (x : Int) (hs : site ∈ records)
    (hi : site.Index' = true) (hu : site.UpperChecked' = true)
    (hx : contains site.Bounds' x) (hn : x < 0) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ accesses ↦* records }}
      (App (Val (@! Select)) (Val #accesses))
    {{ (out : GoSlice), RET #out;
      accesses ↦* records ∗ out ↦* selectFrom [] records ∗
      ⌜∃ found ∈ selectFrom [] records,
        found.Symbol' = site.Symbol' ∧ risky found = true⌝ }} := by
  wp_start_folded as Haccesses
  wp_apply wp_Select accesses records $$ [Haccesses] with %out ⟨Haccesses, Hout⟩
  · iframe
  wp_end
  iframe
  ipureintro
  exact selected_negative records site x hs hi hu hx hn



end proof
end github_com.rstyled.gstyle.domain.runtimeguard.interval
end Perennial
