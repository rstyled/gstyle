module
public import Gstyle.Proof.Transfer.ValueResolution
public import Gstyle.Proof.Transfer.Numeric
public import Gstyle.Proof.Transfer.ComparisonModel
public import Gstyle.Proof.Transfer.ComparisonQueries
public import Gstyle.Proof.Transfer.ComparisonNil
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

theorem wp_comparison_unbound_true (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (ref left right : w32) (row : GoArray w32 2) (op : GoString) (lv rv : Value)
    (sf lf rf : Nat) (wf : ValueSnapshotValid s)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some left) (hright : row.arr[1]? = some right)
    (hlvalid : validSymbolRef s left) (hrvalid : validSymbolRef s right)
    (hlvalue : valueResolve s m sf lf left = some lv)
    (hrvalue : valueResolve s m sf rf right = some rv)
    (hlsymbol : symbolResolve s sf left = some (W32 0))
    (hrsymbol : symbolResolve s sf right = some (W32 0))
    (hparse : parseInt64Contract (GF := GF)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"comparison" #receiver)) (Val #ref)) (Val #op)) (Val #true)) (Val #state))
    {{ RET #state; querySnapshotOwn receiver s ∗ state ↦$ m }} := by
  exact wp_comparison_queries_true receiver s state m ref left right row op lv rv
    href hrow hleft hright
    (wp_value_resolve receiver s state m sf lf left lv wf hlvalid hparse hlvalue)
    (wp_value_resolve receiver s state m sf rf right rv wf hrvalid hparse hrvalue)
    (wp_symbol_resolve receiver s sf left (W32 0) wf.symbols hlvalid hlsymbol)
    (wp_symbol_resolve receiver s sf right (W32 0) wf.symbols hrvalid hrsymbol)

theorem wp_comparison_unbound_false (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (ref left right : w32) (row : GoArray w32 2) (op : GoString) (lv rv : Value)
    (sf lf rf : Nat) (inverseLoc : Loc) (inverseTable : GMap GoString GoString) (wf : ValueSnapshotValid s)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some left) (hright : row.arr[1]? = some right)
    (hlvalid : validSymbolRef s left) (hrvalid : validSymbolRef s right)
    (hlvalue : valueResolve s m sf lf left = some lv)
    (hrvalue : valueResolve s m sf rf right = some rv)
    (hlsymbol : symbolResolve s sf left = some (W32 0))
    (hrsymbol : symbolResolve s sf right = some (W32 0))
    (hparse : parseInt64Contract (GF := GF)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (globalAddr inverse ↦□ inverseLoc ∗ inverseLoc ↦${DFrac.discard} inverseTable ∗ querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"comparison" #receiver)) (Val #ref)) (Val #op)) (Val #false)) (Val #state))
    {{ RET #state; globalAddr inverse ↦□ inverseLoc ∗ inverseLoc ↦${DFrac.discard} inverseTable ∗ querySnapshotOwn receiver s ∗ state ↦$ m }} := by
  exact wp_comparison_queries_false receiver s state m ref left right row op lv rv inverseLoc inverseTable
    href hrow hleft hright
    (wp_value_resolve receiver s state m sf lf left lv wf hlvalid hparse hlvalue)
    (wp_value_resolve receiver s state m sf rf right rv wf hrvalid hparse hrvalue)
    (wp_symbol_resolve receiver s sf left (W32 0) wf.symbols hlvalid hlsymbol)
    (wp_symbol_resolve receiver s sf right (W32 0) wf.symbols hrvalid hrsymbol)


end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
