module
public import Gstyle.Proof.Worklist.KnownComparison
public import Gstyle.Proof.Transfer.Assume
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

/-- Actual comparison pruning followed by interval refinement, including infeasible edges. -/
def successorResult [FfiSyntax] (op : GoString) (lv rv : Value) (truth : Bool)
    (step : StateData → Option StateData) (m : StateData) : Option StateData :=
  if (knownComparisonResult op lv rv).2 && decide ((knownComparisonResult op lv rv).1 ≠ truth)
  then none else step m

theorem successorResult_feasible [FfiSyntax] (op : GoString) (lv rv : Value) (truth : Bool)
    (step : StateData → Option StateData) (m : StateData) (x y : Int)
    (hl : interval.contains (bounds lv) x) (hr : interval.contains (bounds rv) y)
    (branch : comparisonBranch truth op x y) :
    successorResult op lv rv truth step m = step m := by
  unfold successorResult
  rw [knownComparisonResult_no_false_rejection op lv rv x y hl hr truth branch]
  rfl

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
variable [FuncUnfold booleanContradiction [] booleanContradiction.impl]
variable [FuncUnfold copyState [] copyState.impl]

/-- A missing decision keeps the actual state; it does not manufacture a branch. -/
theorem wp_successorState_zero (receiver : Loc) (s : QuerySnapshot) (truth : Bool)
    (state : Loc) (m : StateData) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"successorState" #receiver))
        (Val #(W32 0))) (Val #truth)) (Val #state))
    {{ RET #state; querySnapshotOwn receiver s ∗ state ↦$ m }} := by
  wp_start as ⟨#Hsnapshot, Hstate⟩
  try wp_auto
  wp_apply wp_attrs_zero receiver
  try wp_auto
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, Hgraph, _, _, _, _, _, _, _, _, _, _, _, _, _, Htable, Hstrings⟩
  try wp_auto
  wp_apply pdg.wp_Text s.graph.Tables' s.table s.strings DFrac.discard (zero_val pdg.Attributes).Construct'
      $$ [Htable Hstrings] with ⟨Htable, Hstrings⟩
  · iframe Htable Hstrings
  simp only [pdg.tableText]
  try wp_auto
  have hstatus : (zero_val pdg.Attributes).Operator'.Status' ≠ 1#8 := by
    change (0 : w8) ≠ 1#8
    decide
  wp_apply wp_operatorText_unavailable s.graph.Tables' (zero_val pdg.Attributes).Operator' hstatus
  try wp_auto
  wp_end
  all_goals iframe Hsnapshot Hstate

theorem wp_successorState_if_execution (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (sf : Nat) (ref left right : w32) (row : GoArray w32 2) (op : GoString) (lv rv : Value)
    (decision : w32) (decisionRow : GoArray w32 2)
    (hdecision : validSymbolRef s decision)
    (hdref : 0 < uint.Z decision ∧ uint.Z decision ≤ s.operands.length)
    (hdrow : s.operands[(uint.Z decision - 1).toNat]? = some decisionRow)
    (hdchild : decisionRow.arr[0]? = some ref)
    (hconstruct : pdg.tableText s.strings (snapshotRecord s decision).Construct' = go!"if")
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some left) (hright : row.arr[1]? = some right)
    (hvalid : validSymbolRef s ref)
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46)
    (htrue : snapshotFact s ref (go!"boolean") ≠ go!"true")
    (hfalse : snapshotFact s ref (go!"boolean") ≠ go!"false")
    (hop : decodedFact s (snapshotRecord s ref).Operator' = op)
    (hcomparison : comparisonOperator op)
    (wf : ValueSnapshotValid s)
    (hparse : parseInt64Contract (GF := GF))
    (leftEvaluation : ValueEvaluation s m sf left lv) (rightEvaluation : ValueEvaluation s m sf right rv)
    (truth : Bool)
    (hrefzero : ref ≠ W32 0)
    (o : AssumeOperators) (kind : w32 → GoString) (changing : w32 → Bool)
    (fuel : Nat) (step : StateData → Option StateData)
    (execution : FiniteAssume s o kind changing fuel ref truth step)
    (hm : validBranchState s.kinds s.unstable kind changing m) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗
       isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
       (querySnapshotOwn receiver s ∗ assumeOperatorsOwn o ∗ state ↦$ m) }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"successorState" #receiver))
        (Val #decision)) (Val #truth)) (Val #state))
    {{ (out : Loc), RET #out; querySnapshotOwn receiver s ∗ assumeOperatorsOwn o ∗
        ownsBranch (successorResult op lv rv truth step m) out ∗ ⌜validBranchOption s.kinds s.unstable kind changing (successorResult op lv rv truth step m)⌝ }} := by
  wp_start as ⟨#Hsnapshot, #Hoperators, Hstate⟩
  try wp_auto
  wp_apply wp_attrs_snapshot_full receiver s decision hdecision halloc
      $$ [Hsnapshot] with %attrs ⟨Hsnapshot, %hattrs⟩
  · iframe Hsnapshot
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, Hgraph, _, Hrows, _, _, _, _, _, _, _, _, _, _, _, Htable, Hstrings⟩
  try wp_auto
  have hfield : attrs.Construct' = (snapshotRecord s decision).Construct' :=
    congrArg pdg.Attributes.Construct' hattrs
  rw [hfield]
  wp_apply pdg.wp_Text s.graph.Tables' s.table s.strings DFrac.discard (snapshotRecord s decision).Construct'
      $$ [Htable Hstrings] with ⟨Htable, Hstrings⟩
  · iframe Htable Hstrings
  rw [hconstruct]
  try wp_auto
  wp_apply wp_operand_readonly receiver s.view s.operands decision (W64 0) decisionRow ref hdref (by decide) hdrow hdchild DFrac.discard DFrac.discard
      $$ [Hview Hrows] with ⟨Hview, Hrows⟩
  · iframe Hview Hrows
  try wp_auto
  rw [decide_eq_false hrefzero]
  try wp_auto
  wp_apply wp_knownBoolean_comparison receiver s state m sf ref left right row op lv rv
      href hrow hleft hright hvalid halloc htrue hfalse hop hcomparison wf.symbols wf.unstable hparse
      leftEvaluation rightEvaluation $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
  · iframe Hsnapshot Hstate
  try wp_auto
  cases hk : (knownComparisonResult op lv rv).2
  · have hresult : successorResult op lv rv truth step m = step m := by
      simp [successorResult, hk]
    rw [hresult]
    try wp_auto
    wp_apply wp_finite_assume receiver s o kind changing fuel ref truth step execution wf hparse state m hm
        $$ [Hsnapshot Hoperators Hstate] with %out ⟨Hsnapshot, Hoperators, Hout, %hvalidout⟩
    · iframe Hsnapshot Hoperators Hstate
    wp_end
  · try wp_auto
    by_cases hv : (knownComparisonResult op lv rv).1 = truth
    · have hresult : successorResult op lv rv truth step m = step m := by
        simp [successorResult, hk, hv]
      rw [hresult, hv]
      try wp_auto
      wp_apply wp_finite_assume receiver s o kind changing fuel ref truth step execution wf hparse state m hm
          $$ [Hsnapshot Hoperators Hstate] with %out ⟨Hsnapshot, Hoperators, Hout, %hvalidout⟩
      · iframe Hsnapshot Hoperators Hstate
      wp_end
    · have hresult : successorResult op lv rv truth step m = none := by
        simp [successorResult, hk, hv]
      rw [hresult, decide_eq_false hv]
      try wp_auto
      wp_end
      all_goals iframe Hsnapshot Hoperators
      all_goals simp only [ownsBranch, validBranchOption]
      all_goals ipureintro; simp


/-- A numeric comparison is not the synthetic tagless-switch predicate. -/

theorem comparisonOperator_not_truth_test (op : GoString) (h : comparisonOperator op) :
    op ≠ go!"truth-test" := by
  rcases h with rfl | rfl | rfl | rfl | rfl | rfl <;> decide

theorem wp_successorState_direct_execution (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (sf : Nat) (ref left right : w32) (row : GoArray w32 2) (op : GoString) (lv rv : Value)
    (hif : pdg.tableText s.strings (snapshotRecord s ref).Construct' ≠ go!"if")
    (hfor : pdg.tableText s.strings (snapshotRecord s ref).Construct' ≠ go!"for")
    (hshort : pdg.tableText s.strings (snapshotRecord s ref).Construct' ≠ go!"short-circuit")
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some left) (hright : row.arr[1]? = some right)
    (hvalid : validSymbolRef s ref)
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46)
    (htrue : snapshotFact s ref (go!"boolean") ≠ go!"true")
    (hfalse : snapshotFact s ref (go!"boolean") ≠ go!"false")
    (hop : decodedFact s (snapshotRecord s ref).Operator' = op)
    (hcomparison : comparisonOperator op)
    (wf : ValueSnapshotValid s)
    (hparse : parseInt64Contract (GF := GF))
    (leftEvaluation : ValueEvaluation s m sf left lv) (rightEvaluation : ValueEvaluation s m sf right rv)
    (truth : Bool)
    (hrefzero : ref ≠ W32 0)
    (o : AssumeOperators) (kind : w32 → GoString) (changing : w32 → Bool)
    (fuel : Nat) (step : StateData → Option StateData)
    (execution : FiniteAssume s o kind changing fuel ref truth step)
    (hm : validBranchState s.kinds s.unstable kind changing m) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗
       isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
       (querySnapshotOwn receiver s ∗ assumeOperatorsOwn o ∗ state ↦$ m) }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"successorState" #receiver))
        (Val #ref)) (Val #truth)) (Val #state))
    {{ (out : Loc), RET #out; querySnapshotOwn receiver s ∗ assumeOperatorsOwn o ∗
        ownsBranch (successorResult op lv rv truth step m) out ∗ ⌜validBranchOption s.kinds s.unstable kind changing (successorResult op lv rv truth step m)⌝ }} := by
  wp_start as ⟨#Hsnapshot, #Hoperators, Hstate⟩
  try wp_auto
  wp_apply wp_attrs_snapshot_full receiver s ref hvalid halloc
      $$ [Hsnapshot] with %attrs ⟨Hsnapshot, %hattrs⟩
  · iframe Hsnapshot
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, Hgraph, _, Hrows, _, _, _, _, _, _, _, _, _, _, _, Htable, Hstrings⟩
  try wp_auto
  have hfield : attrs.Construct' = (snapshotRecord s ref).Construct' :=
    congrArg pdg.Attributes.Construct' hattrs
  rw [hfield]
  wp_apply pdg.wp_Text s.graph.Tables' s.table s.strings DFrac.discard (snapshotRecord s ref).Construct'
      $$ [Htable Hstrings] with ⟨Htable, Hstrings⟩
  · iframe Htable Hstrings
  rw [decide_eq_false hshort]
  try wp_auto
  rw [decide_eq_false hfor]
  try wp_auto
  rw [decide_eq_false hif]
  try wp_auto
  have hoperator : attrs.Operator' = (snapshotRecord s ref).Operator' :=
    congrArg pdg.Attributes.Operator' hattrs
  rw [hoperator]
  wp_apply wp_operatorText_snapshot receiver s (snapshotRecord s ref).Operator' $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  rw [hop]
  try wp_auto
  rw [decide_eq_false (comparisonOperator_not_truth_test op hcomparison)]
  try wp_auto
  rw [decide_eq_false hrefzero]
  try wp_auto
  wp_apply wp_knownBoolean_comparison receiver s state m sf ref left right row op lv rv
      href hrow hleft hright hvalid halloc htrue hfalse hop hcomparison wf.symbols wf.unstable hparse
      leftEvaluation rightEvaluation $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
  · iframe Hsnapshot Hstate
  try wp_auto
  cases hk : (knownComparisonResult op lv rv).2
  · have hresult : successorResult op lv rv truth step m = step m := by
      simp [successorResult, hk]
    rw [hresult]
    try wp_auto
    wp_apply wp_finite_assume receiver s o kind changing fuel ref truth step execution wf hparse state m hm
        $$ [Hsnapshot Hoperators Hstate] with %out ⟨Hsnapshot, Hoperators, Hout, %hvalidout⟩
    · iframe Hsnapshot Hoperators Hstate
    wp_end
  · try wp_auto
    by_cases hv : (knownComparisonResult op lv rv).1 = truth
    · have hresult : successorResult op lv rv truth step m = step m := by
        simp [successorResult, hk, hv]
      rw [hresult, hv]
      try wp_auto
      wp_apply wp_finite_assume receiver s o kind changing fuel ref truth step execution wf hparse state m hm
          $$ [Hsnapshot Hoperators Hstate] with %out ⟨Hsnapshot, Hoperators, Hout, %hvalidout⟩
      · iframe Hsnapshot Hoperators Hstate
      wp_end
    · have hresult : successorResult op lv rv truth step m = none := by
        simp [successorResult, hk, hv]
      rw [hresult, decide_eq_false hv]
      try wp_auto
      wp_end
      all_goals iframe Hsnapshot Hoperators
      all_goals simp only [ownsBranch, validBranchOption]
      all_goals ipureintro; simp


end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
