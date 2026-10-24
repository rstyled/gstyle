module
public import Gstyle.Proof.Transfer.ComparisonModel
public import Gstyle.Proof.RuntimePackage
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

/-- Mathematical signed comparisons, including the production unknown-operator flag. -/
def exactOrderResult (op : GoString) (left right : w64) : Bool × Bool :=
  if op = go!"<" then (decide (sint.Z left < sint.Z right), true) else
  if op = go!"<=" then (decide (sint.Z left ≤ sint.Z right), true) else
  if op = go!">" then (decide (sint.Z left > sint.Z right), true) else
  if op = go!">=" then (decide (sint.Z left ≥ sint.Z right), true) else (false, false)

def exactComparisonResult (op : GoString) (left right : w64) : Bool × Bool :=
  if op = go!"==" then (decide (sint.Z left = sint.Z right), true) else
  if op = go!"!=" then (decide (sint.Z left ≠ sint.Z right), true) else exactOrderResult op left right

theorem exactComparisonResult_known (op : GoString) (left right : w64) :
    (exactComparisonResult op left right).2 = true ↔ comparisonOperator op := by
  by_cases hl : op = go!"<" <;> by_cases hle : op = go!"<=" <;>
    by_cases hg : op = go!">" <;> by_cases hge : op = go!">=" <;>
    by_cases he : op = go!"==" <;> by_cases hn : op = go!"!="
  all_goals simp [exactComparisonResult, exactOrderResult, comparisonOperator, hl, hle, hg, hge, he, hn]

theorem exactComparisonResult_truth (op : GoString) (left right : w64)
    (known : comparisonOperator op) :
    (exactComparisonResult op left right).1 = true ↔ integerRelation op (sint.Z left) (sint.Z right) := by
  rcases known with h | h | h | h | h | h
  all_goals subst op; simp [exactComparisonResult, exactOrderResult, integerRelation]

section execution
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics]
variable [FuncUnfold compareExactOrder [] compareExactOrder.impl]
variable [FuncUnfold compareExactIntegers [] compareExactIntegers.impl]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

theorem wp_compareExactOrder (op : GoString) (left right : w64) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (App (Val (@! compareExactOrder)) (Val #op)) (Val #left)) (Val #right))
    {{ RET (PairV #((exactOrderResult op left right).1) #((exactOrderResult op left right).2)); True }} := by
  wp_start
  by_cases hl : op = go!"<" <;> by_cases hle : op = go!"<=" <;>
    by_cases hg : op = go!">" <;> by_cases hge : op = go!">="
  all_goals iterate 7 (try simp [exactOrderResult, hl, hle, hg, hge]; try wp_auto)
  all_goals wp_end

theorem wp_compareExactIntegers (op : GoString) (left right : w64) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (App (Val (@! compareExactIntegers)) (Val #op)) (Val #left)) (Val #right))
    {{ RET (PairV #((exactComparisonResult op left right).1) #((exactComparisonResult op left right).2)); True }} := by
  wp_start
  have heq : (left = right) ↔ sint.Z left = sint.Z right := by word
  by_cases he : op = go!"==" <;> by_cases hn : op = go!"!="
  · subst op
    simp at hn
  · simp [exactComparisonResult, he, hn]
    wp_auto
    simp only [heq]
    wp_end
  · simp [exactComparisonResult, he, hn]
    wp_auto
    simp only [heq]
    wp_end
  · simp [exactComparisonResult, he, hn]
    iterate 4 (try wp_auto; try simp [he, hn])
    wp_apply wp_compareExactOrder op left right
    wp_end

end execution
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
