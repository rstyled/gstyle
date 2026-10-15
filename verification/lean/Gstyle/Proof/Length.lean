module

public import Gstyle.GeneratedProof.github_com.rstyled.gstyle.domain.quality.linecount

@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.quality.linecount

/-- Physical line numbers newly covered by a source-ordered token span. -/
def freshLines (through first last : Nat) : List Nat :=
  List.range' (max first (through + 1)) (last + 1 - max first (through + 1))

/-- The specification counts exactly lines inside the token span and beyond
all previously covered lines. -/
theorem freshLines_mem (through first last n : Nat) :
    n ∈ freshLines through first last ↔ first ≤ n ∧ n ≤ last ∧ through < n := by
  simp only [freshLines, List.mem_range'_1]
  omega

/-- Every line in the contribution occurs once. -/
theorem freshLines_nodup (through first last : Nat) :
    (freshLines through first last).Nodup := by
  exact List.nodup_range' ..

/-- Its cardinality is the arithmetic quantity used in the Go proof. -/
theorem freshLines_length (through first last : Nat) :
    (freshLines through first last).length = last + 1 - max first (through + 1) := by
  simp [freshLines]

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext]
variable {hlc : HasLC} {GF : BundledGFunctors} [hG : HeapGS hlc GF]
variable [sem : go.Semantics] [package_sem : Assumptions]

instance isPkgInit_inst : IsPkgInit (IProp GF) pkg_id.github_com.rstyled.gstyle.domain.quality.linecount :=
  define_is_pkg_init iprop(True)
instance get_isPkgInit_wf_inst : GetIsPkgInitWf (IProp GF) pkg_id.github_com.rstyled.gstyle.domain.quality.linecount :=
  build_get_is_pkg_init_wf

/-- Complete boundary contract for the token-span accumulator. -/
def admissible (c : Counter) (first last : w64) : Prop :=
  0 ≤ sint.Z c.Lines' ∧ sint.Z c.Lines' ≤ sint.Z c.Through' ∧
  1 ≤ sint.Z first ∧ sint.Z first ≤ sint.Z last ∧ sint.Z c.Through' ≤ sint.Z first

instance (c : Counter) (first last : w64) : Decidable (admissible c first last) :=
  inferInstanceAs (Decidable (_ ∧ _ ∧ _ ∧ _ ∧ _))

/-- Every input check in the real Go implementation matches the boundary contract. -/
theorem wp_accepts (c : Counter) (first last : w64) :
    {{ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.quality.linecount }}
      (App (App (Val (c @!! Counter.ty @!! go!"accepts")) (Val #first)) (Val #last))
    {{ RET #(decide (admissible c first last)); True }} := by
  wp_start
  wp_auto
  by_cases h0 : 0 ≤ sint.Z c.Lines' <;>
    by_cases h1 : sint.Z c.Lines' ≤ sint.Z c.Through' <;>
    by_cases h2 : 1 ≤ sint.Z first <;>
    by_cases h3 : sint.Z first ≤ sint.Z last <;>
    by_cases h4 : sint.Z c.Through' ≤ sint.Z first
  all_goals iterate 6 (try simp [admissible, h0, h1, h2, h3, h4]; try wp_auto)
  all_goals wp_end

/-- Each accepted span adds exactly its previously uncovered lines. The result
stays nonnegative and never exceeds its last physical line, so arithmetic cannot
wrap even at the largest signed Go integer. -/
theorem wp_Add (c : Counter) (first last : w64) (h : admissible c first last) :
    {{ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.quality.linecount }}
      (App (App (Val (c @!! Counter.ty @!! go!"Add")) (Val #first)) (Val #last))
    {{ (r : Counter), RET (PairV #r #true);
      ⌜sint.Z r.Lines' = sint.Z c.Lines' +
          (sint.Z last - max (sint.Z first) (sint.Z c.Through' + 1) + 1) ∧
        r.Through' = last ∧ 0 ≤ sint.Z r.Lines' ∧ sint.Z r.Lines' ≤ sint.Z last⌝ }} := by
  wp_start
  wp_auto
  wp_apply wp_accepts
  simp only [show decide (admissible c first last) = true from decide_eq_true h]
  wp_auto
  obtain ⟨h0, h1, h2, h3, h4⟩ := h
  wp_if_destruct
  · wp_end
    ipureintro
    simp_all
    word
  · wp_if_destruct
    · wp_end
      ipureintro
      simp only [Int.max_eq_right (by omega : sint.Z c.Through' ≤ sint.Z c.Through' + 1)]
      word
    · wp_end
      ipureintro
      have hm : sint.Z c.Through' + 1 ≤ sint.Z first := by word
      simp only [Int.max_eq_left hm]
      word

/-- Invalid states or out-of-order spans are rejected without changing state. -/
theorem wp_Reject (c : Counter) (first last : w64) (h : ¬ admissible c first last) :
    {{ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.quality.linecount }}
      (App (App (Val (c @!! Counter.ty @!! go!"Add")) (Val #first)) (Val #last))
    {{ RET (PairV #c #false); True }} := by
  wp_start
  wp_auto
  wp_apply wp_accepts
  simp only [show decide (admissible c first last) = false from decide_eq_false h]
  wp_auto
  wp_end

/-- The production comparison is strict, over signed 64-bit Go integers. -/
theorem wp_Exceeds (actual limit : w64) :
    {{ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.quality.linecount }}
      (App (App (Val (@! Exceeds)) (Val #actual)) (Val #limit))
    {{ (result : Bool), RET #result; ⌜result = decide (sint.Z actual > sint.Z limit)⌝ }} := by
  wp_start
  wp_auto
  wp_end

end proof
end github_com.rstyled.gstyle.domain.quality.linecount
end Perennial
