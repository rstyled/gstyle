module
public import Gstyle.Proof.Worklist.ValueDerivation
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [ext : FfiSyntax]

/-- Source meaning for each operation in order. The only semantic premises are
    primitive value facts; a child block/report soundness theorem is not stored.
    Source lowering still has to establish these facts for admitted programs. -/
inductive ScalarMeaningExecution (s : QuerySnapshot) (kind : w32 → GoString)
    (changing : w32 → Bool) :
    List w32 → StateData → (w32 → Int) → StateData → (w32 → Int) → Prop where
  | nil (m : StateData) (env : w32 → Int) : ScalarMeaningExecution s kind changing [] m env m env
  | noWrites (ref : w32) (tail : List w32) (m n : StateData) (env out : w32 → Int)
      (hvalid : validSymbolRef s ref)
      (hwrites : pdg.rowContents s.accessRows (snapshotRecord s ref).Writes' = [])
      (rest : ScalarMeaningExecution s kind changing tail m env n out) :
      ScalarMeaningExecution s kind changing (ref :: tail) m env n out
  | write (ref : w32) (tail : List w32) (m n : StateData) (env out values : w32 → Int)
      (site : ScalarWriteSite s ref) (raw : Value)
      (evaluation : ValueEvaluation s m site.sf site.child raw)
      (meaning : valueIntervalMeaning s kind changing env values)
      (stable : changing site.write.Symbol' = false)
      (rest : ScalarMeaningExecution s kind changing tail
        (<[site.write.Symbol' := { raw with Guarded' := false }]>m)
        (fun key => if key = site.write.Symbol' then values site.child else env key) n out) :
      ScalarMeaningExecution s kind changing (ref :: tail) m env n out

section proof
variable [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
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
variable [pdg.Access.TypeAssumptions] [pdg.Tables.TypeAssumptions] [pdg.Extension.TypeAssumptions]
variable [FuncUnfold directBindingRead [] directBindingRead.impl]
variable [FuncUnfold operatorText [] operatorText.impl]
variable [FuncUnfold extension [] extension.impl]
variable [Value.TypeAssumptions] [State.TypeAssumptions]
variable [analysis.TypeAssumptions]
variable {dq : DFrac}
variable [FuncUnfold fieldIdentity [] fieldIdentity.impl]
variable [FuncUnfold singleValueWrite [] singleValueWrite.impl]
variable [pdg.FlowBlock.TypeAssumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

theorem ScalarMeaningExecution.sound (s : QuerySnapshot) (kind : w32 → GoString)
    (changing : w32 → Bool) (operations : List w32) (m n : StateData) (env out : w32 → Int)
    (execution : ScalarMeaningExecution s kind changing operations m env n out)
    (hin : stateDenotes kind changing m env) : stateDenotes kind changing n out := by
  induction execution with
  | nil => exact hin
  | noWrites ref tail m n env out hvalid hwrites rest ih => exact ih hin
  | write ref tail m n env out values site raw evaluation meaning stable rest ih =>
    have hv := ValueEvaluation.denotes s kind changing m site.sf site.child raw evaluation env values meaning hin
    exact ih (scalar_store_sound kind changing m env site.write.Symbol' raw (values site.child) hin stable hv)

theorem ScalarMeaningExecution.erase (s : QuerySnapshot) (kind : w32 → GoString)
    (changing : w32 → Bool) (operations : List w32) (m n : StateData) (env out : w32 → Int)
    (execution : ScalarMeaningExecution s kind changing operations m env n out)
    (hm : scalarState m) (hkinds : ∀ k ∈ s.kinds, k ≠ go!"\"pointer\"") :
    ScalarExecution s operations m n := by
  induction execution with
  | nil => exact .nil _
  | noWrites ref tail m n env out hvalid hwrites rest ih =>
    exact .cons ref tail m m n (.noWrites ref m hvalid hwrites) (ih hm)
  | write ref tail m n env out values site raw evaluation meaning stable rest ih =>
    have hid := ValueEvaluation.scalar s m site.sf site.child raw evaluation hm hkinds
    exact .cons ref tail m _ n (.write ref m site raw evaluation hid)
      (ih (scalarState_insert m site.write.Symbol' raw hm hid))

def metadataAgreement (s : QuerySnapshot) (kind : w32 → GoString) (changing : w32 → Bool) : Prop :=
  ∀ key, 0 < uint.Z key ∧ uint.Z key ≤ s.kinds.length →
    s.kinds[(uint.Z key - 1).toNat]? = some (kind key) ∧
    s.unstable[(uint.Z key - 1).toNat]? = some (changing key)

theorem ScalarExecution.valid (s : QuerySnapshot) (kind : w32 → GoString)
    (changing : w32 → Bool) (operations : List w32) (m n : StateData)
    (execution : ScalarExecution s operations m n)
    (agreement : metadataAgreement s kind changing)
    (hm : validBranchState s.kinds s.unstable kind changing m) :
    validBranchState s.kinds s.unstable kind changing n := by
  induction execution with
  | nil => exact hm
  | cons ref tail m middle n step rest ih =>
    apply ih
    cases step with
    | noWrites => exact hm
    | write site raw evaluation identity =>
      exact scalar_store_valid s kind changing m site.write.Symbol' raw hm site.hk
        (agreement site.write.Symbol' site.hk).1 (agreement site.write.Symbol' site.hk).2

/-- Source-bound semantic composition of the actual solver block body. It
    establishes scalar identity, metadata and concrete range meaning together. -/
theorem wp_block_scalar_sound (receiver : Loc) (a : analysis) (s : QuerySnapshot)
    (kind : w32 → GoString) (changing : w32 → Bool)
    (block : pdg.FlowBlock) (operations : List w32) (state : Loc) (m n : StateData)
    (env out : w32 → Int) (execution : ScalarMeaningExecution s kind changing operations m env n out)
    (hm : scalarState m) (hkinds : ∀ k ∈ s.kinds, k ≠ go!"\"pointer\"")
    (hin : stateDenotes kind changing m env)
    (agreement : metadataAgreement s kind changing)
    (hmeta : validBranchState s.kinds s.unstable kind changing m)
    (wf : ValueSnapshotValid s) (hparse : parseInt64Contract (GF := GF))
    (huparse : parseEmptyUint32Contract (GF := GF)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.strconv ∗
       (receiver ↦{dq} a ∗ querySnapshotOwn a.view' s ∗ block.Operations' ↦*□ operations ∗
        localRootsOwn (GF := GF) s.view.fieldRoots' ∅ ∗ state ↦$ m) }}
      (App (App (App (Val #(methods (go.PointerType analysis.ty) go!"block" #receiver))
        (Val #block)) (Val #state)) (Val #null))
    {{ RET #(); receiver ↦{dq} a ∗ querySnapshotOwn a.view' s ∗
        localRootsOwn (GF := GF) s.view.fieldRoots' ∅ ∗ state ↦$ n ∗
        ⌜scalarState n ∧ stateDenotes kind changing n out ∧
          validBranchState s.kinds s.unstable kind changing n⌝ }} := by
  have erased := ScalarMeaningExecution.erase s kind changing operations m n env out execution hm hkinds
  have sound := ScalarMeaningExecution.sound s kind changing operations m n env out execution hin
  have valid := ScalarExecution.valid s kind changing operations m n erased agreement hmeta
  wp_start_folded as Hinputs
  wp_apply wp_block_scalar receiver a s block operations state m n erased hm wf hparse huparse
      $$ [Hinputs] with ⟨Ha, Hsnapshot, Hroots, Hstate, %hscalar⟩
  · iexact Hinputs
  iapply HΦ
  iframe Ha Hsnapshot Hroots Hstate
  ipureintro
  exact ⟨hscalar, sound, valid⟩

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
