module
public import Gstyle.Proof.Transfer.ComparisonSound
public import Gstyle.Proof.Transfer.ComparisonTransfers
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
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

theorem wp_comparison_left_numeric_true_sound (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (ref left right key : w32) (row : GoArray w32 2) (kind op : GoString) (lv rv : Value) (changing active : Bool)
    (sf lf rf : Nat) (wf : ValueSnapshotValid s)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some left) (hright : row.arr[1]? = some right)
    (hlvalid : validSymbolRef s left) (hrvalid : validSymbolRef s right)
    (hlvalue : valueResolve s m sf lf left = some lv)
    (hrvalue : valueResolve s m sf rf right = some rv)
    (hlsymbol : symbolResolve s sf left = some key)
    (hrsymbol : symbolResolve s sf right = some (W32 0))
    (hkey : key ≠ W32 0)
    (metadata : ConstraintMetadata s key kind changing active)
    (ht : s.tracked.length = s.kinds.length)
    (hnil : rv.Nil' = false)
    (kindOf : w32 → GoString) (changingOf : w32 → Bool) (valuation : w32 → Int) (other : Int)
    (hkindMeaning : kind = kindOf key) (hchangingMeaning : changing = changingOf key)
    (hstateRep : stateRepresents kindOf changingOf m valuation)
    (hword : -9223372036854775808 ≤ valuation key ∧ valuation key ≤ 9223372036854775807)
    (hboundRep : interval.contains (bounds rv) other)
    (hbranchFact : comparisonBranch true op (valuation key) other)
    (hparse : parseInt64Contract (GF := GF)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗ (querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"comparison" #receiver)) (Val #ref)) (Val #op)) (Val #true)) (Val #state))
    {{ RET #state;
      (querySnapshotOwn receiver s ∗ state ↦$ (constraintState active kind changing key m op rv)) ∗ ⌜stateRepresents kindOf changingOf (constraintState active kind changing key m op rv) valuation⌝ }} := by
  have relations : integerRelation (normalizedOperator true op) (valuation key) other ∧
      integerRelation (reverseOperator (normalizedOperator true op)) other (valuation key) := by
    try exact normalizedBranchRelations true op (valuation key) other hbranchFact
  have relation : integerRelation op (valuation key) other := by
    try simpa only [normalizedOperator, ite_true] using relations.1
  have semantic := numeric_bound_state_sound kindOf changingOf m valuation key active op rv other
    hstateRep hword hboundRep relation
  have sound : (active && constraintRejected kind changing key m op rv) = false ∧
      stateRepresents kindOf changingOf (constraintState active kind changing key m op rv) valuation := by
    try simpa only [← hkindMeaning, ← hchangingMeaning] using semantic
  wp_start_folded as Hresources
  wp_apply wp_comparison_left_numeric_true receiver s state m ref left right key row kind op lv rv changing active sf lf rf wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hkey metadata ht hnil hparse $$ [Hresources] with Hresources
  · iexact Hresources
  rw [sound.1]
  simp only [Bool.false_eq_true, ite_false]
  wp_end
  all_goals try iframe Hresources
  all_goals ipureintro; exact sound.2

theorem wp_comparison_left_numeric_false_sound (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (ref left right key : w32) (row : GoArray w32 2) (kind op : GoString) (lv rv : Value) (changing active : Bool)
    (sf lf rf : Nat) (inverseLoc : Loc) (inverseTable : GMap GoString GoString) (normal : GoString) (wf : ValueSnapshotValid s)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some left) (hright : row.arr[1]? = some right)
    (hlvalid : validSymbolRef s left) (hrvalid : validSymbolRef s right)
    (hlvalue : valueResolve s m sf lf left = some lv)
    (hrvalue : valueResolve s m sf rf right = some rv)
    (hlsymbol : symbolResolve s sf left = some key)
    (hrsymbol : symbolResolve s sf right = some (W32 0))
    (hkey : key ≠ W32 0)
    (metadata : ConstraintMetadata s key kind changing active)
    (ht : s.tracked.length = s.kinds.length)
    (hnil : rv.Nil' = false)
    (hlookup : (inverseTable !! op).getD (zero_val GoString) = normal)
    (kindOf : w32 → GoString) (changingOf : w32 → Bool) (valuation : w32 → Int) (other : Int)
    (hkindMeaning : kind = kindOf key) (hchangingMeaning : changing = changingOf key)
    (hstateRep : stateRepresents kindOf changingOf m valuation)
    (hword : -9223372036854775808 ≤ valuation key ∧ valuation key ≤ 9223372036854775807)
    (hboundRep : interval.contains (bounds rv) other)
    (hbranchFact : comparisonBranch false op (valuation key) other)
    (hcanonicalInverse : normal = inverseOperator op)
    (hparse : parseInt64Contract (GF := GF)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗ (globalAddr inverse ↦□ inverseLoc ∗ inverseLoc ↦${DFrac.discard} inverseTable ∗ querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"comparison" #receiver)) (Val #ref)) (Val #op)) (Val #false)) (Val #state))
    {{ RET #state;
      (globalAddr inverse ↦□ inverseLoc ∗ inverseLoc ↦${DFrac.discard} inverseTable ∗ querySnapshotOwn receiver s ∗ state ↦$ (constraintState active kind changing key m normal rv)) ∗ ⌜stateRepresents kindOf changingOf (constraintState active kind changing key m normal rv) valuation⌝ }} := by
  have relations : integerRelation (normalizedOperator false op) (valuation key) other ∧
      integerRelation (reverseOperator (normalizedOperator false op)) other (valuation key) := by
    try exact normalizedBranchRelations false op (valuation key) other hbranchFact
  have relation : integerRelation normal (valuation key) other := by
    try simpa only [normalizedOperator, Bool.false_eq_true, ite_false, ← hcanonicalInverse] using relations.1
  have semantic := numeric_bound_state_sound kindOf changingOf m valuation key active normal rv other
    hstateRep hword hboundRep relation
  have sound : (active && constraintRejected kind changing key m normal rv) = false ∧
      stateRepresents kindOf changingOf (constraintState active kind changing key m normal rv) valuation := by
    try simpa only [← hkindMeaning, ← hchangingMeaning] using semantic
  wp_start_folded as Hresources
  wp_apply wp_comparison_left_numeric_false receiver s state m ref left right key row kind op lv rv changing active sf lf rf inverseLoc inverseTable normal wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hkey metadata ht hnil hlookup hparse $$ [Hresources] with Hresources
  · iexact Hresources
  rw [sound.1]
  simp only [Bool.false_eq_true, ite_false]
  wp_end
  all_goals try iframe Hresources
  all_goals ipureintro; exact sound.2

theorem wp_comparison_right_numeric_true_sound (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (ref left right key : w32) (row : GoArray w32 2) (kind op : GoString) (lv rv : Value) (changing active : Bool)
    (sf lf rf : Nat) (reverseLoc : Loc) (reverseTable : GMap GoString GoString) (normal : GoString) (wf : ValueSnapshotValid s)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some left) (hright : row.arr[1]? = some right)
    (hlvalid : validSymbolRef s left) (hrvalid : validSymbolRef s right)
    (hlvalue : valueResolve s m sf lf left = some lv)
    (hrvalue : valueResolve s m sf rf right = some rv)
    (hlsymbol : symbolResolve s sf left = some (W32 0))
    (hrsymbol : symbolResolve s sf right = some key)
    (hkey : key ≠ W32 0)
    (metadata : ConstraintMetadata s key kind changing active)
    (ht : s.tracked.length = s.kinds.length)
    (hnil : lv.Nil' = false)
    (hlookup : (reverseTable !! op).getD (zero_val GoString) = normal)
    (kindOf : w32 → GoString) (changingOf : w32 → Bool) (valuation : w32 → Int) (other : Int)
    (hkindMeaning : kind = kindOf key) (hchangingMeaning : changing = changingOf key)
    (hstateRep : stateRepresents kindOf changingOf m valuation)
    (hword : -9223372036854775808 ≤ valuation key ∧ valuation key ≤ 9223372036854775807)
    (hboundRep : interval.contains (bounds lv) other)
    (hbranchFact : comparisonBranch true op other (valuation key))
    (hcanonicalReverse : normal = reverseOperator op)
    (hparse : parseInt64Contract (GF := GF)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗ (globalAddr reverse ↦□ reverseLoc ∗ reverseLoc ↦${DFrac.discard} reverseTable ∗ querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"comparison" #receiver)) (Val #ref)) (Val #op)) (Val #true)) (Val #state))
    {{ RET #state;
      (globalAddr reverse ↦□ reverseLoc ∗ reverseLoc ↦${DFrac.discard} reverseTable ∗ querySnapshotOwn receiver s ∗ state ↦$ (constraintState active kind changing key m normal lv)) ∗ ⌜stateRepresents kindOf changingOf (constraintState active kind changing key m normal lv) valuation⌝ }} := by
  have relations : integerRelation (normalizedOperator true op) other (valuation key) ∧
      integerRelation (reverseOperator (normalizedOperator true op)) (valuation key) other := by
    try exact normalizedBranchRelations true op other (valuation key) hbranchFact
  have relation : integerRelation normal (valuation key) other := by
    try simpa only [normalizedOperator, ite_true, ← hcanonicalReverse] using relations.2
  have semantic := numeric_bound_state_sound kindOf changingOf m valuation key active normal lv other
    hstateRep hword hboundRep relation
  have sound : (active && constraintRejected kind changing key m normal lv) = false ∧
      stateRepresents kindOf changingOf (constraintState active kind changing key m normal lv) valuation := by
    try simpa only [← hkindMeaning, ← hchangingMeaning] using semantic
  wp_start_folded as Hresources
  wp_apply wp_comparison_right_numeric_true receiver s state m ref left right key row kind op lv rv changing active sf lf rf reverseLoc reverseTable normal wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hkey metadata ht hnil hlookup hparse $$ [Hresources] with Hresources
  · iexact Hresources
  rw [sound.1]
  simp only [Bool.false_eq_true, ite_false]
  wp_end
  all_goals try iframe Hresources
  all_goals ipureintro; exact sound.2

theorem wp_comparison_right_numeric_false_sound (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (ref left right key : w32) (row : GoArray w32 2) (kind op : GoString) (lv rv : Value) (changing active : Bool)
    (sf lf rf : Nat) (reverseLoc : Loc) (reverseTable : GMap GoString GoString) (normal negated : GoString) (inverseLoc : Loc) (inverseTable : GMap GoString GoString) (wf : ValueSnapshotValid s)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some left) (hright : row.arr[1]? = some right)
    (hlvalid : validSymbolRef s left) (hrvalid : validSymbolRef s right)
    (hlvalue : valueResolve s m sf lf left = some lv)
    (hrvalue : valueResolve s m sf rf right = some rv)
    (hlsymbol : symbolResolve s sf left = some (W32 0))
    (hrsymbol : symbolResolve s sf right = some key)
    (hkey : key ≠ W32 0)
    (metadata : ConstraintMetadata s key kind changing active)
    (ht : s.tracked.length = s.kinds.length)
    (hnil : lv.Nil' = false)
    (hlookup : (reverseTable !! negated).getD (zero_val GoString) = normal)
    (hnegate : (inverseTable !! op).getD (zero_val GoString) = negated)
    (kindOf : w32 → GoString) (changingOf : w32 → Bool) (valuation : w32 → Int) (other : Int)
    (hkindMeaning : kind = kindOf key) (hchangingMeaning : changing = changingOf key)
    (hstateRep : stateRepresents kindOf changingOf m valuation)
    (hword : -9223372036854775808 ≤ valuation key ∧ valuation key ≤ 9223372036854775807)
    (hboundRep : interval.contains (bounds lv) other)
    (hbranchFact : comparisonBranch false op other (valuation key))
    (hcanonicalInverse : negated = inverseOperator op)
    (hcanonicalReverse : normal = reverseOperator negated)
    (hparse : parseInt64Contract (GF := GF)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗ (globalAddr inverse ↦□ inverseLoc ∗ inverseLoc ↦${DFrac.discard} inverseTable ∗ globalAddr reverse ↦□ reverseLoc ∗ reverseLoc ↦${DFrac.discard} reverseTable ∗ querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"comparison" #receiver)) (Val #ref)) (Val #op)) (Val #false)) (Val #state))
    {{ RET #state;
      (globalAddr inverse ↦□ inverseLoc ∗ inverseLoc ↦${DFrac.discard} inverseTable ∗ globalAddr reverse ↦□ reverseLoc ∗ reverseLoc ↦${DFrac.discard} reverseTable ∗ querySnapshotOwn receiver s ∗ state ↦$ (constraintState active kind changing key m normal lv)) ∗ ⌜stateRepresents kindOf changingOf (constraintState active kind changing key m normal lv) valuation⌝ }} := by
  have relations : integerRelation (normalizedOperator false op) other (valuation key) ∧
      integerRelation (reverseOperator (normalizedOperator false op)) (valuation key) other := by
    try exact normalizedBranchRelations false op other (valuation key) hbranchFact
  have relation : integerRelation normal (valuation key) other := by
    try simpa only [normalizedOperator, Bool.false_eq_true, ite_false, ← hcanonicalInverse, ← hcanonicalReverse] using relations.2
  have semantic := numeric_bound_state_sound kindOf changingOf m valuation key active normal lv other
    hstateRep hword hboundRep relation
  have sound : (active && constraintRejected kind changing key m normal lv) = false ∧
      stateRepresents kindOf changingOf (constraintState active kind changing key m normal lv) valuation := by
    try simpa only [← hkindMeaning, ← hchangingMeaning] using semantic
  wp_start_folded as Hresources
  wp_apply wp_comparison_right_numeric_false receiver s state m ref left right key row kind op lv rv changing active sf lf rf reverseLoc reverseTable normal negated inverseLoc inverseTable wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hkey metadata ht hnil hlookup hnegate hparse $$ [Hresources] with Hresources
  · iexact Hresources
  rw [sound.1]
  simp only [Bool.false_eq_true, ite_false]
  wp_end
  all_goals try iframe Hresources
  all_goals ipureintro; exact sound.2

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
