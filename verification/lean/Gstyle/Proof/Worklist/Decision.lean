module
public import Gstyle.Proof.Worklist.Successor
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

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


/-- Snapshot and finite-execution data used by the already-proved comparison body.
    This does not assert source meaning or assume a Hoare triple for a child call. -/
structure ComparisonEvidence (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool) (ref : w32) where
  sf : Nat
  left : w32
  right : w32
  row : GoArray w32 2
  operator : GoString
  leftValue : StateData → Value
  rightValue : StateData → Value
  href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length
  hrow : s.operands[(uint.Z ref - 1).toNat]? = some row
  hleft : row.arr[0]? = some left
  hright : row.arr[1]? = some right
  hvalid : validSymbolRef s ref
  htrue : snapshotFact s ref (go!"boolean") ≠ go!"true"
  hfalse : snapshotFact s ref (go!"boolean") ≠ go!"false"
  hop : decodedFact s (snapshotRecord s ref).Operator' = operator
  hcomparison : comparisonOperator operator
  leftEvaluation : ∀ m, validBranchState s.kinds s.unstable kind changing m →
    ValueEvaluation s m sf left (leftValue m)
  rightEvaluation : ∀ m, validBranchState s.kinds s.unstable kind changing m →
    ValueEvaluation s m sf right (rightValue m)
  fuel : Nat
  step : Bool → StateData → Option StateData
  execution : ∀ truth, FiniteAssume s o kind changing fuel ref truth (step truth)

def ComparisonEvidence.result (c : ComparisonEvidence s o kind changing ref)
    (truth : Bool) (m : StateData) : Option StateData :=
  successorResult c.operator (c.leftValue m) (c.rightValue m) truth (c.step truth) m

inductive DecisionEvidence (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool) :
    w32 → (Bool → StateData → Option StateData) → Prop where
  | zero : DecisionEvidence s o kind changing (W32 0) (fun _ m => some m)
  | direct (ref : w32) (c : ComparisonEvidence s o kind changing ref)
      (hif : pdg.tableText s.strings (snapshotRecord s ref).Construct' ≠ go!"if")
      (hfor : pdg.tableText s.strings (snapshotRecord s ref).Construct' ≠ go!"for")
      (hshort : pdg.tableText s.strings (snapshotRecord s ref).Construct' ≠ go!"short-circuit") :
      DecisionEvidence s o kind changing ref c.result
  | wrapped (decision ref : w32) (c : ComparisonEvidence s o kind changing ref)
      (row : GoArray w32 2)
      (hdecision : validSymbolRef s decision)
      (href : 0 < uint.Z decision ∧ uint.Z decision ≤ s.operands.length)
      (hrow : s.operands[(uint.Z decision - 1).toNat]? = some row)
      (hchild : row.arr[0]? = some ref)
      (hconstruct : pdg.tableText s.strings (snapshotRecord s decision).Construct' = go!"if") :
      DecisionEvidence s o kind changing decision c.result


theorem ComparisonEvidence.valid (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool) (ref : w32)
    (c : ComparisonEvidence s o kind changing ref) (truth : Bool) (m : StateData)
    (hm : validBranchState s.kinds s.unstable kind changing m) :
    validBranchOption s.kinds s.unstable kind changing (c.result truth m) := by
  unfold result successorResult
  split
  · trivial
  · exact finiteAssume_valid s o kind changing c.fuel ref truth (c.step truth) (c.execution truth) m hm

theorem DecisionEvidence.valid (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool) (ref : w32)
    (result : Bool → StateData → Option StateData)
    (evidence : DecisionEvidence s o kind changing ref result) (truth : Bool) (m : StateData)
    (hm : validBranchState s.kinds s.unstable kind changing m) :
    validBranchOption s.kinds s.unstable kind changing (result truth m) := by
  cases evidence with
  | zero => exact hm
  | direct ref c _ _ _ => exact ComparisonEvidence.valid s o kind changing ref c truth m hm
  | wrapped _ ref c _ _ _ _ _ _ => exact ComparisonEvidence.valid s o kind changing ref c truth m hm

/-- Every decision derivation executes the actual production dispatcher and refinement. -/
theorem wp_decision (receiver : Loc) (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool) (ref : w32)
    (result : Bool → StateData → Option StateData)
    (evidence : DecisionEvidence s o kind changing ref result)
    (wf : ValueSnapshotValid s) (hparse : parseInt64Contract (GF := GF))
    (truth : Bool) (state : Loc) (m : StateData)
    (hm : validBranchState s.kinds s.unstable kind changing m) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗
       isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
       (querySnapshotOwn receiver s ∗ assumeOperatorsOwn o ∗ state ↦$ m) }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"successorState" #receiver))
        (Val #ref)) (Val #truth)) (Val #state))
    {{ (out : Loc), RET #out; querySnapshotOwn receiver s ∗ assumeOperatorsOwn o ∗
        ownsBranch (result truth m) out ∗ ⌜validBranchOption s.kinds s.unstable kind changing (result truth m)⌝ }} := by
  cases evidence with
  | zero =>
    wp_start_folded as ⟨Hsnapshot, Hoperators, Hstate⟩
    wp_apply wp_successorState_zero receiver s truth state m $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
    · iframe Hsnapshot Hstate
    wp_end
    all_goals simp only [ownsBranch]
    all_goals iframe Hsnapshot Hoperators Hstate
    all_goals ipureintro; exact hm
  | direct ref c hif hfor hshort =>
    have hz : ref ≠ W32 0 := by have hp := c.href.1; word
    exact wp_successorState_direct_execution receiver s state m c.sf ref c.left c.right c.row c.operator
      (c.leftValue m) (c.rightValue m) hif hfor hshort c.href c.hrow c.hleft c.hright c.hvalid
      wf.symbols.allocation c.htrue c.hfalse c.hop c.hcomparison wf hparse
      (c.leftEvaluation m hm) (c.rightEvaluation m hm) truth hz o kind changing c.fuel (c.step truth)
      (c.execution truth) hm
  | wrapped _ ref c row hd hr hrow hc hconstruct =>
    rename_i decision
    have hz : ref ≠ W32 0 := by have hp := c.href.1; word
    exact wp_successorState_if_execution receiver s state m c.sf ref c.left c.right c.row c.operator
      (c.leftValue m) (c.rightValue m) decision row hd hr hrow hc hconstruct
      c.href c.hrow c.hleft c.hright c.hvalid wf.symbols.allocation c.htrue c.hfalse c.hop c.hcomparison wf hparse
      (c.leftEvaluation m hm) (c.rightEvaluation m hm) truth hz o kind changing c.fuel (c.step truth)
      (c.execution truth) hm

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
