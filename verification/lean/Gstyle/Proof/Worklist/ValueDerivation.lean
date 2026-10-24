module
public import Gstyle.Proof.Worklist.ScalarRefinement
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [FfiSyntax]

/-- The execution witnesses already used by production value proofs also compute
    a finite resolver result. Fuel follows the derivation; it is not a depth cap. -/
theorem ValueEvaluation.resolve (s : QuerySnapshot) (m : StateData) (sf : Nat)
    (ref : w32) (v : Value) (evaluation : ValueEvaluation s m sf ref v) :
    ∃ fuel, valueResolve s m sf fuel ref = some v := by
  induction evaluation with
  | zero => exact ⟨1, by simp only [valueResolve, ite_true, ite_false, ne_eq, not_true_eq_false, not_false_eq_true, Bool.true_eq_false, Bool.false_eq_true, Option.getD_some, Option.map_some, Option.map_none, show (go!"conversion" : GoString) ≠ go!"parentheses" by decide]⟩
  | bound ref key kind changing href hresolve hk hkind hunstable =>
    have hz : ref ≠ W32 0 := by have := href.1; word
    have hk0 : key ≠ W32 0 := by have := hk.1; word
    exact ⟨1, by simp only [valueResolve, hz, hresolve, hk0, hk.2, hkind, hunstable, ite_true, ite_false, ne_eq, not_true_eq_false, not_false_eq_true, Bool.true_eq_false, Bool.false_eq_true, Option.getD_some, Option.map_some, Option.map_none, show (go!"conversion" : GoString) ≠ go!"parentheses" by decide]⟩
  | outOfRange ref key href hresolve hout =>
    have hz : ref ≠ W32 0 := by have := href.1; word
    have hk0 : key ≠ W32 0 := by word
    exact ⟨1, by simp only [valueResolve, hz, hresolve, hk0, show ¬uint.Z key ≤ s.kinds.length by omega, ite_true, ite_false, ne_eq, not_true_eq_false, not_false_eq_true, Bool.true_eq_false, Bool.false_eq_true, Option.getD_some, Option.map_some, Option.map_none, show (go!"conversion" : GoString) ≠ go!"parentheses" by decide]⟩
  | parentheses ref child row result href hresolve hop hrow hchild childEvaluation ih =>
    obtain ⟨fuel, hc⟩ := ih
    have hz : ref ≠ W32 0 := by have := href.1; word
    have hv : valueChild s ref = child := by simp only [valueChild, hrow, hchild, Option.getD_some]
    exact ⟨fuel + 1, by simp only [valueResolve, hz, hresolve, hop, hv, hc, ite_true, ite_false, ne_eq, not_true_eq_false, not_false_eq_true, Bool.true_eq_false, Bool.false_eq_true, Option.getD_some, Option.map_some, Option.map_none, show (go!"conversion" : GoString) ≠ go!"parentheses" by decide]⟩
  | conversionConstant ref href hresolve hop hinteger =>
    have hz : ref ≠ W32 0 := by have := href.1; word
    exact ⟨1, by simp only [valueResolve, hz, hresolve, hop, hinteger, ite_true, ite_false, ne_eq, not_true_eq_false, not_false_eq_true, Bool.true_eq_false, Bool.false_eq_true, Option.getD_some, Option.map_some, Option.map_none, show (go!"conversion" : GoString) ≠ go!"parentheses" by decide]⟩
  | conversionChild ref child row source href hresolve hop hinteger hrow hchild childEvaluation ih =>
    obtain ⟨fuel, hc⟩ := ih
    have hz : ref ≠ W32 0 := by have := href.1; word
    have hv : valueChild s ref = child := by simp only [valueChild, hrow, hchild, Option.getD_some]
    exact ⟨fuel + 1, by simp only [valueResolve, hz, hresolve, hop, hinteger, hv, hc, ite_true, ite_false, ne_eq, not_true_eq_false, not_false_eq_true, Bool.true_eq_false, Bool.false_eq_true, Option.getD_some, Option.map_some, Option.map_none, show (go!"conversion" : GoString) ≠ go!"parentheses" by decide]⟩
  | plain ref href hresolve hparent hconversion hlength =>
    have hz : ref ≠ W32 0 := by have := href.1; word
    exact ⟨1, by simp only [valueResolve, hz, hresolve, hparent, hconversion, hlength, ite_true, ite_false, ne_eq, not_true_eq_false, not_false_eq_true, Bool.true_eq_false, Bool.false_eq_true, Option.getD_some, Option.map_some, Option.map_none, show (go!"conversion" : GoString) ≠ go!"parentheses" by decide]⟩
  | lengthChild ref child row source href hresolve hparent hconversion hlength hrow hchild childEvaluation ih =>
    obtain ⟨fuel, hc⟩ := ih
    have hz : ref ≠ W32 0 := by have := href.1; word
    have hv : valueChild s ref = child := by simp only [valueChild, hrow, hchild, Option.getD_some]
    exact ⟨fuel + 1, by simp only [valueResolve, hz, hresolve, hparent, hconversion, hlength, hv, hc, ite_true, ite_false, ne_eq, not_true_eq_false, not_false_eq_true, Bool.true_eq_false, Bool.false_eq_true, Option.getD_some, Option.map_some, Option.map_none, show (go!"conversion" : GoString) ≠ go!"parentheses" by decide]⟩

/-- Numeric meaning follows the existing resolver contract for the same value
    that the actual execution witness returns. -/
theorem ValueEvaluation.denotes (s : QuerySnapshot) (kind : w32 → GoString)
    (changing : w32 → Bool) (m : StateData) (sf : Nat) (ref : w32) (v : Value)
    (evaluation : ValueEvaluation s m sf ref v) (env values : w32 → Int)
    (meaning : valueIntervalMeaning s kind changing env values)
    (hin : stateDenotes kind changing m env) : interval.contains (bounds v) (values ref) := by
  obtain ⟨fuel, hresolve⟩ := evaluation.resolve s m sf ref v
  exact meaning m sf fuel ref v hin hresolve

theorem snapshotConstant_identity (s : QuerySnapshot) (ref : w32) :
    (snapshotConstant s ref).Identity' = W64 0 := by
  unfold snapshotConstant constantRecord
  cases decimalInt64 (snapshotFact s ref (go!"integer")) <;>
    cases decimalInt64 (snapshotFact s ref (go!"literal-extent")) <;> rfl

theorem conversionResult_identity (kind : GoString) (source : Value) :
    (conversionResult kind source).Identity' = W64 0 := by
  unfold conversionResult
  split
  · rfl
  · split <;> rfl

theorem lengthCandidate_identity (s : QuerySnapshot) (ref : w32) :
    (lengthCandidate s ref).Identity' = W64 0 := snapshotConstant_identity s ref

theorem lengthResult_identity (v source : Value) (hv : v.Identity' = W64 0) :
    (lengthResult v source).Identity' = W64 0 := by
  unfold lengthResult
  split
  · rfl
  · exact hv

/-- Scalar source kinds and a scalar input map establish the result invariant;
    this replaces a separate assumed identity fact at scalar assignment sites. -/
theorem ValueEvaluation.scalar (s : QuerySnapshot) (m : StateData) (sf : Nat)
    (ref : w32) (v : Value) (evaluation : ValueEvaluation s m sf ref v)
    (hm : scalarState m) (hkinds : ∀ k ∈ s.kinds, k ≠ go!"\"pointer\"") :
    v.Identity' = W64 0 := by
  induction evaluation with
  | zero => rfl
  | bound ref key kind changing href hresolve hk hkind hunstable =>
    exact bindingValue_scalar kind changing key m hm (hkinds kind (List.mem_of_getElem? hkind))
  | outOfRange => rfl
  | parentheses ref child row result href hresolve hop hrow hchild childEvaluation ih => exact ih
  | conversionConstant ref href hresolve hop hinteger => exact snapshotConstant_identity s ref
  | conversionChild ref child row source href hresolve hop hinteger hrow hchild childEvaluation ih =>
    exact conversionResult_identity _ source
  | plain ref href hresolve hparent hconversion hlength => exact lengthCandidate_identity s ref
  | lengthChild ref child row source href hresolve hparent hconversion hlength hrow hchild childEvaluation ih =>
    exact lengthResult_identity _ source (lengthCandidate_identity s ref)

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
