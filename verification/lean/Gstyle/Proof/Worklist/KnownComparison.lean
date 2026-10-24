module
public import Gstyle.Proof.Worklist.ExactComparison
public import Gstyle.Proof.Transfer.ValueEvaluation
public import Gstyle.Proof.Transfer.NumericPrimitives
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [FfiSyntax]

/-- Only two proved singleton intervals permit pruning on an evaluated comparison. -/
def knownComparisonResult (op : GoString) (left right : Value) : Bool × Bool :=
  if isExact left && isExact right then exactComparisonResult op left.Lower' right.Lower'
  else (false, false)

theorem exactValue_denotes (v : Value) (x : Int)
    (contains : interval.contains (bounds v) x) (exact : isExact v = true) :
    x = sint.Z v.Lower' := by
  simp only [isExact, Bool.and_eq_true, decide_eq_true_eq, and_assoc] at exact
  obtain ⟨hl, hu, he⟩ := exact
  simp [interval.contains, bounds, hl, hu, ← he] at contains
  omega

theorem knownComparisonResult_correct (op : GoString) (left right : Value) (x y : Int)
    (hl : interval.contains (bounds left) x) (hr : interval.contains (bounds right) y)
    (known : (knownComparisonResult op left right).2 = true) :
    comparisonOperator op ∧ ((knownComparisonResult op left right).1 = true ↔ integerRelation op x y) := by
  by_cases hex : (isExact left && isExact right) = true
  · have he := Bool.and_eq_true_iff.mp hex
    have hx := exactValue_denotes left x hl he.1
    have hy := exactValue_denotes right y hr he.2
    simp only [knownComparisonResult, hex, ite_true] at known ⊢
    have hop := (exactComparisonResult_known op left.Lower' right.Lower').mp known
    exact ⟨hop, by simpa [hx, hy] using exactComparisonResult_truth op left.Lower' right.Lower' hop⟩
  · simp [knownComparisonResult, hex] at known

theorem knownComparisonResult_keeps_feasible (op : GoString) (left right : Value) (x y : Int)
    (hl : interval.contains (bounds left) x) (hr : interval.contains (bounds right) y)
    (truth : Bool) (branch : comparisonBranch truth op x y)
    (known : (knownComparisonResult op left right).2 = true) :
    (knownComparisonResult op left right).1 = truth := by
  obtain ⟨hop, htruth⟩ := knownComparisonResult_correct op left right x y hl hr known
  cases truth
  · have hn : ¬integerRelation op x y := branch hop
    have hf : (knownComparisonResult op left right).1 ≠ true := fun ht => hn (htruth.mp ht)
    cases hv : (knownComparisonResult op left right).1 <;> simp_all
  · exact htruth.mpr (branch hop)

theorem knownComparisonResult_no_false_rejection (op : GoString) (left right : Value) (x y : Int)
    (hl : interval.contains (bounds left) x) (hr : interval.contains (bounds right) y)
    (truth : Bool) (branch : comparisonBranch truth op x y) :
    ((knownComparisonResult op left right).2 &&
      decide ((knownComparisonResult op left right).1 ≠ truth)) = false := by
  cases hk : (knownComparisonResult op left right).2
  · simp [hk]
  · have ht := knownComparisonResult_keeps_feasible op left right x y hl hr truth branch hk
    simp [hk, ht]

theorem comparisonOperator_dispatch (op : GoString) (hop : comparisonOperator op) :
    op ≠ go!"parentheses" ∧ op ≠ go!"!" ∧ op ≠ go!"&&" ∧ op ≠ go!"||" := by
  rcases hop with rfl | rfl | rfl | rfl | rfl | rfl <;> decide

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

variable [FuncUnfold exactInteger [] exactInteger.impl]
variable [FuncUnfold integerConstraint [] integerConstraint.impl]
variable [FuncUnfold lengthConstraint [] lengthConstraint.impl]
variable [FuncUnfold emptyRange [] emptyRange.impl]
variable [interval.Assumptions]
variable [FuncUnfold compareExactOrder [] compareExactOrder.impl]
variable [FuncUnfold compareExactIntegers [] compareExactIntegers.impl]

/-- Both value calls are discharged by production execution witnesses, not assumed callbacks. -/
theorem wp_knownComparison (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (sf : Nat) (ref left right : w32) (row : GoArray w32 2) (op : GoString) (lv rv : Value)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some left) (hright : row.arr[1]? = some right)
    (wf : SymbolSnapshotValid s) (hulen : s.unstable.length = s.kinds.length)
    (hparse : parseInt64Contract (GF := GF))
    (leftEvaluation : ValueEvaluation s m sf left lv) (rightEvaluation : ValueEvaluation s m sf right rv) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"knownComparison" #receiver))
        (Val #ref)) (Val #op)) (Val #state))
    {{ RET (PairV #((knownComparisonResult op lv rv).1) #((knownComparisonResult op lv rv).2));
        querySnapshotOwn receiver s ∗ state ↦$ m }} := by
  wp_start as ⟨#Hsnapshot, Hstate⟩
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, _, _, Hrows, _, _, _, _, _, _, _, _, _, _, _, _, _⟩
  try wp_auto
  wp_apply wp_operand_readonly receiver s.view s.operands ref (W64 0) row left href (by decide) hrow hleft DFrac.discard DFrac.discard
      $$ [Hview Hrows] with ⟨Hview, Hrows⟩
  · iframe Hview Hrows
  wp_apply wp_value_evaluation receiver s state m sf left lv wf hulen hparse leftEvaluation
      $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
  · iframe Hsnapshot Hstate
  wp_apply wp_operand_readonly receiver s.view s.operands ref (W64 1) row right href (by decide) hrow hright DFrac.discard DFrac.discard
      $$ [Hview Hrows] with ⟨Hview, Hrows⟩
  · iframe Hview Hrows
  wp_apply wp_value_evaluation receiver s state m sf right rv wf hulen hparse rightEvaluation
      $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
  · iframe Hsnapshot Hstate
  wp_apply wp_exactInteger lv
  cases hl : isExact lv
  · simp [knownComparisonResult, hl]
    try wp_auto
    wp_end
  · simp [knownComparisonResult, hl]
    try wp_auto
    wp_apply wp_exactInteger rv
    cases hr : isExact rv
    · simp [hr]
      try wp_auto
      wp_end
    · simp [hr]
      try wp_auto
      wp_apply wp_compareExactIntegers op lv.Lower' rv.Lower'
      wp_end


theorem wp_knownBoolean_comparison (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (sf : Nat) (ref left right : w32) (row : GoArray w32 2) (op : GoString) (lv rv : Value)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some left) (hright : row.arr[1]? = some right)
    (hvalid : validSymbolRef s ref)
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46)
    (htrue : snapshotFact s ref (go!"boolean") ≠ go!"true")
    (hfalse : snapshotFact s ref (go!"boolean") ≠ go!"false")
    (hop : decodedFact s (snapshotRecord s ref).Operator' = op)
    (hcomparison : comparisonOperator op)
    (wf : SymbolSnapshotValid s) (hulen : s.unstable.length = s.kinds.length)
    (hparse : parseInt64Contract (GF := GF))
    (leftEvaluation : ValueEvaluation s m sf left lv) (rightEvaluation : ValueEvaluation s m sf right rv) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"knownBoolean" #receiver))
        (Val #ref)) (Val #state))
    {{ RET (PairV #((knownComparisonResult op lv rv).1) #((knownComparisonResult op lv rv).2));
        querySnapshotOwn receiver s ∗ state ↦$ m }} := by
  wp_start as ⟨#Hsnapshot, Hstate⟩
  try wp_auto
  wp_apply wp_fact_snapshot receiver s ref (go!"boolean") hvalid halloc $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  try wp_auto
  rw [show decide (snapshotFact s ref (go!"boolean") = go!"true") = false from decide_eq_false htrue]
  try wp_auto
  rw [show decide (snapshotFact s ref (go!"boolean") = go!"false") = false from decide_eq_false hfalse]
  try wp_auto
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, Hgraph, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _⟩
  try wp_auto
  wp_apply wp_attrs_snapshot receiver s ref hvalid halloc $$ [Hsnapshot] with %attrs ⟨Hsnapshot, %hattrs⟩
  · iframe Hsnapshot
  try wp_auto
  rw [hattrs.1]
  wp_apply wp_operatorText_snapshot receiver s (snapshotRecord s ref).Operator' $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  rw [hop]
  obtain ⟨hp, hn, ha, ho⟩ := comparisonOperator_dispatch op hcomparison
  try wp_auto
  rw [decide_eq_false hp]
  try wp_auto
  rw [decide_eq_false hn]
  try wp_auto
  rw [decide_eq_false ho]
  try wp_auto
  rw [decide_eq_false ha]
  try wp_auto
  wp_apply wp_knownComparison receiver s state m sf ref left right row op lv rv
      href hrow hleft hright wf hulen hparse leftEvaluation rightEvaluation
      $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
  · iframe Hsnapshot Hstate
  wp_end


end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
