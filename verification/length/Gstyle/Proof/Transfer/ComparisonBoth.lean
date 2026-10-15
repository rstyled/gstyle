module
public import Gstyle.Proof.Transfer.ComparisonTransfers
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

theorem wp_comparison_both_numeric_true (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (ref left right : w32) (row : GoArray w32 2) (c : NumericComparison)
    (sf lf rf : Nat) (reverseLoc : Loc) (reverseTable : GMap GoString GoString) (wf : ValueSnapshotValid s)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some left) (hright : row.arr[1]? = some right)
    (hlvalid : validSymbolRef s left) (hrvalid : validSymbolRef s right)
    (hlvalue : valueResolve s m sf lf left = some c.leftValue)
    (hrvalue : valueResolve s m sf rf right = some c.rightValue)
    (hlsymbol : symbolResolve s sf left = some c.leftKey)
    (hrsymbol : symbolResolve s sf right = some c.rightKey)
    (hleftNonzero : c.leftKey ≠ W32 0) (hrightNonzero : c.rightKey ≠ W32 0)
    (hleftMetadata : ConstraintMetadata s c.leftKey c.leftKind c.leftChanging c.leftActive)
    (hrightMetadata : ConstraintMetadata s c.rightKey c.rightKind c.rightChanging c.rightActive)
    (ht : s.tracked.length = s.kinds.length)
    (hnill : c.leftValue.Nil' = false) (hnilr : c.rightValue.Nil' = false)
    (hreverse : (reverseTable !! c.operator).getD (zero_val GoString) = c.reversed)
    (hparse : parseInt64Contract (GF := GF)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
      (globalAddr reverse ↦□ reverseLoc ∗ reverseLoc ↦${DFrac.discard} reverseTable ∗ querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"comparison" #receiver)) (Val #ref)) (Val #c.operator)) (Val #true)) (Val #state))
    {{ RET #(if comparisonRejected c m then map.nil else state);
      globalAddr reverse ↦□ reverseLoc ∗ reverseLoc ↦${DFrac.discard} reverseTable ∗
        querySnapshotOwn receiver s ∗ state ↦$ (comparisonResultMap c m) }} := by
  dsimp only [comparisonRejected, comparisonResultMap, comparisonRightRejected, comparisonRightMap, comparisonLeftRejected, comparisonLeftMap]
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
  wp_apply wp_value_resolve receiver s state m sf lf left c.leftValue wf hlvalid hparse hlvalue
    $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
  · try iframe Hsnapshot Hstate
  try wp_auto
  wp_apply wp_value_resolve receiver s state m sf rf right c.rightValue wf hrvalid hparse hrvalue
    $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
  · try iframe Hsnapshot Hstate
  try wp_auto
  wp_apply wp_symbol_resolve receiver s sf left c.leftKey wf.symbols hlvalid hlsymbol
    $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  try wp_auto
  wp_if_destruct
  · exfalso; contradiction
  · wp_apply wp_constrain_snapshot_numeric_any receiver s state m c.leftKey c.leftKind c.operator c.leftChanging c.leftActive c.rightValue
      wf.unstable ht hleftMetadata hnilr $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
    · try iframe Hsnapshot Hstate
    by_cases hlreject : (c.leftActive && constraintRejected c.leftKind c.leftChanging c.leftKey m c.operator c.rightValue) = true
    ·
      simp only [hlreject, Bool.true_or, ite_true]
      try wp_auto
      wp_end
      all_goals try iframe Hglobal Hoperators Hsnapshot Hstate
    · have hfalse : (c.leftActive && constraintRejected c.leftKind c.leftChanging c.leftKey m c.operator c.rightValue) = false := Bool.eq_false_iff.mpr hlreject
      simp only [hfalse, Bool.false_eq_true, Bool.false_or, ite_false, ite_true]
      try wp_auto
      wp_if_destruct
      · exfalso; contradiction
      · wp_apply wp_symbol_resolve receiver s sf right c.rightKey wf.symbols hrvalid hrsymbol
          $$ [Hsnapshot] with Hsnapshot
        · iframe Hsnapshot
        try wp_auto
        wp_if_destruct
        · exfalso; contradiction
        · try wp_auto
          wp_apply wp_map_lookup1 go.string go.string reverseLoc reverseTable c.operator DFrac.discard $$ [Hoperators] with Hoperators
          · iframe Hoperators
          rw [hreverse]
          try wp_auto
          wp_apply wp_constrain_snapshot_numeric_any receiver s state
            (constraintState c.leftActive c.leftKind c.leftChanging c.leftKey m c.operator c.rightValue)
            c.rightKey c.rightKind c.reversed c.rightChanging c.rightActive c.leftValue
            wf.unstable ht hrightMetadata hnill $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
          · try iframe Hsnapshot Hstate
          try wp_auto
          wp_end
          all_goals try iframe Hglobal Hoperators Hsnapshot Hstate

theorem wp_comparison_both_numeric_false (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (ref left right : w32) (row : GoArray w32 2) (c : NumericComparison)
    (sf lf rf : Nat) (rawOperator : GoString) (inverseLoc : Loc) (inverseTable : GMap GoString GoString) (reverseLoc : Loc) (reverseTable : GMap GoString GoString) (wf : ValueSnapshotValid s)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some left) (hright : row.arr[1]? = some right)
    (hlvalid : validSymbolRef s left) (hrvalid : validSymbolRef s right)
    (hlvalue : valueResolve s m sf lf left = some c.leftValue)
    (hrvalue : valueResolve s m sf rf right = some c.rightValue)
    (hlsymbol : symbolResolve s sf left = some c.leftKey)
    (hrsymbol : symbolResolve s sf right = some c.rightKey)
    (hleftNonzero : c.leftKey ≠ W32 0) (hrightNonzero : c.rightKey ≠ W32 0)
    (hleftMetadata : ConstraintMetadata s c.leftKey c.leftKind c.leftChanging c.leftActive)
    (hrightMetadata : ConstraintMetadata s c.rightKey c.rightKind c.rightChanging c.rightActive)
    (ht : s.tracked.length = s.kinds.length)
    (hnill : c.leftValue.Nil' = false) (hnilr : c.rightValue.Nil' = false)
    (hreverse : (reverseTable !! c.operator).getD (zero_val GoString) = c.reversed)
    (hnegate : (inverseTable !! rawOperator).getD (zero_val GoString) = c.operator)
    (hparse : parseInt64Contract (GF := GF)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
      (globalAddr inverse ↦□ inverseLoc ∗ inverseLoc ↦${DFrac.discard} inverseTable ∗ globalAddr reverse ↦□ reverseLoc ∗ reverseLoc ↦${DFrac.discard} reverseTable ∗ querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"comparison" #receiver)) (Val #ref)) (Val #rawOperator)) (Val #false)) (Val #state))
    {{ RET #(if comparisonRejected c m then map.nil else state);
      globalAddr inverse ↦□ inverseLoc ∗ inverseLoc ↦${DFrac.discard} inverseTable ∗ globalAddr reverse ↦□ reverseLoc ∗ reverseLoc ↦${DFrac.discard} reverseTable ∗
        querySnapshotOwn receiver s ∗ state ↦$ (comparisonResultMap c m) }} := by
  dsimp only [comparisonRejected, comparisonResultMap, comparisonRightRejected, comparisonRightMap, comparisonLeftRejected, comparisonLeftMap]
  wp_start as ⟨#HinvGlobal, #HinvOperators, #Hglobal, #Hoperators, #Hsnapshot, Hstate⟩
  ihave %Hnonnull := ownMap_not_nil state m (DFrac.own 1) $$ Hstate
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, _, _, Hrows, _, _, _, _, _, _, _, _, _, _, _, _, _⟩
  try wp_auto
  wp_apply wp_map_lookup1 go.string go.string inverseLoc inverseTable rawOperator DFrac.discard $$ [HinvOperators] with HinvOperators
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
  wp_apply wp_value_resolve receiver s state m sf lf left c.leftValue wf hlvalid hparse hlvalue
    $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
  · try iframe Hsnapshot Hstate
  try wp_auto
  wp_apply wp_value_resolve receiver s state m sf rf right c.rightValue wf hrvalid hparse hrvalue
    $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
  · try iframe Hsnapshot Hstate
  try wp_auto
  wp_apply wp_symbol_resolve receiver s sf left c.leftKey wf.symbols hlvalid hlsymbol
    $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  try wp_auto
  wp_if_destruct
  · exfalso; contradiction
  · wp_apply wp_constrain_snapshot_numeric_any receiver s state m c.leftKey c.leftKind c.operator c.leftChanging c.leftActive c.rightValue
      wf.unstable ht hleftMetadata hnilr $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
    · try iframe Hsnapshot Hstate
    by_cases hlreject : (c.leftActive && constraintRejected c.leftKind c.leftChanging c.leftKey m c.operator c.rightValue) = true
    ·
      simp only [hlreject, Bool.true_or, ite_true]
      try wp_auto
      wp_end
      all_goals try iframe HinvGlobal HinvOperators Hglobal Hoperators Hsnapshot Hstate
    · have hfalse : (c.leftActive && constraintRejected c.leftKind c.leftChanging c.leftKey m c.operator c.rightValue) = false := Bool.eq_false_iff.mpr hlreject
      simp only [hfalse, Bool.false_eq_true, Bool.false_or, ite_false, ite_true]
      try wp_auto
      wp_if_destruct
      · exfalso; contradiction
      · wp_apply wp_symbol_resolve receiver s sf right c.rightKey wf.symbols hrvalid hrsymbol
          $$ [Hsnapshot] with Hsnapshot
        · iframe Hsnapshot
        try wp_auto
        wp_if_destruct
        · exfalso; contradiction
        · try wp_auto
          wp_apply wp_map_lookup1 go.string go.string reverseLoc reverseTable c.operator DFrac.discard $$ [Hoperators] with Hoperators
          · iframe Hoperators
          rw [hreverse]
          try wp_auto
          wp_apply wp_constrain_snapshot_numeric_any receiver s state
            (constraintState c.leftActive c.leftKind c.leftChanging c.leftKey m c.operator c.rightValue)
            c.rightKey c.rightKind c.reversed c.rightChanging c.rightActive c.leftValue
            wf.unstable ht hrightMetadata hnill $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
          · try iframe Hsnapshot Hstate
          try wp_auto
          wp_end
          all_goals try iframe HinvGlobal HinvOperators Hglobal Hoperators Hsnapshot Hstate

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
