module
public import Gstyle.Proof.StateMap.Copy
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

theorem state_equal_of_size_and_entries [FfiSyntax] (a b : StateData)
    (hsize : GMap.size a = GMap.size b)
    (hentries : ∀ key value, a !! key = some value → b !! key = some value) : a = b := by
  apply GMap.map_subseteq_antisymm hentries
  intro key value hb
  cases ha : a !! key with
  | some other =>
    have same := hentries key other ha
    rw [hb] at same
    cases same
    rfl
  | none =>
    have hsub : (<[key := value]> a) ⊆ b := by
      intro k v hv
      by_cases hk : key = k
      · subst k
        simp only [GMap.lookup_insert_eq] at hv
        cases hv
        exact hb
      · simp only [GMap.lookup_insert_eq_iff, hk, ↓reduceIte] at hv
        exact hentries k v hv
    have hle := GMap.map_subseteq_size hsub
    rw [GMap.map_size_insert_None a key value ha, hsize] at hle
    omega

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [Value.TypeAssumptions] [State.TypeAssumptions] [pdg.query.Assumptions]
variable [FuncUnfold equalState [] equalState.impl]
local instance : DecidableEq StateData := Classical.typeDecidableEq _

omit [FfiInterp ffi] [FfiSemantics ext ffi] in
theorem map_body_return (P : IProp GF) (Φ : val → IProp GF) (v : val) :
    Φ (returnVal v) ⊢ forMapPostcondition P Φ (returnVal v) := by
  iintro HΦ
  unfold forMapPostcondition
  iright; iright; iright; ileft
  iexists v
  iframe
  ipureintro; rfl

theorem wp_equalState (left right : Loc) (a b : StateData) (ad bd : DFrac) :
    {{ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard ∗
       (left ↦${ad} a ∗ right ↦${bd} b) }}
      (App (App (Val (@! equalState)) (Val #left)) (Val #right))
    {{ RET #(decide (a = b)); left ↦${ad} a ∗ right ↦${bd} b }} := by
  wp_start as ⟨Hleft, Hright⟩
  try wp_auto
  wp_apply wp_map_len_lt left a ad $$ [Hleft] with ⟨Hleft, %Hasize⟩
  · iframe
  wp_apply wp_map_len_lt right b bd $$ [Hright] with ⟨Hright, %Hbsize⟩
  · iframe
  try wp_auto
  have hword : (W64 (GMap.size a) = W64 (GMap.size b)) ↔ GMap.size a = GMap.size b := by
    rw [← uint_Z_inj, uint_Z_W64 _ (by omega) (by omega), uint_Z_W64 _ (by omega) (by omega)]
    omega
  by_cases hsize : GMap.size a = GMap.size b
  · rw [decide_eq_true (hword.mpr hsize)]
    try wp_auto
    wp_func_lits
    wp_bind (App (App (Val (map.forRange _ _)) _) _)
    iapply (wp_map_for_range (fun keys i => iprop(∃ (sv : w32) (vv : Value),
      symbol_ptr ↦ sv ∗ value_ptr ↦ vv ∗ b_ptr ↦ right ∗ right ↦${bd} b ∗
      (left ↦${ad} a ∗ right ↦${bd} b -∗ Φ #(decide (a = b))) ∗
      ⌜0 ≤ i ∧ ∀ k, k ∈ keys.take i.toNat → a !! k = b !! k⌝))) $$ Hleft
    iintro %keys %Hkeys
    isplitl [symbol value b Hright HΦ]
    · iexists _, _
      iframe
      ipureintro
      simp
    isplitl []
    · imodintro
      iintro %i %key %value ⟨%Hkey, %Hlookup⟩ ⟨%sv, %vv, Hsymbol, Hvalue, Hbptr, Hright, HΦ, %Hi, %Hprefix⟩
      try wp_auto
      wp_apply wp_map_lookup2 pdg.Ref.ty Value.ty right b key bd $$ [Hright] with Hright
      · iframe
      cases hb : b !! key with
      | none =>
        have hne : a ≠ b := by intro h; subst b; rw [Hlookup] at hb; contradiction
        simp only [Option.isSome_none, Option.getD_none]
        try wp_auto
        iapply map_body_return
        iintro Hleft
        try wp_auto
        rw [decide_eq_false hne]
        iapply HΦ
        iframe
      | some other =>
        simp only [Option.isSome_some, _root_.decide_true, Option.getD_some]
        by_cases heq : other = value
        · subst other
          try wp_auto
          iapply map_body_done
          iexists key, value
          iframe
          ipureintro
          refine ⟨by omega, ?_⟩
          intro k hk
          have hn : (i + 1).toNat = i.toNat + 1 := by omega
          rw [hn, List.take_add_one, Hkey] at hk
          simp only [Option.toList_some, List.mem_append, List.mem_singleton] at hk
          rcases hk with hk | rfl
          · exact Hprefix k hk
          · exact Hlookup.trans hb.symm
        · have hne : a ≠ b := by
            intro h
            subst b
            rw [Hlookup] at hb
            exact heq (Option.some.inj hb).symm
          try wp_auto
          repeat (wp_if_destruct <;> try (solve | exfalso; apply heq; cases other; cases value; simp_all))
          all_goals
            iapply map_body_return
            iintro Hleft
            try wp_auto
            try rw [decide_eq_false hne]
            iapply HΦ
            iframe
    · iintro ⟨%sv, %vv, Hsymbol, Hvalue, Hbptr, Hright, HΦ, %Hi, %Hprefix⟩ Hleft
      have hsame : a = b := by
        apply state_equal_of_size_and_entries a b hsize
        intro key value ha
        have hk : key ∈ keys := (Hkeys.1 key).mpr (by simp [ha])
        have htake : keys.take (Int.toNat (GMap.size a : Int)) = keys := by rw [← Hkeys.2.1]; simp
        have hp := Hprefix key
        rw [htake] at hp
        exact (hp hk).symm.trans ha
      try wp_auto
      rw [decide_eq_true hsame]
      iapply HΦ
      iframe
  · have hwne : W64 (GMap.size a) ≠ W64 (GMap.size b) := fun h => hsize (hword.mp h)
    have hne : a ≠ b := by intro h; subst b; exact hsize rfl
    rw [decide_eq_false hwne]
    try wp_auto
    rw [decide_eq_false hne]
    wp_end

theorem wp_equalState_left_nil (right : Loc) (b : StateData) (bd : DFrac) :
    {{ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard ∗ right ↦${bd} b }}
      (App (App (Val (@! equalState)) (Val #map.nil)) (Val #right))
    {{ RET #(decide ((∅ : StateData) = b)); right ↦${bd} b }} := by
  wp_start as Hright
  try wp_auto
  wp_apply wp_map_len_lt right b bd $$ [Hright] with ⟨Hright, %Hbsize⟩
  · iframe
  have hword : W64 (GMap.size b) = W64 0 ↔ GMap.size b = 0 := by
    rw [← uint_Z_inj, uint_Z_W64 _ (by omega) (by omega), uint_Z_W64 _ (by omega) (by omega)]
    omega
  by_cases hb : b = ∅
  · subst b
    rw [GMap.map_size_empty]
    try wp_auto
    wp_func_lits
    try wp_auto
    rw [decide_eq_true (show (∅ : StateData) = ∅ from rfl)]
    wp_end
  · have hs : GMap.size b ≠ 0 := fun h => hb (GMap.map_size_empty_inv b h)
    have hw : W64 0 ≠ W64 (GMap.size b) := fun h => hs (hword.mp h.symm)
    try wp_auto
    rw [decide_eq_false hw]
    try wp_auto
    have he : (∅ : StateData) ≠ b := Ne.symm hb
    rw [decide_eq_false he]
    wp_end

theorem wp_equalState_right_nil (left : Loc) (a : StateData) (ad : DFrac) :
    {{ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard ∗ left ↦${ad} a }}
      (App (App (Val (@! equalState)) (Val #left)) (Val #map.nil))
    {{ RET #(decide (a = ∅)); left ↦${ad} a }} := by
  wp_start as Hleft
  try wp_auto
  wp_apply wp_map_len_lt left a ad $$ [Hleft] with ⟨Hleft, %Hasize⟩
  · iframe
  try wp_auto
  have hword : W64 (GMap.size a) = W64 0 ↔ GMap.size a = 0 := by
    rw [← uint_Z_inj, uint_Z_W64 _ (by omega) (by omega), uint_Z_W64 _ (by omega) (by omega)]
    omega
  by_cases ha : a = ∅
  · subst a
    rw [GMap.map_size_empty]
    try wp_auto
    wp_func_lits
    wp_bind (App (App (Val (map.forRange _ _)) _) _)
    iapply (wp_map_for_range (fun _ _ => iprop(True))) $$ Hleft
    iintro %keys %Hkeys
    have hkeys : keys = [] := List.length_eq_zero_iff.mp (by simpa [GMap.map_size_empty] using Hkeys.2.1)
    subst keys
    isplitl []
    · ipureintro; trivial
    isplitl []
    · imodintro
      iintro %i %key %value ⟨%Hkey, %Hlookup⟩ _
      simp at Hlookup
    · iintro _ Hleft
      try wp_auto
      rw [decide_eq_true (show (∅ : StateData) = ∅ from rfl)]
      wp_end
  · have hs : GMap.size a ≠ 0 := fun h => ha (GMap.map_size_empty_inv a h)
    have hw : W64 (GMap.size a) ≠ W64 0 := fun h => hs (hword.mp h)
    rw [decide_eq_false hw]
    try wp_auto
    rw [decide_eq_false ha]
    wp_end

theorem wp_equalState_both_nil :
    {{ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard }}
      (App (App (Val (@! equalState)) (Val #map.nil)) (Val #map.nil))
    {{ RET #true; True }} := by
  wp_start
  try wp_auto
  wp_func_lits
  try wp_auto
  wp_end

end proof

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
