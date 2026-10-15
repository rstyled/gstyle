module
public import Gstyle.Proof.Transfer.BindingRead
public import Gstyle.Proof.Transfer.FactSnapshot
public import Gstyle.Proof.Transfer.Boolean
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
variable [FuncUnfold booleanContradiction [] booleanContradiction.impl]

variable [interval.Assumptions]



theorem wp_assume_parentheses (receiver : Loc) (s : QuerySnapshot) (ref child : w32)
    (truth : Bool) (state : Loc) (P : IProp GF) (Q : Loc → IProp GF)
    (hvalid : validSymbolRef s ref)
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46)
    (hguard : booleanConflict (snapshotFact s ref (go!"boolean")) truth = false)
    (hop : decodedFact s (snapshotRecord s ref).Operator' = go!"parentheses")
    (row : GoArray w32 2)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row) (hc : row.arr[0]? = some child)
    (hnext :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
       (querySnapshotOwn receiver s ∗ P) }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"assume" #receiver)) (Val #child)) (Val #truth)) (Val #state))
    {{ (out : Loc), RET #out; Q out }}) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
       (querySnapshotOwn receiver s ∗ P) }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"assume" #receiver)) (Val #ref)) (Val #truth)) (Val #state))
    {{ (out : Loc), RET #out; Q out }} := by
  wp_start as ⟨#Hsnapshot, HP⟩
  try wp_auto
  wp_apply wp_fact_snapshot receiver s ref (go!"boolean") hvalid halloc $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  try wp_auto
  wp_apply wp_booleanContradiction (snapshotFact s ref (go!"boolean")) truth
  try wp_auto
  rw [hguard]
  try wp_auto
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, Hgraph, _, Hrows, _, _, _, _, _, _, _, _, _, _, _, _, _⟩
  try wp_auto
  wp_apply wp_attrs_snapshot receiver s ref hvalid halloc $$ [Hsnapshot] with %attrs ⟨Hsnapshot, %hattrs⟩
  · iframe Hsnapshot
  try wp_auto
  rw [hattrs.1]
  wp_apply wp_operatorText_snapshot receiver s (snapshotRecord s ref).Operator' $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  rw [hop]
  try wp_auto
  wp_apply wp_operand_readonly receiver s.view s.operands ref (W64 0) row child href (by decide) hrow hc DFrac.discard DFrac.discard
    $$ [Hview Hrows] with ⟨Hview, Hrows⟩
  · iframe Hview Hrows
  try wp_auto
  wp_apply hnext $$ [Hsnapshot HP] with %out Hresult
  · iframe Hsnapshot HP
  try wp_auto
  wp_end
  all_goals iexact Hresult

theorem wp_assume_negation (receiver : Loc) (s : QuerySnapshot) (ref child : w32)
    (truth : Bool) (state : Loc) (P : IProp GF) (Q : Loc → IProp GF)
    (hvalid : validSymbolRef s ref)
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46)
    (hguard : booleanConflict (snapshotFact s ref (go!"boolean")) truth = false)
    (hop : decodedFact s (snapshotRecord s ref).Operator' = go!"!")
    (row : GoArray w32 2)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row) (hc : row.arr[0]? = some child)
    (hnext :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
       (querySnapshotOwn receiver s ∗ P) }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"assume" #receiver)) (Val #child)) (Val #(!truth))) (Val #state))
    {{ (out : Loc), RET #out; Q out }}) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
       (querySnapshotOwn receiver s ∗ P) }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"assume" #receiver)) (Val #ref)) (Val #truth)) (Val #state))
    {{ (out : Loc), RET #out; Q out }} := by
  wp_start as ⟨#Hsnapshot, HP⟩
  try wp_auto
  wp_apply wp_fact_snapshot receiver s ref (go!"boolean") hvalid halloc $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  try wp_auto
  wp_apply wp_booleanContradiction (snapshotFact s ref (go!"boolean")) truth
  try wp_auto
  rw [hguard]
  try wp_auto
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, Hgraph, _, Hrows, _, _, _, _, _, _, _, _, _, _, _, _, _⟩
  try wp_auto
  wp_apply wp_attrs_snapshot receiver s ref hvalid halloc $$ [Hsnapshot] with %attrs ⟨Hsnapshot, %hattrs⟩
  · iframe Hsnapshot
  try wp_auto
  rw [hattrs.1]
  wp_apply wp_operatorText_snapshot receiver s (snapshotRecord s ref).Operator' $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  rw [hop]
  try wp_auto
  wp_apply wp_operand_readonly receiver s.view s.operands ref (W64 0) row child href (by decide) hrow hc DFrac.discard DFrac.discard
    $$ [Hview Hrows] with ⟨Hview, Hrows⟩
  · iframe Hview Hrows
  try wp_auto
  wp_apply hnext $$ [Hsnapshot HP] with %out Hresult
  · iframe Hsnapshot HP
  try wp_auto
  wp_end
  all_goals iexact Hresult

theorem wp_assume_logical (receiver : Loc) (s : QuerySnapshot) (ref : w32)
    (truth : Bool) (state : Loc) (op : GoString) (P : IProp GF) (Q : Loc → IProp GF)
    (hvalid : validSymbolRef s ref)
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46)
    (hguard : booleanConflict (snapshotFact s ref (go!"boolean")) truth = false)
    (hop : decodedFact s (snapshotRecord s ref).Operator' = op)
    (hlogical : op = go!"&&" ∨ op = go!"||")
    (hnext :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
       (querySnapshotOwn receiver s ∗ P) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"logical" #receiver)) (Val #ref)) (Val #op)) (Val #truth)) (Val #state))
    {{ (out : Loc), RET #out; Q out }}) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
       (querySnapshotOwn receiver s ∗ P) }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"assume" #receiver)) (Val #ref)) (Val #truth)) (Val #state))
    {{ (out : Loc), RET #out; Q out }} := by
  wp_start as ⟨#Hsnapshot, HP⟩
  try wp_auto
  wp_apply wp_fact_snapshot receiver s ref (go!"boolean") hvalid halloc $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  try wp_auto
  wp_apply wp_booleanContradiction (snapshotFact s ref (go!"boolean")) truth
  try wp_auto
  rw [hguard]
  try wp_auto
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, Hgraph, _, Hrows, _, _, _, _, _, _, _, _, _, _, _, _, _⟩
  try wp_auto
  wp_apply wp_attrs_snapshot receiver s ref hvalid halloc $$ [Hsnapshot] with %attrs ⟨Hsnapshot, %hattrs⟩
  · iframe Hsnapshot
  try wp_auto
  rw [hattrs.1]
  wp_apply wp_operatorText_snapshot receiver s (snapshotRecord s ref).Operator' $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  rw [hop]
  try wp_auto
  rcases hlogical with rfl | rfl
  all_goals try wp_auto
  all_goals
    wp_apply hnext $$ [Hsnapshot HP] with %out Hresult
    · iframe Hsnapshot HP
    try wp_auto
    wp_end
    all_goals iexact Hresult

theorem wp_assume_comparison (receiver : Loc) (s : QuerySnapshot) (ref : w32)
    (truth : Bool) (state : Loc) (op : GoString) (P : IProp GF) (Q : Loc → IProp GF)
    (hvalid : validSymbolRef s ref)
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46)
    (hguard : booleanConflict (snapshotFact s ref (go!"boolean")) truth = false)
    (hop : decodedFact s (snapshotRecord s ref).Operator' = op)
    (hparentheses : op ≠ go!"parentheses") (hnot : op ≠ go!"!")
    (hand : op ≠ go!"&&") (hor : op ≠ go!"||")
    (hnext :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
       (querySnapshotOwn receiver s ∗ P) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"comparison" #receiver)) (Val #ref)) (Val #op)) (Val #truth)) (Val #state))
    {{ (out : Loc), RET #out; Q out }}) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
       (querySnapshotOwn receiver s ∗ P) }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"assume" #receiver)) (Val #ref)) (Val #truth)) (Val #state))
    {{ (out : Loc), RET #out; Q out }} := by
  wp_start as ⟨#Hsnapshot, HP⟩
  try wp_auto
  wp_apply wp_fact_snapshot receiver s ref (go!"boolean") hvalid halloc $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  try wp_auto
  wp_apply wp_booleanContradiction (snapshotFact s ref (go!"boolean")) truth
  try wp_auto
  rw [hguard]
  try wp_auto
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, Hgraph, _, Hrows, _, _, _, _, _, _, _, _, _, _, _, _, _⟩
  try wp_auto
  wp_apply wp_attrs_snapshot receiver s ref hvalid halloc $$ [Hsnapshot] with %attrs ⟨Hsnapshot, %hattrs⟩
  · iframe Hsnapshot
  try wp_auto
  rw [hattrs.1]
  wp_apply wp_operatorText_snapshot receiver s (snapshotRecord s ref).Operator' $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  rw [hop]
  try wp_auto
  iterate 5 (try simp only [hparentheses, hnot, hand, hor, decide_false, ite_false]; try wp_auto)
  wp_apply hnext $$ [Hsnapshot HP] with %out Hresult
  · iframe Hsnapshot HP
  try wp_auto
  wp_end
  all_goals iexact Hresult

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
