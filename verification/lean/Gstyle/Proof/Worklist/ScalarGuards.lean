module
public import Gstyle.Proof.Worklist.Successor
public import Gstyle.Proof.Worklist.Layout
public import Gstyle.Proof.Transfer.Assignment
public import Gstyle.Proof.Transfer.LocalInvalidation
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




/-- The only unsigned-parser behavior needed by scalar transfer. This is an
    explicit standard-library premise, not an asserted axiom or a proved body. -/
def parseEmptyUint32Contract : Prop :=
  {{ isPkgInit (PROP := IProp GF) pkg_id.strconv }}
    (App (App (App (Val (@! strconv.ParseUint)) (Val #(go!""))) (Val #(W64 10))) (Val #(W64 32)))
  {{ (value : w64) (err : GoError), RET (PairV #value #err); ⌜err ≠ interface.nil⌝ }}

theorem wp_incrementValue_scalar (receiver : Loc) (s : QuerySnapshot)
    (ref symbol : w32) (fallback : Value) (state : Loc)
    (hvalid : validSymbolRef s ref)
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46)
    (hstep : snapshotFact s ref (go!"signed-step64") = go!"") :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ querySnapshotOwn receiver s }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"incrementValue" #receiver))
        (Val #ref)) (Val #symbol)) (Val #fallback)) (Val #state))
    {{ RET #fallback; querySnapshotOwn receiver s }} := by
  wp_start as #Hsnapshot
  try wp_auto
  wp_apply wp_fact_snapshot receiver s ref (go!"signed-step64") hvalid halloc
      $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  rw [hstep]
  try wp_auto
  wp_end
  all_goals iframe Hsnapshot

theorem wp_interfaceAssignment_scalar (receiver : Loc) (s : QuerySnapshot)
    (ref symbol : w32) (value : Value)
    (hk : 0 < uint.Z symbol ∧ uint.Z symbol ≤ s.kinds.length)
    (hkind : s.kinds[(uint.Z symbol - 1).toNat]? = some (go!"\"signed\"")) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ querySnapshotOwn receiver s }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"interfaceAssignment" #receiver))
        (Val #ref)) (Val #symbol)) (Val #value))
    {{ RET #value; querySnapshotOwn receiver s }} := by
  wp_start as #Hsnapshot
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, _, _, _, Hkinds, _, _, _, _, _, _, _, _, _, _, _, _⟩
  ihave %Hlen := ownSlice_len $$ Hkinds
  try wp_auto
  have Hi : sint.Z (W64 (uint.Z (symbol - W32 1))) = uint.Z symbol - 1 := by
    obtain ⟨_, _⟩ := hk; word
  rw [Hi]
  rw [ite_eq_left (by obtain ⟨_, _⟩ := hk; constructor <;> word :
    0 ≤ uint.Z symbol - 1 ∧ uint.Z symbol - 1 < sint.Z s.view.kinds'.len)]
  wp_apply wp_load_slice_index s.view.kinds' (uint.Z symbol - 1) s.kinds DFrac.discard
      (go!"\"signed\"") (by obtain ⟨_, _⟩ := hk; word) $$ [Hkinds] with Hkinds
  · iframe Hkinds; ipureintro; exact hkind
  try wp_auto
  wp_end
  all_goals iframe Hsnapshot

variable [analysis.TypeAssumptions]

theorem wp_rangeWrite_scalar (receiver : Loc) (a : analysis) (s : QuerySnapshot)
    (ref symbol : w32) (state : Loc) (m : StateData)
    (hvalid : validSymbolRef s ref)
    (halloc : ∀ b ∈ s.attributes, (pdg.rowContents s.referenceRows b.Outcomes').length ≤ 2^46)
    (habsent : snapshotFact s ref (go!"range-index") = go!"")
    (hparse : parseEmptyUint32Contract (GF := GF)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.strconv ∗
       (receiver ↦□ a ∗ querySnapshotOwn a.view' s ∗ state ↦$ m) }}
      (App (App (App (Val #(methods (go.PointerType analysis.ty) go!"rangeWrite" #receiver))
        (Val #ref)) (Val #symbol)) (Val #state))
    {{ RET #(); receiver ↦□ a ∗ querySnapshotOwn a.view' s ∗ state ↦$ m }} := by
  wp_start as ⟨#Ha, #Hsnapshot, Hstate⟩
  try wp_auto
  wp_apply wp_fact_snapshot a.view' s ref (go!"range-index") hvalid halloc
      $$ [Hsnapshot] with Hsnapshot
  · iframe Hsnapshot
  rw [habsent]
  unfold parseEmptyUint32Contract at hparse
  wp_apply hparse with %value %err %herr
  try wp_auto
  cases err with
  | nil => exact False.elim (herr rfl)
  | ok err =>
    try wp_auto
    wp_end
    all_goals iframe Ha Hsnapshot Hstate

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
