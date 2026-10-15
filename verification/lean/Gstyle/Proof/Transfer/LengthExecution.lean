module
public import Gstyle.Proof.Transfer.QuerySnapshot
public import Gstyle.Proof.Transfer.BindingRead
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
def lengthResult [FfiSyntax] (value source : Value) : Value :=
  if source.HasExtent' then
    { (zero_val Value) with Length' := true, Integer' := true, HasLower' := true, HasUpper' := true, Lower' := source.Extent', Upper' := source.Extent' }
  else value

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

theorem wp_lengthValue_not_length (receiver : Loc) (ref : w32) (state : Loc) (value : Value)
    (h : value.Length' = false) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"lengthValue" #receiver)) (Val #ref)) (Val #state)) (Val #value))
    {{ RET #value; True }} := by
  wp_start
  wp_auto
  rw [h]
  wp_auto
  wp_end

theorem wp_value_zero (receiver : Loc) (state : Loc) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"value" #receiver)) (Val #(W32 0))) (Val #state))
    {{ RET #(zero_val Value); True }} := by
  wp_start
  wp_auto
  wp_end

theorem wp_conversionValue_integer (receiver : Loc) (ref : w32) (state : Loc) (value : Value)
    (h : value.Integer' = true) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"conversionValue" #receiver)) (Val #ref)) (Val #state)) (Val #value))
    {{ RET #value; True }} := by
  wp_start
  wp_auto
  rw [h]
  wp_auto
  wp_end

theorem wp_lengthValue_child (receiver : Loc) (s : QuerySnapshot) (ref child : w32)
    (state : Loc) (value source : Value) (row : GoArray w32 2) (extra : IProp GF)
    (hlen : value.Length' = true)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hchildref : row.arr[0]? = some child)
    (hchild :
      {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ extra) }}
        (App (App (Val #(methods (go.PointerType graphView.ty) go!"value" #receiver)) (Val #child)) (Val #state))
      {{ RET #source; querySnapshotOwn receiver s ∗ extra }}) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ extra) }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"lengthValue" #receiver)) (Val #ref)) (Val #state)) (Val #value))
    {{ RET #(lengthResult value source); querySnapshotOwn receiver s ∗ extra }} := by
  wp_start as ⟨#Hsnapshot, Hextra⟩
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, _, _, Hrows, _, _, _, _, _, _, _, _, _, _, _, _, _⟩
  try wp_auto
  rw [hlen]
  try wp_auto
  wp_apply wp_operand_readonly receiver s.view s.operands ref (W64 0) row child href (by decide) hrow hchildref DFrac.discard DFrac.discard
    $$ [Hview Hrows] with ⟨Hview, Hrows⟩
  · iframe Hview Hrows
  try wp_auto
  wp_apply hchild $$ [Hsnapshot Hextra] with ⟨Hsnapshot, Hextra⟩
  · iframe Hsnapshot Hextra
  try wp_auto
  unfold lengthResult
  cases he : source.HasExtent'
  all_goals
    simp only [Bool.false_eq_true, ite_false, ite_true]
    try wp_auto
    wp_end

theorem wp_lengthValue_zero_child (receiver : Loc) (s : QuerySnapshot) (ref : w32)
    (state : Loc) (value : Value) (row : GoArray w32 2) (extra : IProp GF)
    (hlen : value.Length' = true)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hchildref : row.arr[0]? = some (W32 0)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ extra) }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"lengthValue" #receiver)) (Val #ref)) (Val #state)) (Val #value))
    {{ RET #value; querySnapshotOwn receiver s ∗ extra }} := by
  have hchild :
      {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ extra) }}
        (App (App (Val #(methods (go.PointerType graphView.ty) go!"value" #receiver)) (Val #(W32 0))) (Val #state))
      {{ RET #(zero_val Value); querySnapshotOwn receiver s ∗ extra }} := by
    wp_start_folded as Hresources
    wp_apply wp_value_zero receiver state
    wp_end
  wp_start_folded as Hresources
  wp_apply wp_lengthValue_child receiver s ref (W32 0) state value (zero_val Value) row extra hlen href hrow hchildref hchild
    $$ [Hresources] with Hresources
  · iexact Hresources
  have hz : (zero_val Value).HasExtent' = false := rfl
  simp only [lengthResult, hz, Bool.false_eq_true, ite_false]
  wp_end

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
