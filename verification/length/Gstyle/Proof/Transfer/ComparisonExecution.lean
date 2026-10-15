module
public import Gstyle.Proof.Transfer.ComparisonModel
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
  wp_start as ⟨#Hsnapshot, Hstate⟩
  ihave %Hnonnull := ownMap_not_nil state m (DFrac.own 1) $$ Hstate
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, _, _, Hrows, _, _, _, _, _, _, _, _, _, _, _, _, _⟩
  try wp_auto
  wp_apply wp_operand_readonly receiver s.view s.operands ref (W64 0) row left href (by decide) hrow hleft DFrac.discard DFrac.discard
    $$ [Hview Hrows] with ⟨Hview, Hrows⟩
  · iframe Hview Hrows
  try wp_auto
  wp_apply wp_operand_readonly receiver s.view s.operands ref (W64 1) row right href (by decide) hrow hright DFrac.discard DFrac.discard
    $$ [Hview Hrows] with ⟨Hview, Hrows⟩
  · iframe Hview Hrows
  try wp_auto
  wp_apply wp_value_resolve receiver s state m sf lf left lv wf hlvalid hparse hlvalue
    $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
  · iframe Hsnapshot Hstate
  try wp_auto
  wp_apply wp_value_resolve receiver s state m sf rf right rv wf hrvalid hparse hrvalue
    $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
  · iframe Hsnapshot Hstate
  try wp_auto
  wp_apply wp_symbol_resolve receiver s sf left (W32 0) wf.symbols hlvalid hlsymbol
    $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  try wp_auto
  wp_if_destruct
  · exfalso; contradiction
  · wp_apply wp_symbol_resolve receiver s sf right (W32 0) wf.symbols hrvalid hrsymbol
      $$ [Hsnapshot] with Hsnapshot
    · iframe Hsnapshot
    try wp_auto
    wp_end

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
  wp_start as ⟨#Hglobal, #Hoperators, #Hsnapshot, Hstate⟩
  ihave %Hnonnull := ownMap_not_nil state m (DFrac.own 1) $$ Hstate
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, _, _, Hrows, _, _, _, _, _, _, _, _, _, _, _, _, _⟩
  try wp_auto
  wp_apply wp_map_lookup1 go.string go.string inverseLoc inverseTable op DFrac.discard $$ [Hoperators] with Hoperators
  · iframe Hoperators
  try wp_auto
  wp_apply wp_operand_readonly receiver s.view s.operands ref (W64 0) row left href (by decide) hrow hleft DFrac.discard DFrac.discard
    $$ [Hview Hrows] with ⟨Hview, Hrows⟩
  · iframe Hview Hrows
  try wp_auto
  wp_apply wp_operand_readonly receiver s.view s.operands ref (W64 1) row right href (by decide) hrow hright DFrac.discard DFrac.discard
    $$ [Hview Hrows] with ⟨Hview, Hrows⟩
  · iframe Hview Hrows
  try wp_auto
  wp_apply wp_value_resolve receiver s state m sf lf left lv wf hlvalid hparse hlvalue
    $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
  · iframe Hsnapshot Hstate
  try wp_auto
  wp_apply wp_value_resolve receiver s state m sf rf right rv wf hrvalid hparse hrvalue
    $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
  · iframe Hsnapshot Hstate
  try wp_auto
  wp_apply wp_symbol_resolve receiver s sf left (W32 0) wf.symbols hlvalid hlsymbol
    $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  try wp_auto
  wp_if_destruct
  · exfalso; contradiction
  · wp_apply wp_symbol_resolve receiver s sf right (W32 0) wf.symbols hrvalid hrsymbol
      $$ [Hsnapshot] with Hsnapshot
    · iframe Hsnapshot
    try wp_auto
    wp_end
    all_goals iframe Hglobal Hoperators Hsnapshot Hstate

theorem wp_comparison_nil_zero_operands_true (receiver : Loc) (s : QuerySnapshot)
    (ref : w32) (row : GoArray w32 2) (op : GoString)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some (W32 0)) (hright : row.arr[1]? = some (W32 0)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ querySnapshotOwn receiver s }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"comparison" #receiver)) (Val #ref)) (Val #op)) (Val #true)) (Val #map.nil))
    {{ RET #map.nil; querySnapshotOwn receiver s }} := by
  wp_start as #Hsnapshot
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, _, _, Hrows, _, _, _, _, _, _, _, _, _, _, _, _, _⟩
  try wp_auto
  wp_apply wp_operand_readonly receiver s.view s.operands ref (W64 0) row (W32 0) href (by decide) hrow hleft DFrac.discard DFrac.discard
    $$ [Hview Hrows] with ⟨Hview, Hrows⟩
  · iframe Hview Hrows
  try wp_auto
  wp_apply wp_operand_readonly receiver s.view s.operands ref (W64 1) row (W32 0) href (by decide) hrow hright DFrac.discard DFrac.discard
    $$ [Hview Hrows] with ⟨Hview, Hrows⟩
  · iframe Hview Hrows
  try wp_auto
  wp_apply wp_value_zero receiver map.nil
  try wp_auto
  wp_apply wp_value_zero receiver map.nil
  try wp_auto
  wp_apply wp_symbol_zero receiver
  try wp_auto
  wp_end
  all_goals iframe Hsnapshot

theorem wp_comparison_nil_zero_operands_false (receiver : Loc) (s : QuerySnapshot)
    (ref : w32) (row : GoArray w32 2) (op : GoString) (inverseLoc : Loc) (inverseTable : GMap GoString GoString)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some (W32 0)) (hright : row.arr[1]? = some (W32 0)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (globalAddr inverse ↦□ inverseLoc ∗ inverseLoc ↦${DFrac.discard} inverseTable ∗ querySnapshotOwn receiver s) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"comparison" #receiver)) (Val #ref)) (Val #op)) (Val #false)) (Val #map.nil))
    {{ RET #map.nil; globalAddr inverse ↦□ inverseLoc ∗ inverseLoc ↦${DFrac.discard} inverseTable ∗ querySnapshotOwn receiver s }} := by
  wp_start as ⟨#Hglobal, #Hoperators, #Hsnapshot⟩
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, _, _, Hrows, _, _, _, _, _, _, _, _, _, _, _, _, _⟩
  try wp_auto
  wp_apply wp_map_lookup1 go.string go.string inverseLoc inverseTable op DFrac.discard $$ [Hoperators] with Hoperators
  · iframe Hoperators
  try wp_auto
  wp_apply wp_operand_readonly receiver s.view s.operands ref (W64 0) row (W32 0) href (by decide) hrow hleft DFrac.discard DFrac.discard
    $$ [Hview Hrows] with ⟨Hview, Hrows⟩
  · iframe Hview Hrows
  try wp_auto
  wp_apply wp_operand_readonly receiver s.view s.operands ref (W64 1) row (W32 0) href (by decide) hrow hright DFrac.discard DFrac.discard
    $$ [Hview Hrows] with ⟨Hview, Hrows⟩
  · iframe Hview Hrows
  try wp_auto
  wp_apply wp_value_zero receiver map.nil
  try wp_auto
  wp_apply wp_value_zero receiver map.nil
  try wp_auto
  wp_apply wp_symbol_zero receiver
  try wp_auto
  wp_end
  all_goals iframe Hglobal Hoperators Hsnapshot

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
