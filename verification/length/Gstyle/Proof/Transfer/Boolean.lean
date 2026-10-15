module
public import Gstyle.Proof.Transfer.Facts
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

def booleanConflict (value : GoString) (truth : Bool) : Bool :=
  decide ((value = go!"true" ∧ truth = false) ∨ (value = go!"false" ∧ truth = true))

theorem booleanConflict_unknown (value : GoString) (truth : Bool)
    (ht : value ≠ go!"true") (hf : value ≠ go!"false") :
    booleanConflict value truth = false := by
  simp [booleanConflict, ht, hf]

theorem booleanConflict_literals (truth : Bool) :
    booleanConflict (go!"true") truth = !truth ∧ booleanConflict (go!"false") truth = truth := by
  cases truth <;> decide

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics]
variable [FuncUnfold booleanContradiction [] booleanContradiction.impl]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

theorem wp_booleanContradiction (value : GoString) (truth : Bool) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! booleanContradiction)) (Val #value)) (Val #truth))
    {{ RET #(booleanConflict value truth); True }} := by
  wp_start
  by_cases ht : value = go!"true" <;> by_cases hf : value = go!"false" <;> cases truth
  all_goals iterate 5 (try simp [booleanConflict, ht, hf]; try wp_auto)
  all_goals wp_end

variable [graphView.TypeAssumptions] [State.TypeAssumptions] [pdg.query.Assumptions]
variable [pdg.Graph.TypeAssumptions] [pdg.Node.TypeAssumptions] [pdg.Attributes.TypeAssumptions]
variable [pdg.AttributeSet.TypeAssumptions] [pdg.AttributeTables.TypeAssumptions]
variable [pdg.Producer.TypeAssumptions] [pdg.Function.TypeAssumptions] [pdg.Span.TypeAssumptions]
variable [pdg.CountFact.TypeAssumptions] [pdg.Conformance.TypeAssumptions] [pdg.Fact.TypeAssumptions]
variable [pdg.Position.TypeAssumptions] [pdg.CallForm.TypeAssumptions] [pdg.ExitKind.TypeAssumptions]
variable [pdg.ConformanceStatus.TypeAssumptions] [pdg.Category.TypeAssumptions]
variable [pdg.Tables.TypeAssumptions] [pdg.Extension.TypeAssumptions]
variable [FuncUnfold pdg.listAt [pdg.Ref.ty] (pdg.listAt.impl pdg.Ref.ty)]
variable [FuncUnfold pdg.listAt [pdg.Access.ty] (pdg.listAt.impl pdg.Access.ty)]
variable [FuncUnfold pdg.listAt [pdg.Extension.ty] (pdg.listAt.impl pdg.Extension.ty)]
variable [FuncUnfold extension [] extension.impl]

theorem wp_assume_boolean_reject (receiver : Loc) (view : graphView) (graph : pdg.Graph)
    (nodes : List pdg.Node) (node : pdg.Node) (records : List pdg.AttributeSet) (a : pdg.AttributeSet)
    (references accesses extensions : List GoSlice) (values : List w32)
    (table : pdg.Tables) (strings : List GoString) (entries : List pdg.Extension)
    (ref : w32) (truth : Bool) (state : Loc) (vd gd td ed : DFrac)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ nodes.length)
    (hnode : nodes[(uint.Z ref - 1).toNat]? = some node)
    (ha : 0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ records.length)
    (hrecord : records[(uint.Z node.Attributes' - 1).toNat]? = some a) (halloc : values.length ≤ 2^46)
    (hreject : booleanConflict (extensionValue strings (go!"boolean") entries) truth = true) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦{vd} view ∗ view.graph' ↦{gd} graph ∗
       graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records ∗
       graph.AttributeTables'.References' ↦*{gd} references ∗ graph.AttributeTables'.Accesses' ↦*{gd} accesses ∗
       graph.AttributeTables'.Extensions' ↦*{gd} extensions ∗ pdg.listRow references a.Outcomes' ↦*{gd} values ∗
       graph.Tables' ↦{td} table ∗ table.Strings' ↦*{td} strings ∗ pdg.listRow extensions a.Extensions' ↦*{ed} entries) }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"assume" #receiver)) (Val #ref)) (Val #truth)) (Val #state))
    {{ RET #map.nil;
       receiver ↦{vd} view ∗ view.graph' ↦{gd} graph ∗ graph.Nodes' ↦*{gd} nodes ∗ graph.Attributes' ↦*{gd} records ∗
       graph.AttributeTables'.References' ↦*{gd} references ∗ graph.AttributeTables'.Accesses' ↦*{gd} accesses ∗
       graph.AttributeTables'.Extensions' ↦*{gd} extensions ∗ pdg.listRow references a.Outcomes' ↦*{gd} values ∗
       graph.Tables' ↦{td} table ∗ table.Strings' ↦*{td} strings ∗ pdg.listRow extensions a.Extensions' ↦*{ed} entries }} := by
  wp_start as ⟨Hview, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Htable, Hstrings, Hentries⟩
  try wp_auto
  wp_apply wp_fact_present receiver view graph nodes node records a references accesses extensions values table strings entries ref (go!"boolean") vd gd td ed href hnode ha hrecord halloc $$ [Hview Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues Htable Hstrings Hentries] with ⟨Hview, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Htable, Hstrings, Hentries⟩
  · iframe
  try wp_auto
  wp_apply wp_booleanContradiction (extensionValue strings (go!"boolean") entries) truth
  rw [hreject]
  try wp_auto
  wp_end

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
