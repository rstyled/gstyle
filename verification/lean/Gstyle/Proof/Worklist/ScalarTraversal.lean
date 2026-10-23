module
public import Gstyle.Proof.Worklist.InputCoverage
public import Gstyle.Proof.Worklist.FlowLayout
public import Gstyle.Proof.Worklist.Layout
public import Gstyle.Proof.Worklist.StepModel
public import Gstyle.Proof.StateMap.Equality
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


variable [analysis.TypeAssumptions] [pdg.NormalFlow.TypeAssumptions] [pdg.FlowBlock.TypeAssumptions]
variable [FuncUnfold equalState [] equalState.impl]


def scalarWorkInvariant (s : QuerySnapshot) (kind : w32 → GoString)
    (changing : w32 → Bool) (cursor : Nat) (state : WorkState) : Prop :=
  QueueInvariant state.flags cursor state.queue ∧ validWorkInputs s kind changing state ∧ scalarInputs state

def sourceEdgesCovered (s : QuerySnapshot) (kind : w32 → GoString) (changing : w32 → Bool)
    (condition : w32) (nexts : List w32) (eval : w32 → Bool) (env : w32 → Int) (out : WorkState) : Prop :=
  ∀ i next, nexts[i]? = some next →
    decisionEnabled s eval condition (decide (i = 0)) →
    branchDenotes kind changing (out.input (queueIndex next)) env

/-- The actual dispatcher preserves the scalar invariant and every concrete
    environment whose truth selects this edge. All execution calls are discharged. -/
theorem wp_decision_scalar_sound (receiver : Loc) (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool) (ref : w32)
    (result : Bool → StateData → Option StateData)
    (decision : DecisionEvidence s o kind changing ref result)
    (wf : ValueSnapshotValid s) (hparse : parseInt64Contract (GF := GF))
    (truth : Bool) (state : Loc) (m : StateData)
    (hm : validBranchState s.kinds s.unstable kind changing m) (hs : scalarState m)
    (hk : ∀ key, kind key ≠ go!"\"pointer\"")
    (env : w32 → Int) (meaning : ScalarMeaning s kind changing env)
    (hin : stateDenotes kind changing m env) (htruth : decisionEnabled s meaning.eval ref truth)
    (hinverse : ∀ op, (o.inverseTable !! op).getD (zero_val GoString) = inverseOperator op)
    (hreverse : ∀ op, (o.reverseTable !! op).getD (zero_val GoString) = reverseOperator op) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗
       isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
       (querySnapshotOwn receiver s ∗ assumeOperatorsOwn o ∗ state ↦$ m) }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"successorState" #receiver))
        (Val #ref)) (Val #truth)) (Val #state))
    {{ (out : Loc), RET #out; querySnapshotOwn receiver s ∗ assumeOperatorsOwn o ∗
        ownsBranch (result truth m) out ∗
        ⌜validBranchOption s.kinds s.unstable kind changing (result truth m) ∧
          scalarOption (result truth m) ∧ branchDenotes kind changing (result truth m) env⌝ }} := by
  have hscalar := DecisionEvidence.scalar s o kind changing ref result decision truth m hm hs hk
  have hsound := DecisionEvidence.sound s o kind changing ref result decision truth m hm env meaning
    hin htruth hinverse hreverse
  wp_start_folded as Hinputs
  wp_apply wp_decision receiver s o kind changing ref result decision wf hparse truth state m hm
      $$ [Hinputs] with %out ⟨Hsnapshot, Hoperators, Hout, %hvalid⟩
  · iexact Hinputs
  iapply HΦ
  iframe Hsnapshot Hoperators Hout
  ipureintro
  exact ⟨hvalid, hscalar, hsound⟩

/-- Semantic composition for the entire real successor loop. It preserves old
    represented environments and adds each feasible edge, including repeated
    targets, while maintaining scalar, metadata and pending-queue invariants. -/
theorem wp_successors_scalar_sound (receiver view state : Loc) (s : QuerySnapshot)
    (flow : pdg.NormalFlow) (blocks : List pdg.FlowBlock) (block : pdg.FlowBlock)
    (nexts : List w32) (ref : w32) (m : StateData)
    (inputs flags queue : GoSlice) (pointers : List Loc) (initial : WorkState)
    (o : AssumeOperators) (kind : w32 → GoString) (changing : w32 → Bool)
    (result : Bool → StateData → Option StateData)
    (decision : DecisionEvidence s o kind changing block.Condition' result)
    (wf : ValueSnapshotValid s) (hparse : parseInt64Contract (GF := GF))
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ blocks.length)
    (hblock : blocks[(uint.Z ref - 1).toNat]? = some block)
    (hnexts : ∀ next ∈ nexts, queueRefValid initial.flags next)
    (hshape : initial.flags.length = initial.inputs.length)
    (hm : validBranchState s.kinds s.unstable kind changing m) (hs : scalarState m)
    (cursor : Nat) (hinvariant : scalarWorkInvariant s kind changing cursor initial)
    (hk : ∀ key, kind key ≠ go!"\"pointer\"")
    (env : w32 → Int) (meaning : ScalarMeaning s kind changing env)
    (hin : stateDenotes kind changing m env)
    (hinverse : ∀ op, (o.inverseTable !! op).getD (zero_val GoString) = inverseOperator op)
    (hreverse : ∀ op, (o.reverseTable !! op).getD (zero_val GoString) = reverseOperator op) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗
       isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
       (querySnapshotOwn view s ∗ s.graph.ValueFlow' ↦□ flow ∗ flow.Blocks' ↦*□ blocks ∗
        block.Successors' ↦*□ nexts ∗ assumeOperatorsOwn o ∗
        workStateOwn receiver view inputs flags queue pointers initial ∗ state ↦$ m) }}
      (App (App (Val #(methods (go.PointerType analysis.ty) go!"successors" #receiver))
        (Val #ref)) (Val #state))
    {{ RET #(); ∃ (out : GoSlice) (locations : List Loc),
        workStateOwn receiver view inputs flags out locations
          (edgePrefix initial nexts kind changing result m nexts.length) ∗ state ↦$ m ∗
        ⌜scalarWorkInvariant s kind changing cursor
            (edgePrefix initial nexts kind changing result m nexts.length) ∧
          inputsExtend kind changing initial (edgePrefix initial nexts kind changing result m nexts.length) ∧
          sourceEdgesCovered s kind changing block.Condition' nexts meaning.eval env
            (edgePrefix initial nexts kind changing result m nexts.length)⌝ }} := by
  obtain ⟨hq, hmodels, hscalars⟩ := hinvariant
  have hb : ∀ truth, scalarOption (result truth m) :=
    fun truth => DecisionEvidence.scalar s o kind changing block.Condition' result decision truth m hm hs hk
  have hscalar := edgePrefix_scalar initial nexts kind changing result m nexts.length hscalars hb
  have hindices : ∀ next ∈ nexts, queueIndex next < initial.inputs.length := by
    intro next hn
    have hvalid := hnexts next hn
    rw [← hshape]
    exact queueIndex_lt initial.flags next hvalid
  have hextends : inputsExtend kind changing initial
      (edgePrefix initial nexts kind changing result m nexts.length) :=
    edgePrefix_extends initial nexts kind changing result m 0 nexts.length (by omega) hindices
  have hcovered : sourceEdgesCovered s kind changing block.Condition' nexts meaning.eval env
      (edgePrefix initial nexts kind changing result m nexts.length) := by
    intro i next hnext htruth
    have hbranch := DecisionEvidence.sound s o kind changing block.Condition' result decision
      (decide (i = 0)) m hm env meaning hin htruth hinverse hreverse
    have hi : i < nexts.length := (List.getElem?_eq_some_iff.mp hnext).choose
    exact edgePrefix_covers initial nexts kind changing result m i nexts.length next env hindices hnext
      hi hbranch
  wp_start_folded as Hinputs
  wp_apply wp_successors_invariant receiver view state s flow blocks block nexts ref m
      inputs flags queue pointers initial o kind changing result decision wf hparse href hblock hnexts hshape
      hmodels hm cursor hq $$ [Hinputs] with ⟨%out, %locations, Hwork, Hstate, %hvalid⟩
  · iexact Hinputs
  iapply HΦ
  iexists out, locations
  iframe Hwork Hstate
  ipureintro
  exact ⟨⟨hvalid.1, hvalid.2, hscalar⟩, hextends, hcovered⟩

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
