module
public import Gstyle.Proof.Worklist.ValueDerivation
public import Gstyle.Proof.Worklist.Traversal
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

/-- Meaning of immutable facts, not a theorem about a child analyzer result.
    The source-lowering proof must establish this record for each concrete state. -/
structure ScalarMeaning (s : QuerySnapshot) (kind : w32 → GoString)
    (changing : w32 → Bool) (env : w32 → Int) where
  values : w32 → Int
  eval : w32 → Bool
  assume : AssumeMeaning s values eval
  valuesMeaning : valueIntervalMeaning s kind changing env values
  symbolsMeaning : symbolIntervalMeaning s env values
  words : ∀ key, -9223372036854775808 ≤ env key ∧ env key ≤ 9223372036854775807

def decisionCondition (s : QuerySnapshot) (ref : w32) : w32 :=
  if ref = W32 0 then W32 0 else
  let construct := pdg.tableText s.strings (snapshotRecord s ref).Construct'
  if construct = go!"if" ∨ construct = go!"for" ∨ construct = go!"short-circuit"
  then valueChild s ref else ref

def decisionEnabled (s : QuerySnapshot) (eval : w32 → Bool) (ref : w32) (truth : Bool) : Prop :=
  ref = W32 0 ∨ eval (decisionCondition s ref) = truth

theorem ComparisonEvidence.scalar (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool) (ref : w32)
    (c : ComparisonEvidence s o kind changing ref) (truth : Bool) (m : StateData)
    (hm : validBranchState s.kinds s.unstable kind changing m) (hs : scalarState m)
    (hk : ∀ key, kind key ≠ go!"\"pointer\"") : scalarOption (c.result truth m) := by
  unfold ComparisonEvidence.result successorResult
  split
  · trivial
  · exact finiteAssume_scalar s o kind changing c.fuel ref truth (c.step truth) (c.execution truth) hk m hm hs

theorem DecisionEvidence.scalar (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool) (ref : w32)
    (result : Bool → StateData → Option StateData) (evidence : DecisionEvidence s o kind changing ref result)
    (truth : Bool) (m : StateData) (hm : validBranchState s.kinds s.unstable kind changing m)
    (hs : scalarState m) (hk : ∀ key, kind key ≠ go!"\"pointer\"") : scalarOption (result truth m) := by
  cases evidence with
  | zero => exact hs
  | direct ref c _ _ _ => exact ComparisonEvidence.scalar s o kind changing ref c truth m hm hs hk
  | wrapped _ ref c _ _ _ _ _ _ => exact ComparisonEvidence.scalar s o kind changing ref c truth m hm hs hk

/-- Concrete truth prevents the exact-comparison fast path from discarding an
    environment; finiteAssume_sound supplies the actual interval refinement. -/
theorem ComparisonEvidence.sound (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool) (ref : w32)
    (c : ComparisonEvidence s o kind changing ref) (truth : Bool) (m : StateData)
    (hm : validBranchState s.kinds s.unstable kind changing m)
    (env : w32 → Int) (meaning : ScalarMeaning s kind changing env)
    (hin : stateDenotes kind changing m env) (htruth : meaning.eval ref = truth)
    (hinverse : ∀ op, (o.inverseTable !! op).getD (zero_val GoString) = inverseOperator op)
    (hreverse : ∀ op, (o.reverseTable !! op).getD (zero_val GoString) = reverseOperator op) :
    branchDenotes kind changing (c.result truth m) env := by
  have hl := ValueEvaluation.denotes s kind changing m c.sf c.left (c.leftValue m)
    (c.leftEvaluation m hm) env meaning.values meaning.valuesMeaning hin
  have hr := ValueEvaluation.denotes s kind changing m c.sf c.right (c.rightValue m)
    (c.rightEvaluation m hm) env meaning.values meaning.valuesMeaning hin
  have branch := meaning.assume.comparison ref c.operator truth c.hop htruth c.row c.left c.right
    c.hrow c.hleft c.hright
  have hfeasible := successorResult_feasible c.operator (c.leftValue m) (c.rightValue m) truth
    (c.step truth) m (meaning.values c.left) (meaning.values c.right) hl hr branch
  unfold ComparisonEvidence.result
  rw [hfeasible]
  exact finiteAssume_sound s o kind changing c.fuel ref truth (c.step truth) (c.execution truth)
    env meaning.values meaning.eval meaning.assume meaning.valuesMeaning meaning.symbolsMeaning meaning.words
    hinverse hreverse m hm hin htruth

theorem DecisionEvidence.sound (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool) (ref : w32)
    (result : Bool → StateData → Option StateData) (evidence : DecisionEvidence s o kind changing ref result)
    (truth : Bool) (m : StateData) (hm : validBranchState s.kinds s.unstable kind changing m)
    (env : w32 → Int) (meaning : ScalarMeaning s kind changing env)
    (hin : stateDenotes kind changing m env) (htruth : decisionEnabled s meaning.eval ref truth)
    (hinverse : ∀ op, (o.inverseTable !! op).getD (zero_val GoString) = inverseOperator op)
    (hreverse : ∀ op, (o.reverseTable !! op).getD (zero_val GoString) = reverseOperator op) :
    branchDenotes kind changing (result truth m) env := by
  cases evidence with
  | zero => exact hin
  | direct ref c hif hfor hshort =>
    have hn : ref ≠ W32 0 := by have := c.href.1; word
    have hc : decisionCondition s ref = ref := by
      simp only [decisionCondition, hn, hif, hfor, hshort, or_self, ite_false]
    have ht := htruth.resolve_left hn
    rw [hc] at ht
    exact ComparisonEvidence.sound s o kind changing ref c truth m hm env meaning hin ht hinverse hreverse
  | wrapped _ ref c row hvalid href hrow hchild hconstruct =>
    rename_i decision
    have hn : decision ≠ W32 0 := by have := href.1; word
    have hc : decisionCondition s decision = ref := by
      simp only [decisionCondition, hn, ite_false, hconstruct, _root_.true_or, ite_true,
        valueChild, hrow, Option.getD_some, hchild]
    have ht := htruth.resolve_left hn
    rw [hc] at ht
    exact ComparisonEvidence.sound s o kind changing ref c truth m hm env meaning hin ht hinverse hreverse

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
