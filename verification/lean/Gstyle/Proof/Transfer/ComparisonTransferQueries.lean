module
public import Gstyle.Proof.Transfer.ComparisonQueries
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


def comparisonConstraintContract (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (key : w32) (kind op : GoString) (changing active : Bool) (bound : Value) : Prop :=
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
       (querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"constrain" #receiver)) (Val #key)) (Val #op)) (Val #bound)) (Val #state))
    {{ RET #(if active && constraintRejected kind changing key m op bound then map.nil else state);
       querySnapshotOwn receiver s ∗ state ↦$ (constraintState active kind changing key m op bound) }}

theorem wp_comparison_left_numeric_true_queries (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (ref left right key : w32) (row : GoArray w32 2) (kind op : GoString) (lv rv : Value) (changing active : Bool)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some left) (hright : row.arr[1]? = some right)
    (hkey : key ≠ W32 0)
    (hlvalue : comparisonValueContract (GF := GF) receiver s state m left lv)
    (hrvalue : comparisonValueContract (GF := GF) receiver s state m right rv)
    (hlsymbol : comparisonSymbolContract (GF := GF) receiver s left key)
    (hrsymbol : comparisonSymbolContract (GF := GF) receiver s right (W32 0))
    (hconstraint : comparisonConstraintContract (GF := GF) receiver s state m key kind op changing active rv) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗ (querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"comparison" #receiver)) (Val #ref)) (Val #op)) (Val #true)) (Val #state))
    {{ RET #(if active && constraintRejected kind changing key m op rv then map.nil else state);
      querySnapshotOwn receiver s ∗ state ↦$ (constraintState active kind changing key m op rv) }} := by
  unfold comparisonValueContract at hlvalue hrvalue
  unfold comparisonSymbolContract at hlsymbol hrsymbol
  unfold comparisonConstraintContract at hconstraint
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
  wp_apply hlvalue
    $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
  · iframe Hsnapshot Hstate
  try wp_auto
  wp_apply hrvalue
    $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
  · iframe Hsnapshot Hstate
  try wp_auto
  wp_apply hlsymbol
    $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  try wp_auto
  wp_if_destruct
  · exfalso; contradiction
  · wp_apply hconstraint
      $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
    · iframe Hsnapshot Hstate
    cases rejected : active && constraintRejected kind changing key m op rv with
    | false =>
      simp only [rejected, Bool.false_eq_true, ite_false]
      try wp_auto
      wp_if_destruct
      · exfalso; contradiction
      · wp_apply hrsymbol
          $$ [Hsnapshot] with Hsnapshot
        · iframe Hsnapshot
        try wp_auto
        wp_end
    | true =>
      simp only [rejected, ite_true]
      try wp_auto
      wp_end

theorem wp_comparison_left_numeric_false_queries (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (ref left right key : w32) (row : GoArray w32 2) (kind op : GoString) (lv rv : Value) (changing active : Bool)
    (inverseLoc : Loc) (inverseTable : GMap GoString GoString) (normal : GoString) 
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some left) (hright : row.arr[1]? = some right)
    (hkey : key ≠ W32 0)
    (hlookup : (inverseTable !! op).getD (zero_val GoString) = normal)
    (hlvalue : comparisonValueContract (GF := GF) receiver s state m left lv)
    (hrvalue : comparisonValueContract (GF := GF) receiver s state m right rv)
    (hlsymbol : comparisonSymbolContract (GF := GF) receiver s left key)
    (hrsymbol : comparisonSymbolContract (GF := GF) receiver s right (W32 0))
    (hconstraint : comparisonConstraintContract (GF := GF) receiver s state m key kind normal changing active rv) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗ (globalAddr inverse ↦□ inverseLoc ∗ inverseLoc ↦${DFrac.discard} inverseTable ∗ querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"comparison" #receiver)) (Val #ref)) (Val #op)) (Val #false)) (Val #state))
    {{ RET #(if active && constraintRejected kind changing key m normal rv then map.nil else state);
      globalAddr inverse ↦□ inverseLoc ∗ inverseLoc ↦${DFrac.discard} inverseTable ∗ querySnapshotOwn receiver s ∗ state ↦$ (constraintState active kind changing key m normal rv) }} := by
  unfold comparisonValueContract at hlvalue hrvalue
  unfold comparisonSymbolContract at hlsymbol hrsymbol
  unfold comparisonConstraintContract at hconstraint
  wp_start as ⟨#Hglobal, #Hoperators, #Hsnapshot, Hstate⟩
  ihave %Hnonnull := ownMap_not_nil state m (DFrac.own 1) $$ Hstate
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, _, _, Hrows, _, _, _, _, _, _, _, _, _, _, _, _, _⟩
  try wp_auto
  wp_apply wp_map_lookup1 go.string go.string inverseLoc inverseTable op DFrac.discard $$ [Hoperators] with Hoperators
  · iframe Hoperators
  rw [hlookup]
  try wp_auto
  wp_apply wp_operand_readonly receiver s.view s.operands ref (W64 0) row left href (by decide) hrow hleft DFrac.discard DFrac.discard
    $$ [Hview Hrows] with ⟨Hview, Hrows⟩
  · iframe Hview Hrows
  try wp_auto
  wp_apply wp_operand_readonly receiver s.view s.operands ref (W64 1) row right href (by decide) hrow hright DFrac.discard DFrac.discard
    $$ [Hview Hrows] with ⟨Hview, Hrows⟩
  · iframe Hview Hrows
  try wp_auto
  wp_apply hlvalue
    $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
  · iframe Hsnapshot Hstate
  try wp_auto
  wp_apply hrvalue
    $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
  · iframe Hsnapshot Hstate
  try wp_auto
  wp_apply hlsymbol
    $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  try wp_auto
  wp_if_destruct
  · exfalso; contradiction
  · wp_apply hconstraint
      $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
    · iframe Hsnapshot Hstate
    cases rejected : active && constraintRejected kind changing key m normal rv with
    | false =>
      simp only [rejected, Bool.false_eq_true, ite_false]
      try wp_auto
      wp_if_destruct
      · exfalso; contradiction
      · wp_apply hrsymbol
          $$ [Hsnapshot] with Hsnapshot
        · iframe Hsnapshot
        try wp_auto
        wp_end
        all_goals iframe Hglobal Hoperators Hsnapshot Hstate
    | true =>
      simp only [rejected, ite_true]
      try wp_auto
      wp_end
      all_goals iframe Hglobal Hoperators Hsnapshot Hstate

theorem wp_comparison_right_numeric_true_queries (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (ref left right key : w32) (row : GoArray w32 2) (kind op : GoString) (lv rv : Value) (changing active : Bool)
    (reverseLoc : Loc) (reverseTable : GMap GoString GoString) (normal : GoString) 
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some left) (hright : row.arr[1]? = some right)
    (hkey : key ≠ W32 0)
    (hlookup : (reverseTable !! op).getD (zero_val GoString) = normal)
    (hlvalue : comparisonValueContract (GF := GF) receiver s state m left lv)
    (hrvalue : comparisonValueContract (GF := GF) receiver s state m right rv)
    (hlsymbol : comparisonSymbolContract (GF := GF) receiver s left (W32 0))
    (hrsymbol : comparisonSymbolContract (GF := GF) receiver s right key)
    (hconstraint : comparisonConstraintContract (GF := GF) receiver s state m key kind normal changing active lv) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗ (globalAddr reverse ↦□ reverseLoc ∗ reverseLoc ↦${DFrac.discard} reverseTable ∗ querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"comparison" #receiver)) (Val #ref)) (Val #op)) (Val #true)) (Val #state))
    {{ RET #(if active && constraintRejected kind changing key m normal lv then map.nil else state);
      globalAddr reverse ↦□ reverseLoc ∗ reverseLoc ↦${DFrac.discard} reverseTable ∗ querySnapshotOwn receiver s ∗ state ↦$ (constraintState active kind changing key m normal lv) }} := by
  unfold comparisonValueContract at hlvalue hrvalue
  unfold comparisonSymbolContract at hlsymbol hrsymbol
  unfold comparisonConstraintContract at hconstraint
  wp_start as ⟨#Hglobal, #Hoperators, #Hsnapshot, Hstate⟩
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
  wp_apply hlvalue
    $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
  · iframe Hsnapshot Hstate
  try wp_auto
  wp_apply hrvalue
    $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
  · iframe Hsnapshot Hstate
  try wp_auto
  wp_apply hlsymbol
    $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  try wp_auto
  wp_if_destruct
  · exfalso; contradiction
  · wp_apply hrsymbol
      $$ [Hsnapshot] with Hsnapshot
    · iframe Hsnapshot
    try wp_auto
    wp_if_destruct
    · exfalso; contradiction
    · try wp_auto
      wp_apply wp_map_lookup1 go.string go.string reverseLoc reverseTable op DFrac.discard $$ [Hoperators] with Hoperators
      · iframe Hoperators
      rw [hlookup]
      try wp_auto
      wp_apply hconstraint
        $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
      · iframe Hsnapshot Hstate
      try wp_auto
      wp_end
      all_goals iframe Hglobal Hoperators Hsnapshot Hstate

theorem wp_comparison_right_numeric_false_queries (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (ref left right key : w32) (row : GoArray w32 2) (kind op : GoString) (lv rv : Value) (changing active : Bool)
    (reverseLoc : Loc) (reverseTable : GMap GoString GoString) (normal negated : GoString) (inverseLoc : Loc) (inverseTable : GMap GoString GoString) 
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some left) (hright : row.arr[1]? = some right)
    (hkey : key ≠ W32 0)
    (hlookup : (reverseTable !! negated).getD (zero_val GoString) = normal)
    (hnegate : (inverseTable !! op).getD (zero_val GoString) = negated)
    (hlvalue : comparisonValueContract (GF := GF) receiver s state m left lv)
    (hrvalue : comparisonValueContract (GF := GF) receiver s state m right rv)
    (hlsymbol : comparisonSymbolContract (GF := GF) receiver s left (W32 0))
    (hrsymbol : comparisonSymbolContract (GF := GF) receiver s right key)
    (hconstraint : comparisonConstraintContract (GF := GF) receiver s state m key kind normal changing active lv) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗ (globalAddr inverse ↦□ inverseLoc ∗ inverseLoc ↦${DFrac.discard} inverseTable ∗ globalAddr reverse ↦□ reverseLoc ∗ reverseLoc ↦${DFrac.discard} reverseTable ∗ querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"comparison" #receiver)) (Val #ref)) (Val #op)) (Val #false)) (Val #state))
    {{ RET #(if active && constraintRejected kind changing key m normal lv then map.nil else state);
      globalAddr inverse ↦□ inverseLoc ∗ inverseLoc ↦${DFrac.discard} inverseTable ∗ globalAddr reverse ↦□ reverseLoc ∗ reverseLoc ↦${DFrac.discard} reverseTable ∗ querySnapshotOwn receiver s ∗ state ↦$ (constraintState active kind changing key m normal lv) }} := by
  unfold comparisonValueContract at hlvalue hrvalue
  unfold comparisonSymbolContract at hlsymbol hrsymbol
  unfold comparisonConstraintContract at hconstraint
  wp_start as ⟨#HinvGlobal, #HinvOperators, #Hglobal, #Hoperators, #Hsnapshot, Hstate⟩
  ihave %Hnonnull := ownMap_not_nil state m (DFrac.own 1) $$ Hstate
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, _, _, Hrows, _, _, _, _, _, _, _, _, _, _, _, _, _⟩
  try wp_auto
  wp_apply wp_map_lookup1 go.string go.string inverseLoc inverseTable op DFrac.discard $$ [HinvOperators] with HinvOperators
  · iframe HinvOperators
  rw [hnegate]
  try wp_auto
  wp_apply wp_operand_readonly receiver s.view s.operands ref (W64 0) row left href (by decide) hrow hleft DFrac.discard DFrac.discard
    $$ [Hview Hrows] with ⟨Hview, Hrows⟩
  · iframe Hview Hrows
  try wp_auto
  wp_apply wp_operand_readonly receiver s.view s.operands ref (W64 1) row right href (by decide) hrow hright DFrac.discard DFrac.discard
    $$ [Hview Hrows] with ⟨Hview, Hrows⟩
  · iframe Hview Hrows
  try wp_auto
  wp_apply hlvalue
    $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
  · iframe Hsnapshot Hstate
  try wp_auto
  wp_apply hrvalue
    $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
  · iframe Hsnapshot Hstate
  try wp_auto
  wp_apply hlsymbol
    $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  try wp_auto
  wp_if_destruct
  · exfalso; contradiction
  · wp_apply hrsymbol
      $$ [Hsnapshot] with Hsnapshot
    · iframe Hsnapshot
    try wp_auto
    wp_if_destruct
    · exfalso; contradiction
    · try wp_auto
      wp_apply wp_map_lookup1 go.string go.string reverseLoc reverseTable negated DFrac.discard $$ [Hoperators] with Hoperators
      · iframe Hoperators
      rw [hlookup]
      try wp_auto
      wp_apply hconstraint
        $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
      · iframe Hsnapshot Hstate
      try wp_auto
      wp_end
      all_goals iframe HinvGlobal HinvOperators Hglobal Hoperators Hsnapshot Hstate

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
