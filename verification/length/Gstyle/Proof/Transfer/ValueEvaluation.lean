module
public import Gstyle.Proof.Transfer.ValueDispatch
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
inductive ValueEvaluation [FfiSyntax] (s : QuerySnapshot) (m : StateData) (sf : Nat) : w32 → Value → Prop where
  | zero : ValueEvaluation s m sf (W32 0) (zero_val Value)
  | bound (ref key : w32) (kind : GoString) (changing : Bool)
      (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.nodes.length)
      (hresolve : symbolResolve s sf ref = some key)
      (hk : 0 < uint.Z key ∧ uint.Z key ≤ s.kinds.length)
      (hkind : s.kinds[(uint.Z key - 1).toNat]? = some kind)
      (hunstable : s.unstable[(uint.Z key - 1).toNat]? = some changing) :
      ValueEvaluation s m sf ref (bindingValue kind changing key m)
  | outOfRange (ref key : w32)
      (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.nodes.length)
      (hresolve : symbolResolve s sf ref = some key)
      (hout : s.kinds.length < uint.Z key) :
      ValueEvaluation s m sf ref (zero_val Value)
  | parentheses (ref child : w32) (row : GoArray w32 2) (result : Value)
      (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.nodes.length)
      (hresolve : symbolResolve s sf ref = some (W32 0))
      (hop : decodedFact s (snapshotRecord s ref).Operator' = go!"parentheses")
      (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
      (hchildref : row.arr[0]? = some child)
      (childEvaluation : ValueEvaluation s m sf child result) :
      ValueEvaluation s m sf ref result
  | conversionConstant (ref : w32)
      (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.nodes.length)
      (hresolve : symbolResolve s sf ref = some (W32 0))
      (hop : decodedFact s (snapshotRecord s ref).Operator' = go!"conversion")
      (hinteger : (snapshotConstant s ref).Integer' = true) :
      ValueEvaluation s m sf ref (snapshotConstant s ref)
  | conversionChild (ref child : w32) (row : GoArray w32 2) (source : Value)
      (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.nodes.length)
      (hresolve : symbolResolve s sf ref = some (W32 0))
      (hop : decodedFact s (snapshotRecord s ref).Operator' = go!"conversion")
      (hinteger : (snapshotConstant s ref).Integer' = false)
      (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
      (hchildref : row.arr[0]? = some child)
      (childEvaluation : ValueEvaluation s m sf child source) :
      ValueEvaluation s m sf ref (conversionResult (snapshotFact s ref (go!"type")) source)
  | plain (ref : w32)
      (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.nodes.length)
      (hresolve : symbolResolve s sf ref = some (W32 0))
      (hparent : decodedFact s (snapshotRecord s ref).Operator' ≠ go!"parentheses")
      (hconversion : decodedFact s (snapshotRecord s ref).Operator' ≠ go!"conversion")
      (hlength : (lengthCandidate s ref).Length' = false) :
      ValueEvaluation s m sf ref (lengthCandidate s ref)
  | lengthChild (ref child : w32) (row : GoArray w32 2) (source : Value)
      (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.nodes.length)
      (hresolve : symbolResolve s sf ref = some (W32 0))
      (hparent : decodedFact s (snapshotRecord s ref).Operator' ≠ go!"parentheses")
      (hconversion : decodedFact s (snapshotRecord s ref).Operator' ≠ go!"conversion")
      (hlength : (lengthCandidate s ref).Length' = true)
      (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
      (hchildref : row.arr[0]? = some child)
      (childEvaluation : ValueEvaluation s m sf child source) :
      ValueEvaluation s m sf ref (lengthResult (lengthCandidate s ref) source)

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

theorem wp_value_evaluation (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (sf : Nat) (ref : w32) (result : Value) (wf : SymbolSnapshotValid s)
    (hulen : s.unstable.length = s.kinds.length) (hparse : parseInt64Contract (GF := GF))
    (evaluation : ValueEvaluation s m sf ref result) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"value" #receiver)) (Val #ref)) (Val #state))
    {{ RET #result; querySnapshotOwn receiver s ∗ state ↦$ m }} := by
  induction evaluation with
  | zero =>
    wp_start_folded as Hresources
    wp_apply wp_value_zero receiver state
    wp_end
  | bound ref key kind changing href hresolve hk hkind hunstable =>
    have hn : ref ≠ W32 0 := by obtain ⟨_, _⟩ := href; word
    exact wp_value_bound receiver s state m ref key sf kind changing (Or.inr href) hn wf hresolve hk hkind hulen hunstable wf.allocation hparse
  | outOfRange ref key href hresolve hout =>
    have hn : ref ≠ W32 0 := by obtain ⟨_, _⟩ := href; word
    exact wp_value_bound_out_of_range receiver s state m ref key sf (Or.inr href) hn wf hresolve hout wf.allocation hparse
  | parentheses ref child row result href hresolve hop hrow hchildref childEvaluation ih =>
    have ho : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length := by rw [wf.operands]; exact href
    exact wp_value_parentheses_dispatch receiver s ref child state row sf result iprop(state ↦$ m) ho (Or.inr href) hrow hchildref hop wf.allocation hparse wf hresolve ih
  | conversionConstant ref href hresolve hop hinteger =>
    have hn : ref ≠ W32 0 := by obtain ⟨_, _⟩ := href; word
    exact wp_value_conversion_constant receiver s ref state sf iprop(state ↦$ m) hn (Or.inr href) hop hinteger wf.allocation hparse wf hresolve
  | conversionChild ref child row source href hresolve hop hinteger hrow hchildref childEvaluation ih =>
    have ho : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length := by rw [wf.operands]; exact href
    exact wp_value_conversion_child receiver s ref child state row sf source iprop(state ↦$ m) ho (Or.inr href) hrow hchildref hop hinteger wf.allocation hparse wf hresolve ih
  | plain ref href hresolve hparent hconversion hlength =>
    have hn : ref ≠ W32 0 := by obtain ⟨_, _⟩ := href; word
    exact wp_value_other_no_length receiver s ref state sf iprop(state ↦$ m) hn (Or.inr href) hparent hconversion wf.allocation hparse wf hresolve hlength
  | lengthChild ref child row source href hresolve hparent hconversion hlength hrow hchildref childEvaluation ih =>
    have hn : ref ≠ W32 0 := by obtain ⟨_, _⟩ := href; word
    have ho : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length := by rw [wf.operands]; exact href
    exact wp_value_other_length_child receiver s ref child state sf source row iprop(state ↦$ m) hn (Or.inr href) hparent hconversion wf.allocation hparse wf hresolve hlength ho hrow hchildref ih

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
