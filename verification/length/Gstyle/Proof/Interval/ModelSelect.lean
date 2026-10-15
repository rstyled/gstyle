module
public import Gstyle.Proof.Interval.ModelRisk
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard.interval

section model
variable [FfiSyntax]

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



theorem selected_negative (records : List Access) (site : Access) (x : Int)
    (hs : site ∈ records) (hi : site.Index' = true) (hu : site.UpperChecked' = true)
    (hx : contains site.Bounds' x) (hn : x < 0) :
    ∃ found ∈ selectFrom [] records, found.Symbol' = site.Symbol' ∧ risky found = true := by
  have hr : risky site = true := by
    unfold risky
    rw [hi, hu]
    exact negative_detected site.Bounds' x hx hn
  have hk : site.Symbol' ∈ (selectFrom [] records).map Access.Symbol' :=
    (selected_keys [] records site.Symbol').mpr ⟨by simp, site, hs, hr, rfl⟩
  obtain ⟨found, hf, he⟩ := List.mem_map.mp hk
  exact ⟨found, hf, he, selected_risky [] records found hf⟩



end model
end github_com.rstyled.gstyle.domain.runtimeguard.interval
end Perennial
