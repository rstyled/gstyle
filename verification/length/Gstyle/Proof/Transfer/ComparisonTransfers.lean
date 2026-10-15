module
public import Gstyle.Proof.Transfer.ComparisonTransferQueries
public import Gstyle.Proof.Transfer.ValueResolution
public import Gstyle.Proof.Transfer.ComparisonConstraint
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

variable [FuncUnfold integerConstraint [] integerConstraint.impl]
variable [FuncUnfold lengthConstraint [] lengthConstraint.impl]
variable [FuncUnfold exactInteger [] exactInteger.impl]
variable [FuncUnfold emptyRange [] emptyRange.impl]
variable [interval.Assumptions]

theorem wp_comparison_left_numeric_true (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
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
    (hparse : parseInt64Contract (GF := GF)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗ (querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"comparison" #receiver)) (Val #ref)) (Val #op)) (Val #true)) (Val #state))
    {{ RET #(if active && constraintRejected kind changing key m op rv then map.nil else state);
      querySnapshotOwn receiver s ∗ state ↦$ (constraintState active kind changing key m op rv) }} := by
  exact wp_comparison_left_numeric_true_queries receiver s state m ref left right key row kind op lv rv changing active
    href hrow hleft hright hkey
    (wp_value_resolve receiver s state m sf lf left lv wf hlvalid hparse hlvalue)
    (wp_value_resolve receiver s state m sf rf right rv wf hrvalid hparse hrvalue)
    (wp_symbol_resolve receiver s sf left key wf.symbols hlvalid hlsymbol)
    (wp_symbol_resolve receiver s sf right (W32 0) wf.symbols hrvalid hrsymbol)
    (wp_constrain_snapshot_numeric_any receiver s state m key kind op changing active rv wf.unstable ht metadata hnil)

theorem wp_comparison_left_numeric_false (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
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
    (hparse : parseInt64Contract (GF := GF)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗ (globalAddr inverse ↦□ inverseLoc ∗ inverseLoc ↦${DFrac.discard} inverseTable ∗ querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"comparison" #receiver)) (Val #ref)) (Val #op)) (Val #false)) (Val #state))
    {{ RET #(if active && constraintRejected kind changing key m normal rv then map.nil else state);
      globalAddr inverse ↦□ inverseLoc ∗ inverseLoc ↦${DFrac.discard} inverseTable ∗ querySnapshotOwn receiver s ∗ state ↦$ (constraintState active kind changing key m normal rv) }} := by
  exact wp_comparison_left_numeric_false_queries receiver s state m ref left right key row kind op lv rv changing active inverseLoc inverseTable normal
    href hrow hleft hright hkey hlookup
    (wp_value_resolve receiver s state m sf lf left lv wf hlvalid hparse hlvalue)
    (wp_value_resolve receiver s state m sf rf right rv wf hrvalid hparse hrvalue)
    (wp_symbol_resolve receiver s sf left key wf.symbols hlvalid hlsymbol)
    (wp_symbol_resolve receiver s sf right (W32 0) wf.symbols hrvalid hrsymbol)
    (wp_constrain_snapshot_numeric_any receiver s state m key kind normal changing active rv wf.unstable ht metadata hnil)

theorem wp_comparison_right_numeric_true (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
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
    (hparse : parseInt64Contract (GF := GF)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗ (globalAddr reverse ↦□ reverseLoc ∗ reverseLoc ↦${DFrac.discard} reverseTable ∗ querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"comparison" #receiver)) (Val #ref)) (Val #op)) (Val #true)) (Val #state))
    {{ RET #(if active && constraintRejected kind changing key m normal lv then map.nil else state);
      globalAddr reverse ↦□ reverseLoc ∗ reverseLoc ↦${DFrac.discard} reverseTable ∗ querySnapshotOwn receiver s ∗ state ↦$ (constraintState active kind changing key m normal lv) }} := by
  exact wp_comparison_right_numeric_true_queries receiver s state m ref left right key row kind op lv rv changing active reverseLoc reverseTable normal
    href hrow hleft hright hkey hlookup
    (wp_value_resolve receiver s state m sf lf left lv wf hlvalid hparse hlvalue)
    (wp_value_resolve receiver s state m sf rf right rv wf hrvalid hparse hrvalue)
    (wp_symbol_resolve receiver s sf left (W32 0) wf.symbols hlvalid hlsymbol)
    (wp_symbol_resolve receiver s sf right key wf.symbols hrvalid hrsymbol)
    (wp_constrain_snapshot_numeric_any receiver s state m key kind normal changing active lv wf.unstable ht metadata hnil)

theorem wp_comparison_right_numeric_false (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
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
    (hparse : parseInt64Contract (GF := GF)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗ (globalAddr inverse ↦□ inverseLoc ∗ inverseLoc ↦${DFrac.discard} inverseTable ∗ globalAddr reverse ↦□ reverseLoc ∗ reverseLoc ↦${DFrac.discard} reverseTable ∗ querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"comparison" #receiver)) (Val #ref)) (Val #op)) (Val #false)) (Val #state))
    {{ RET #(if active && constraintRejected kind changing key m normal lv then map.nil else state);
      globalAddr inverse ↦□ inverseLoc ∗ inverseLoc ↦${DFrac.discard} inverseTable ∗ globalAddr reverse ↦□ reverseLoc ∗ reverseLoc ↦${DFrac.discard} reverseTable ∗ querySnapshotOwn receiver s ∗ state ↦$ (constraintState active kind changing key m normal lv) }} := by
  exact wp_comparison_right_numeric_false_queries receiver s state m ref left right key row kind op lv rv changing active reverseLoc reverseTable normal negated inverseLoc inverseTable
    href hrow hleft hright hkey hlookup hnegate
    (wp_value_resolve receiver s state m sf lf left lv wf hlvalid hparse hlvalue)
    (wp_value_resolve receiver s state m sf rf right rv wf hrvalid hparse hrvalue)
    (wp_symbol_resolve receiver s sf left (W32 0) wf.symbols hlvalid hlsymbol)
    (wp_symbol_resolve receiver s sf right key wf.symbols hrvalid hrsymbol)
    (wp_constrain_snapshot_numeric_any receiver s state m key kind normal changing active lv wf.unstable ht metadata hnil)


end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
