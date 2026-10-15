module
public import Gstyle.Proof.Transfer.Branch
public import Gstyle.Proof.Transfer.QuerySnapshot
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
open Iris.BI Iris.ProgramLogic
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
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

variable [pdg.Access.TypeAssumptions] [pdg.Tables.TypeAssumptions] [pdg.Extension.TypeAssumptions]

theorem querySnapshot_branchResources (receiver : Loc) (s : QuerySnapshot) :
    querySnapshotOwn (GF := GF) receiver s ⊢
      branchResources receiver s.view s.operands s.kinds s.unstable (querySnapshotOwn receiver s) DFrac.discard := by
  iintro #Hsnapshot
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, _, _, Hrows, Hkinds, Hunstable, _, _, _, _, _, _, _, _, _, _, _⟩
  unfold branchResources
  iframe Hview Hrows Hkinds Hunstable Hsnapshot

variable [Value.TypeAssumptions] [State.TypeAssumptions]

def snapshotChildContract (receiver : Loc) (s : QuerySnapshot)
    (kind : w32 → GoString) (changing : w32 → Bool) (child : w32) (truth : Bool)
    (step : StateData → Option StateData) : Prop :=
  ∀ (state : Loc) (m : StateData), validBranchState s.kinds s.unstable kind changing m →
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
      (querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"assume" #receiver)) (Val #child)) (Val #truth)) (Val #state))
    {{ (out : Loc), RET #out;
      querySnapshotOwn receiver s ∗ ownsBranch (step m) out ∗
      ⌜validBranchOption s.kinds s.unstable kind changing (step m)⌝ }}

theorem snapshot_child_to_branch (receiver : Loc) (s : QuerySnapshot)
    (kind : w32 → GoString) (changing : w32 → Bool) (child : w32) (truth : Bool)
    (step : StateData → Option StateData)
    (hchild : snapshotChildContract (GF := GF) receiver s kind changing child truth step) :
    branchChildContract (GF := GF) receiver s.view s.operands s.kinds s.unstable kind changing
      (querySnapshotOwn receiver s) child truth step DFrac.discard := by
  intro state m hm
  wp_start_folded as ⟨Hresources, Hstate⟩
  iunfold branchResources at Hresources
  icases Hresources with ⟨#Hview, #Hrows, #Hkinds, #Hunstable, #Hsnapshot⟩
  wp_apply (hchild state m hm) $$ [Hsnapshot Hstate] with %out ⟨#Hsnapshot, Hresult, %Hvalid⟩
  · iframe Hsnapshot Hstate
  wp_end
  unfold branchResources
  iframe Hview Hrows Hkinds Hunstable Hsnapshot Hresult
  ipureintro
  exact Hvalid

variable [FuncUnfold integerConstraint [] integerConstraint.impl] [interval.Assumptions]

theorem wp_join_snapshot (receiver : Loc) (s : QuerySnapshot)
    (kind : w32 → GoString) (changing : w32 → Bool) (previous next : Loc) (p n : StateData)
    (hlen : s.unstable.length = s.kinds.length)
    (hmeta : ∀ key, (p !! key).isSome ∨ (n !! key).isSome →
      (0 < uint.Z key ∧ uint.Z key ≤ s.kinds.length) ∧
      s.kinds[(uint.Z key - 1).toNat]? = some (kind key) ∧
      s.unstable[(uint.Z key - 1).toNat]? = some (changing key)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
      (querySnapshotOwn receiver s ∗ previous ↦$ p ∗ next ↦$ n) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"join" #receiver)) (Val #previous)) (Val #next))
    {{ (out : Loc), RET #out; ∃ om : StateData,
      (querySnapshotOwn receiver s ∗ previous ↦$ p ∗ next ↦$ n ∗ out ↦$ om) ∗
      ⌜(∀ key, (om !! key).isSome = ((p !! key).isSome || (n !! key).isSome)) ∧
        ∀ key x, (interval.contains (bounds (bindingValue (kind key) (changing key) key p)) x ∨
          interval.contains (bounds (bindingValue (kind key) (changing key) key n)) x) →
          interval.contains (bounds (bindingValue (kind key) (changing key) key om)) x⌝ }} := by
  wp_start_folded as ⟨#Hsnapshot, Hprevious, Hnext⟩
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, _, _, _, Hkinds, Hunstable, _, _, _, _, _, _, _, _, _, _, _⟩
  wp_apply wp_join_sound receiver s.view s.kinds s.unstable kind changing previous next p n hlen hmeta DFrac.discard
    $$ [Hview Hkinds Hunstable Hprevious Hnext] with %out ⟨%om, Hview, Hkinds, Hunstable, Hprevious, Hnext, Hout, %Hsound⟩
  · iframe Hview Hkinds Hunstable Hprevious Hnext
  wp_end
  iexists om
  iframe Hsnapshot Hprevious Hnext Hout
  ipureintro
  exact Hsound

variable [FuncUnfold copyState [] copyState.impl]

theorem wp_logical_snapshot (receiver : Loc) (s : QuerySnapshot)
    (kind : w32 → GoString) (changing : w32 → Bool) (state : Loc) (m : StateData)
    (ref : w32) (op : GoString) (truth : Bool) (row : GoArray w32 2)
    (leftRef rightRef : w32) (left right : StateData → Option StateData)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hl : row.arr[0]? = some leftRef) (hr : row.arr[1]? = some rightRef)
    (hlen : s.unstable.length = s.kinds.length)
    (hm : validBranchState s.kinds s.unstable kind changing m)
    (hleft : snapshotChildContract (GF := GF) receiver s kind changing leftRef truth left)
    (hright : snapshotChildContract (GF := GF) receiver s kind changing rightRef truth right) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
      (querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"logical" #receiver)) (Val #ref)) (Val #op)) (Val #truth)) (Val #state))
    {{ (out : Loc), RET #out; ∃ result : Option StateData,
      querySnapshotOwn receiver s ∗ ownsBranch result out ∗
      (if sequentialBranch op truth then True else state ↦$ m) ∗
      ⌜branchResult kind changing op truth left right m result ∧ validBranchOption s.kinds s.unstable kind changing result⌝ }} := by
  have leftContract := snapshot_child_to_branch receiver s kind changing leftRef truth left hleft
  have rightContract := snapshot_child_to_branch receiver s kind changing rightRef truth right hright
  wp_start_folded as ⟨#Hsnapshot, Hstate⟩
  ihave Hresources := querySnapshot_branchResources receiver s $$ Hsnapshot
  wp_apply wp_logical receiver s.view s.operands s.kinds s.unstable kind changing (querySnapshotOwn receiver s)
    state m ref op truth row leftRef rightRef left right href hrow hl hr hlen hm leftContract rightContract
    $$ [Hresources Hstate] with %out ⟨%result, Hresources, Hresult, Hsource, %Hspec⟩
  · iframe Hresources Hstate
  wp_end
  iexists result
  iunfold branchResources at Hresources
  icases Hresources with ⟨_, _, _, _, Hsnapshot⟩
  iframe Hsnapshot Hresult Hsource
  ipureintro
  exact Hspec

theorem wp_logical_snapshot_sound (receiver : Loc) (s : QuerySnapshot)
    (kind : w32 → GoString) (changing : w32 → Bool) (state : Loc) (m : StateData)
    (ref : w32) (op : GoString) (truth : Bool) (row : GoArray w32 2)
    (leftRef rightRef : w32) (left right : StateData → Option StateData)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hl : row.arr[0]? = some leftRef) (hr : row.arr[1]? = some rightRef)
    (hlen : s.unstable.length = s.kinds.length)
    (hm : validBranchState s.kinds s.unstable kind changing m)
    (hleft : snapshotChildContract (GF := GF) receiver s kind changing leftRef truth left)
    (hright : snapshotChildContract (GF := GF) receiver s kind changing rightRef truth right)
    (env : w32 → Int) (lhs rhs : Bool)
    (hop : op = go!"&&" ∨ op = go!"||")
    (htruth : (if op = go!"&&" then lhs && rhs else lhs || rhs) = truth)
    (hin : stateDenotes kind changing m env)
    (hleftSound : ∀ m, stateDenotes kind changing m env → lhs = truth → branchDenotes kind changing (left m) env)
    (hrightSound : ∀ m, stateDenotes kind changing m env → rhs = truth → branchDenotes kind changing (right m) env) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
      (querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"logical" #receiver)) (Val #ref)) (Val #op)) (Val #truth)) (Val #state))
    {{ (out : Loc), RET #out; ∃ result : Option StateData,
      querySnapshotOwn receiver s ∗ ownsBranch result out ∗
      (if sequentialBranch op truth then True else state ↦$ m) ∗
      ⌜branchDenotes kind changing result env ∧ validBranchOption s.kinds s.unstable kind changing result⌝ }} := by
  wp_start_folded as Hinputs
  wp_apply wp_logical_snapshot receiver s kind changing state m ref op truth row leftRef rightRef left right
    href hrow hl hr hlen hm hleft hright $$ [Hinputs] with %out ⟨%result, Hsnapshot, Hresult, Hsource, %Hspec⟩
  · iexact Hinputs
  wp_end
  iexists result
  iframe Hsnapshot Hresult Hsource
  ipureintro
  exact ⟨branch_result_sound kind changing op truth left right m result env lhs rhs hop htruth hin hleftSound hrightSound Hspec.1, Hspec.2⟩

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
