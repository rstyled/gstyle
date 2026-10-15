module
public import Gstyle.Proof.Transfer.LengthExecution
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
def conversionResult [FfiSyntax] (kind : GoString) (source : Value) : Value :=
  if kind = go!"\"pointer\"" then
    { (zero_val Value) with NonNil' := source.NonNil', Nil' := source.Nil' }
  else if kind = go!"\"unsigned\"" then
    { (zero_val Value) with HasLower' := true, Lower' := W64 0, Length' := source.Length' }
  else { (zero_val Value) with Length' := source.Length' }

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

theorem wp_conversionValue_child (receiver : Loc) (s : QuerySnapshot) (ref child : w32)
    (state : Loc) (value source : Value) (row : GoArray w32 2) (extra : IProp GF)
    (hinteger : value.Integer' = false)
    (hvalid : validSymbolRef s ref)
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hchildref : row.arr[0]? = some child)
    (hchild :
      {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ extra) }}
        (App (App (Val #(methods (go.PointerType graphView.ty) go!"value" #receiver)) (Val #child)) (Val #state))
      {{ RET #source; querySnapshotOwn receiver s ∗ extra }}) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ extra) }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"conversionValue" #receiver)) (Val #ref)) (Val #state)) (Val #value))
    {{ RET #(conversionResult (snapshotFact s ref (go!"type")) source); querySnapshotOwn receiver s ∗ extra }} := by
  wp_start as ⟨#Hsnapshot, Hextra⟩
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, _, _, Hrows, _, _, _, _, _, _, _, _, _, _, _, _, _⟩
  try wp_auto
  rw [hinteger]
  try wp_auto
  wp_apply wp_operand_readonly receiver s.view s.operands ref (W64 0) row child href (by decide) hrow hchildref DFrac.discard DFrac.discard
    $$ [Hview Hrows] with ⟨Hview, Hrows⟩
  · iframe Hview Hrows
  try wp_auto
  wp_apply hchild $$ [Hsnapshot Hextra] with ⟨Hsnapshot, Hextra⟩
  · iframe Hsnapshot Hextra
  try wp_auto
  wp_apply wp_fact_snapshot receiver s ref (go!"type") hvalid halloc $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  try wp_auto
  unfold conversionResult
  by_cases hp : snapshotFact s ref (go!"type") = go!"\"pointer\""
  · rw [ite_eq_left hp]
    simp only [hp, _root_.decide_true]
    try wp_auto
    wp_end
  · rw [ite_eq_right hp]
    simp only [hp, _root_.decide_false]
    try wp_auto
    by_cases hu : snapshotFact s ref (go!"type") = go!"\"unsigned\""
    · rw [ite_eq_left hu]
      simp only [hu, _root_.decide_true]
      try wp_auto
      wp_end
    · rw [ite_eq_right hu]
      simp only [hu, _root_.decide_false]
      try wp_auto
      wp_end

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
