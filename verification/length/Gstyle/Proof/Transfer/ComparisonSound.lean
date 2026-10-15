module
public import Gstyle.Proof.Transfer.ComparisonBoth
public import Gstyle.Proof.Transfer.Sound
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
section model
variable [FfiSyntax]

def stateRepresents (kind : w32 → GoString) (changing : w32 → Bool)
    (m : StateData) (valuation : w32 → Int) : Prop :=
  ∀ key, interval.contains (bounds (bindingValue (kind key) (changing key) key m)) (valuation key)

theorem exact_value_contains (b : Value) (x : Int)
    (hx : interval.contains (bounds b) x) (he : isExact b = true) : x = sint.Z b.Lower' := by
  have hl : b.HasLower' = true := by cases h : b.HasLower' <;> simp_all [isExact]
  have hu : b.HasUpper' = true := by cases h : b.HasUpper' <;> simp_all [isExact]
  have hb : b.Lower' = b.Upper' := by simpa [isExact, hl, hu] using he
  simp only [interval.contains, bounds] at hx
  have lower := hx.1 hl
  have upper := hx.2 hu
  rw [← hb] at upper
  omega

theorem integerRelation_satisfies (op : GoString) (left right : Int)
    (hrel : integerRelation op left right) : interval.satisfies op left right := by
  by_cases h : comparisonOperator op
  · rcases h with h | h | h | h | h | h
    all_goals subst op
    all_goals simpa [integerRelation, interval.satisfies] using hrel
  · simp only [comparisonOperator, not_or] at h
    rcases h with ⟨h1, h2, h3, h4, h5, h6⟩
    simp only [interval.satisfies, h1, h2, h3, h4, h5, ite_false]

theorem numeric_state_step_sound (kind : w32 → GoString) (changing : w32 → Bool)
    (m : StateData) (valuation : w32 → Int) (key : w32) (active : Bool) (op : GoString) (b : Value)
    (hs : stateRepresents kind changing m valuation)
    (hword : -9223372036854775808 ≤ valuation key ∧ valuation key ≤ 9223372036854775807)
    (hrel : isExact b = true → interval.satisfies op (valuation key) (sint.Z b.Lower')) :
    (active && constraintRejected (kind key) (changing key) key m op b) = false ∧
      stateRepresents kind changing (constraintState active (kind key) (changing key) key m op b) valuation := by
  have step := constraint_state_sound active (kind key) (changing key) key m op b (valuation key) (hs key) hword hrel
  refine ⟨step.1, fun other => ?_⟩
  by_cases he : key = other
  · subst other
    exact step.2
  · have lookup := constraint_preserves_other_keys active (kind key) (changing key) key other m op b he
    simpa only [bindingValue, lookup] using hs other

theorem comparison_numeric_state_sound (c : NumericComparison) (m : StateData)
    (kind : w32 → GoString) (changing : w32 → Bool) (valuation : w32 → Int)
    (hkl : c.leftKind = kind c.leftKey) (hkr : c.rightKind = kind c.rightKey)
    (hcl : c.leftChanging = changing c.leftKey) (hcr : c.rightChanging = changing c.rightKey)
    (hs : stateRepresents kind changing m valuation)
    (hlb : interval.contains (bounds c.leftValue) (valuation c.leftKey))
    (hrb : interval.contains (bounds c.rightValue) (valuation c.rightKey))
    (hwl : -9223372036854775808 ≤ valuation c.leftKey ∧ valuation c.leftKey ≤ 9223372036854775807)
    (hwr : -9223372036854775808 ≤ valuation c.rightKey ∧ valuation c.rightKey ≤ 9223372036854775807)
    (hleft : integerRelation c.operator (valuation c.leftKey) (valuation c.rightKey))
    (hright : integerRelation c.reversed (valuation c.rightKey) (valuation c.leftKey)) :
    comparisonRejected c m = false ∧ stateRepresents kind changing (comparisonResultMap c m) valuation := by
  have relLeft : isExact c.rightValue = true → interval.satisfies c.operator (valuation c.leftKey) (sint.Z c.rightValue.Lower') := by
    intro he
    have exactBound := exact_value_contains c.rightValue (valuation c.rightKey) hrb he
    rw [← exactBound]
    exact integerRelation_satisfies _ _ _ hleft
  have leftStep : comparisonLeftRejected c m = false ∧ stateRepresents kind changing (comparisonLeftMap c m) valuation := by
    simpa only [comparisonLeftRejected, comparisonLeftMap, hkl, hcl] using
      numeric_state_step_sound kind changing m valuation c.leftKey c.leftActive c.operator c.rightValue hs hwl relLeft
  have relRight : isExact c.leftValue = true → interval.satisfies c.reversed (valuation c.rightKey) (sint.Z c.leftValue.Lower') := by
    intro he
    have exactBound := exact_value_contains c.leftValue (valuation c.leftKey) hlb he
    rw [← exactBound]
    exact integerRelation_satisfies _ _ _ hright
  have rightStep : comparisonRightRejected c m = false ∧ stateRepresents kind changing (comparisonRightMap c m) valuation := by
    simpa only [comparisonRightRejected, comparisonRightMap, hkr, hcr] using
      numeric_state_step_sound kind changing (comparisonLeftMap c m) valuation c.rightKey c.rightActive c.reversed c.leftValue leftStep.2 hwr relRight
  constructor
  · simp only [comparisonRejected, leftStep.1, rightStep.1, Bool.false_or]
  · simpa only [comparisonResultMap, leftStep.1, Bool.false_eq_true, ite_false] using rightStep.2

theorem comparison_branch_state_sound (c : NumericComparison) (m : StateData) (truth : Bool) (rawOperator : GoString)
    (kind : w32 → GoString) (changing : w32 → Bool) (valuation : w32 → Int)
    (hkl : c.leftKind = kind c.leftKey) (hkr : c.rightKind = kind c.rightKey)
    (hcl : c.leftChanging = changing c.leftKey) (hcr : c.rightChanging = changing c.rightKey)
    (hs : stateRepresents kind changing m valuation)
    (hlb : interval.contains (bounds c.leftValue) (valuation c.leftKey))
    (hrb : interval.contains (bounds c.rightValue) (valuation c.rightKey))
    (hwl : -9223372036854775808 ≤ valuation c.leftKey ∧ valuation c.leftKey ≤ 9223372036854775807)
    (hwr : -9223372036854775808 ≤ valuation c.rightKey ∧ valuation c.rightKey ≤ 9223372036854775807)
    (hnormal : c.operator = normalizedOperator truth rawOperator)
    (hreverse : c.reversed = reverseOperator c.operator)
    (hbranch : comparisonBranch truth rawOperator (valuation c.leftKey) (valuation c.rightKey)) :
    comparisonRejected c m = false ∧ stateRepresents kind changing (comparisonResultMap c m) valuation := by
  have relations := normalizedBranchRelations truth rawOperator (valuation c.leftKey) (valuation c.rightKey) hbranch
  have leftRelation : integerRelation c.operator (valuation c.leftKey) (valuation c.rightKey) := by
    rw [hnormal]
    exact relations.1
  have rightRelation : integerRelation c.reversed (valuation c.rightKey) (valuation c.leftKey) := by
    rw [hreverse, hnormal]
    exact relations.2
  exact comparison_numeric_state_sound c m kind changing valuation hkl hkr hcl hcr hs hlb hrb hwl hwr leftRelation rightRelation

end model

open Iris.BI Iris.ProgramLogic
section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [graphView.TypeAssumptions] [pdg.query.Assumptions]
variable [pdg.Graph.TypeAssumptions] [pdg.Node.TypeAssumptions] [pdg.Attributes.TypeAssumptions]
variable [pdg.AttributeSet.TypeAssumptions] [pdg.AttributeTables.TypeAssumptions]
variable [pdg.Producer.TypeAssumptions] [pdg.Function.TypeAssumptions] [pdg.Span.TypeAssumptions]
variable [pdg.CountFact.TypeAssumptions] [pdg.Conformance.TypeAssumptions] [pdg.Fact.TypeAssumptions]
variable [pdg.Position.TypeAssumptions] [pdg.CallForm.TypeAssumptions] [pdg.ExitKind.TypeAssumptions]
variable [pdg.ConformanceStatus.TypeAssumptions] [pdg.Category.TypeAssumptions]
variable [FuncUnfold pdg.listAt [pdg.Ref.ty] (pdg.listAt.impl pdg.Ref.ty)]
variable [FuncUnfold pdg.listAt [pdg.Access.ty] (pdg.listAt.impl pdg.Access.ty)]
variable [FuncUnfold pdg.listAt [pdg.Extension.ty] (pdg.listAt.impl pdg.Extension.ty)]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

variable [pdg.Access.TypeAssumptions] [pdg.Tables.TypeAssumptions] [pdg.Extension.TypeAssumptions]
variable [FuncUnfold directBindingRead [] directBindingRead.impl]
variable [FuncUnfold operatorText [] operatorText.impl]


variable [FuncUnfold extension [] extension.impl]

variable [Value.TypeAssumptions] [State.TypeAssumptions]

variable [FuncUnfold integerConstraint [] integerConstraint.impl]
variable [FuncUnfold lengthConstraint [] lengthConstraint.impl]
variable [FuncUnfold exactInteger [] exactInteger.impl]
variable [FuncUnfold emptyRange [] emptyRange.impl]
variable [interval.Assumptions]

theorem wp_comparison_both_numeric_true_sound (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (ref left right : w32) (row : GoArray w32 2) (c : NumericComparison)
    (sf lf rf : Nat) (reverseLoc : Loc) (reverseTable : GMap GoString GoString) (wf : ValueSnapshotValid s)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some left) (hright : row.arr[1]? = some right)
    (hlvalid : validSymbolRef s left) (hrvalid : validSymbolRef s right)
    (hlvalue : valueResolve s m sf lf left = some c.leftValue)
    (hrvalue : valueResolve s m sf rf right = some c.rightValue)
    (hlsymbol : symbolResolve s sf left = some c.leftKey)
    (hrsymbol : symbolResolve s sf right = some c.rightKey)
    (hleftNonzero : c.leftKey ≠ W32 0) (hrightNonzero : c.rightKey ≠ W32 0)
    (hleftMetadata : ConstraintMetadata s c.leftKey c.leftKind c.leftChanging c.leftActive)
    (hrightMetadata : ConstraintMetadata s c.rightKey c.rightKind c.rightChanging c.rightActive)
    (ht : s.tracked.length = s.kinds.length)
    (hnill : c.leftValue.Nil' = false) (hnilr : c.rightValue.Nil' = false)
    (hreverse : (reverseTable !! c.operator).getD (zero_val GoString) = c.reversed)
    (kindOf : w32 → GoString) (changingOf : w32 → Bool) (valuation : w32 → Int)
    (hmetaKindLeft : c.leftKind = kindOf c.leftKey) (hmetaKindRight : c.rightKind = kindOf c.rightKey)
    (hmetaChangingLeft : c.leftChanging = changingOf c.leftKey) (hmetaChangingRight : c.rightChanging = changingOf c.rightKey)
    (hstateRep : stateRepresents kindOf changingOf m valuation)
    (hleftRep : interval.contains (bounds c.leftValue) (valuation c.leftKey))
    (hrightRep : interval.contains (bounds c.rightValue) (valuation c.rightKey))
    (hwordLeft : -9223372036854775808 ≤ valuation c.leftKey ∧ valuation c.leftKey ≤ 9223372036854775807)
    (hwordRight : -9223372036854775808 ≤ valuation c.rightKey ∧ valuation c.rightKey ≤ 9223372036854775807)
    (hcanonicalReverse : c.reversed = reverseOperator c.operator)
    (hbranchFact : comparisonBranch true (c.operator) (valuation c.leftKey) (valuation c.rightKey))
    (hparse : parseInt64Contract (GF := GF)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
      (globalAddr reverse ↦□ reverseLoc ∗ reverseLoc ↦${DFrac.discard} reverseTable ∗ querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"comparison" #receiver)) (Val #ref)) (Val #c.operator)) (Val #true)) (Val #state))
    {{ RET #state;
      (globalAddr reverse ↦□ reverseLoc ∗ reverseLoc ↦${DFrac.discard} reverseTable ∗
        querySnapshotOwn receiver s ∗ state ↦$ (comparisonResultMap c m)) ∗ ⌜stateRepresents kindOf changingOf (comparisonResultMap c m) valuation⌝ }} := by
  have semantic := comparison_branch_state_sound c m true (c.operator) kindOf changingOf valuation
    hmetaKindLeft hmetaKindRight hmetaChangingLeft hmetaChangingRight hstateRep hleftRep hrightRep hwordLeft hwordRight
    (by rfl) hcanonicalReverse hbranchFact
  wp_start_folded as Hresources
  wp_apply wp_comparison_both_numeric_true receiver s state m ref left right row c sf lf rf reverseLoc reverseTable wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hleftNonzero hrightNonzero hleftMetadata hrightMetadata ht hnill hnilr hreverse hparse $$ [Hresources] with Hresources
  · iexact Hresources
  rw [semantic.1]
  simp only [Bool.false_eq_true, ite_false]
  wp_end
  all_goals iframe Hresources
  all_goals ipureintro; exact semantic.2

theorem wp_comparison_both_numeric_false_sound (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (ref left right : w32) (row : GoArray w32 2) (c : NumericComparison)
    (sf lf rf : Nat) (rawOperator : GoString) (inverseLoc : Loc) (inverseTable : GMap GoString GoString) (reverseLoc : Loc) (reverseTable : GMap GoString GoString) (wf : ValueSnapshotValid s)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some left) (hright : row.arr[1]? = some right)
    (hlvalid : validSymbolRef s left) (hrvalid : validSymbolRef s right)
    (hlvalue : valueResolve s m sf lf left = some c.leftValue)
    (hrvalue : valueResolve s m sf rf right = some c.rightValue)
    (hlsymbol : symbolResolve s sf left = some c.leftKey)
    (hrsymbol : symbolResolve s sf right = some c.rightKey)
    (hleftNonzero : c.leftKey ≠ W32 0) (hrightNonzero : c.rightKey ≠ W32 0)
    (hleftMetadata : ConstraintMetadata s c.leftKey c.leftKind c.leftChanging c.leftActive)
    (hrightMetadata : ConstraintMetadata s c.rightKey c.rightKind c.rightChanging c.rightActive)
    (ht : s.tracked.length = s.kinds.length)
    (hnill : c.leftValue.Nil' = false) (hnilr : c.rightValue.Nil' = false)
    (hreverse : (reverseTable !! c.operator).getD (zero_val GoString) = c.reversed)
    (hnegate : (inverseTable !! rawOperator).getD (zero_val GoString) = c.operator)
    (kindOf : w32 → GoString) (changingOf : w32 → Bool) (valuation : w32 → Int)
    (hmetaKindLeft : c.leftKind = kindOf c.leftKey) (hmetaKindRight : c.rightKind = kindOf c.rightKey)
    (hmetaChangingLeft : c.leftChanging = changingOf c.leftKey) (hmetaChangingRight : c.rightChanging = changingOf c.rightKey)
    (hstateRep : stateRepresents kindOf changingOf m valuation)
    (hleftRep : interval.contains (bounds c.leftValue) (valuation c.leftKey))
    (hrightRep : interval.contains (bounds c.rightValue) (valuation c.rightKey))
    (hwordLeft : -9223372036854775808 ≤ valuation c.leftKey ∧ valuation c.leftKey ≤ 9223372036854775807)
    (hwordRight : -9223372036854775808 ≤ valuation c.rightKey ∧ valuation c.rightKey ≤ 9223372036854775807)
    (hcanonicalReverse : c.reversed = reverseOperator c.operator)
    (hcanonicalInverse : c.operator = inverseOperator rawOperator)
    (hbranchFact : comparisonBranch false (rawOperator) (valuation c.leftKey) (valuation c.rightKey))
    (hparse : parseInt64Contract (GF := GF)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
      (globalAddr inverse ↦□ inverseLoc ∗ inverseLoc ↦${DFrac.discard} inverseTable ∗ globalAddr reverse ↦□ reverseLoc ∗ reverseLoc ↦${DFrac.discard} reverseTable ∗ querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"comparison" #receiver)) (Val #ref)) (Val #rawOperator)) (Val #false)) (Val #state))
    {{ RET #state;
      (globalAddr inverse ↦□ inverseLoc ∗ inverseLoc ↦${DFrac.discard} inverseTable ∗ globalAddr reverse ↦□ reverseLoc ∗ reverseLoc ↦${DFrac.discard} reverseTable ∗
        querySnapshotOwn receiver s ∗ state ↦$ (comparisonResultMap c m)) ∗ ⌜stateRepresents kindOf changingOf (comparisonResultMap c m) valuation⌝ }} := by
  have semantic := comparison_branch_state_sound c m false (rawOperator) kindOf changingOf valuation
    hmetaKindLeft hmetaKindRight hmetaChangingLeft hmetaChangingRight hstateRep hleftRep hrightRep hwordLeft hwordRight
    (by simpa only [normalizedOperator, Bool.false_eq_true, ite_false] using hcanonicalInverse) hcanonicalReverse hbranchFact
  wp_start_folded as Hresources
  wp_apply wp_comparison_both_numeric_false receiver s state m ref left right row c sf lf rf rawOperator inverseLoc inverseTable reverseLoc reverseTable wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hleftNonzero hrightNonzero hleftMetadata hrightMetadata ht hnill hnilr hreverse hnegate hparse $$ [Hresources] with Hresources
  · iexact Hresources
  rw [semantic.1]
  simp only [Bool.false_eq_true, ite_false]
  wp_end
  all_goals iframe Hresources
  all_goals ipureintro; exact semantic.2

end proof

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
