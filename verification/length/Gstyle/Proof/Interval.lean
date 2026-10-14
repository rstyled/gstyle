module
public import Gstyle.GeneratedProof.github_com.rstyled.gstyle.domain.runtimeguard.interval
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard.interval

section model
variable [FfiSyntax]

def contains (r : Range) (x : Int) : Prop :=
  (r.HasLower' = true → sint.Z r.Lower' ≤ x) ∧
  (r.HasUpper' = true → x ≤ sint.Z r.Upper')

def joined (a b : Range) : Range :=
  { HasLower' := a.HasLower' && b.HasLower',
    HasUpper' := a.HasUpper' && b.HasUpper',
    Lower' := if a.HasLower' && b.HasLower' then
      (if sint.Z a.Lower' < sint.Z b.Lower' then a.Lower' else b.Lower') else W64 0,
    Upper' := if a.HasUpper' && b.HasUpper' then
      (if sint.Z a.Upper' > sint.Z b.Upper' then a.Upper' else b.Upper') else W64 0 }

theorem join_sound (a b : Range) (x : Int) :
    contains a x ∨ contains b x → contains (joined a b) x := by
  intro h
  rcases h with h | h
  all_goals
    cases ha : a.HasLower' <;> cases hb : b.HasLower' <;>
      cases hc : a.HasUpper' <;> cases hd : b.HasUpper' <;>
      by_cases hl : sint.Z a.Lower' < sint.Z b.Lower' <;>
      by_cases hu : sint.Z a.Upper' > sint.Z b.Upper'
  all_goals simp [contains, joined, ha, hb, hc, hd, hl, hu] at h ⊢ <;> omega

def risk (index upperChecked : Bool) (r : Range) : Bool :=
  index && upperChecked && Bool.or (!r.HasLower') (decide (sint.Z r.Lower' < 0))

theorem negative_detected (r : Range) (x : Int) (h : contains r x) (hx : x < 0) :
    risk true true r = true := by
  cases hl : r.HasLower'
  · simp [risk, hl]
  · have hn : sint.Z r.Lower' < 0 := by have := h.1 hl; omega
    simp [risk, hl, hn]


abbrev risky (site : Access) : Bool := risk site.Index' site.UpperChecked' site.Bounds'

def selectFrom (seen : List w32) : List Access → List Access
  | [] => []
  | site :: rest =>
    if risky site = true ∧ site.Symbol' ∉ seen then
      site :: selectFrom (site.Symbol' :: seen) rest
    else selectFrom seen rest

theorem selected_avoids (seen : List w32) (records : List Access) (site : Access)
    (h : site ∈ selectFrom seen records) : site.Symbol' ∉ seen := by
  induction records generalizing seen with
  | nil => simp [selectFrom] at h
  | cons head rest ih =>
    by_cases hc : risky head = true ∧ head.Symbol' ∉ seen
    · simp only [selectFrom, if_pos hc, List.mem_cons] at h
      rcases h with h | h
      · subst site; exact hc.2
      · have hn := ih (head.Symbol' :: seen) h
        intro hs
        exact hn (List.mem_cons_of_mem _ hs)
    · simp only [selectFrom, if_neg hc] at h
      exact ih seen h

theorem selected_risky (seen : List w32) (records : List Access) (site : Access)
    (h : site ∈ selectFrom seen records) : risky site = true := by
  induction records generalizing seen with
  | nil => simp [selectFrom] at h
  | cons head rest ih =>
    by_cases hc : risky head = true ∧ head.Symbol' ∉ seen
    · simp only [selectFrom, if_pos hc, List.mem_cons] at h
      rcases h with h | h
      · subst site; exact hc.1
      · exact ih _ h
    · simp only [selectFrom, if_neg hc] at h
      exact ih _ h

theorem selected_unique (seen : List w32) (records : List Access) :
    ((selectFrom seen records).map Access.Symbol').Nodup := by
  induction records generalizing seen with
  | nil => simp [selectFrom]
  | cons head rest ih =>
    by_cases hc : risky head = true ∧ head.Symbol' ∉ seen
    · simp only [selectFrom, if_pos hc, List.map_cons, List.nodup_cons]
      constructor
      · intro hm
        obtain ⟨site, hs, hk⟩ := List.mem_map.mp hm
        have hn := selected_avoids (head.Symbol' :: seen) rest site hs
        apply hn
        simp [hk]
      · exact ih _
    · simp only [selectFrom, if_neg hc]
      exact ih _

theorem selected_sublist (seen : List w32) (records : List Access) :
    (selectFrom seen records).Sublist records := by
  induction records generalizing seen with
  | nil => exact .refl _
  | cons head rest ih =>
    by_cases hc : risky head = true ∧ head.Symbol' ∉ seen
    · simp only [selectFrom, if_pos hc]
      exact (ih _).cons_cons _
    · simp only [selectFrom, if_neg hc]
      exact (ih _).cons _

theorem selected_keys (seen : List w32) (records : List Access) (key : w32) :
    key ∈ (selectFrom seen records).map Access.Symbol' ↔
      key ∉ seen ∧ ∃ site ∈ records, risky site = true ∧ site.Symbol' = key := by
  induction records generalizing seen with
  | nil => simp [selectFrom]
  | cons head rest ih =>
    by_cases hr : risky head = true <;> by_cases hs : head.Symbol' ∈ seen <;>
      by_cases hk : head.Symbol' = key
    all_goals have hkSymm : key = head.Symbol' ↔ head.Symbol' = key := eq_comm
    all_goals simp_all [selectFrom, ih, hkSymm]

end model

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics] [package_sem : Assumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval
instance isPkgInit_inst : IsPkgInit (IProp GF) pkg := define_is_pkg_init iprop(True)
instance get_isPkgInit_wf_inst : GetIsPkgInitWf (IProp GF) pkg := build_get_is_pkg_init_wf


theorem wp_minInt64 (x y : w64) :
    {{ (True : IProp GF) }}
      (App (App (Val #(functions go.min [go.int64, go.int64])) (Val #x)) (Val #y))
    {{ RET #(if sint.Z x < sint.Z y then x else y); True }} := by
  haveI : FuncUnfold go.min [go.int64, go.int64] (_root_.Perennial.min.impl go.int64 2) :=
    go.PredeclaredSemantics.min_unfold 2 go.int64
  wp_start_folded as _
  wp_func_call
  wp_call
  by_cases h : sint.Z x < sint.Z y
  all_goals iterate 3 (try simp [h]; try wp_auto)
  all_goals wp_end

theorem wp_maxInt64 (x y : w64) :
    {{ (True : IProp GF) }}
      (App (App (Val #(functions go.max [go.int64, go.int64])) (Val #x)) (Val #y))
    {{ RET #(if sint.Z x > sint.Z y then x else y); True }} := by
  haveI : FuncUnfold go.max [go.int64, go.int64] (_root_.Perennial.max.impl go.int64 2) :=
    go.PredeclaredSemantics.max_unfold 2 go.int64
  wp_start_folded as _
  wp_func_call
  wp_call
  by_cases h : sint.Z x > sint.Z y
  all_goals iterate 3 (try simp [h]; try wp_auto)
  all_goals wp_end

theorem wp_Join (a b : Range) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! Join)) (Val #a)) (Val #b))
    {{ RET #(joined a b); True }} := by
  wp_start
  wp_auto
  wp_apply wp_minInt64 a.Lower' b.Lower'
  wp_apply wp_maxInt64 a.Upper' b.Upper'
  cases ha : a.HasLower' <;> cases hb : b.HasLower' <;>
    cases hc : a.HasUpper' <;> cases hd : b.HasUpper'
  all_goals iterate 5 (try simp [joined, ha, hb, hc, hd]; try wp_auto)
  all_goals wp_end

theorem wp_NegativeIndex (index upperChecked : Bool) (r : Range) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (App (Val (@! NegativeIndex)) (Val #index)) (Val #upperChecked)) (Val #r))
    {{ RET #(risk index upperChecked r); True }} := by
  wp_start
  wp_auto
  cases index <;> cases upperChecked <;> cases hl : r.HasLower' <;>
    by_cases hn : sint.Z r.Lower' < 0
  all_goals iterate 5 (try simp [risk, hl, hn]; try wp_auto)
  all_goals wp_end


theorem wp_JoinSound (a b : Range) (x : Int)
    (h : contains a x ∨ contains b x) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! Join)) (Val #a)) (Val #b))
    {{ (r : Range), RET #r; ⌜contains r x⌝ }} := by
  wp_start_folded
  wp_apply wp_Join a b
  wp_end
  ipureintro
  exact join_sound a b x h

theorem wp_NegativeDetected (r : Range) (x : Int) (h : contains r x) (hx : x < 0) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (App (Val (@! NegativeIndex)) (Val #true)) (Val #true)) (Val #r))
    {{ RET #true; True }} := by
  wp_start_folded
  wp_apply wp_NegativeIndex true true r
  rw [negative_detected r x h hx]
  wp_end


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

end proof
end github_com.rstyled.gstyle.domain.runtimeguard.interval
end Perennial
