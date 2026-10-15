module
public import Gstyle.Proof.StateMap.Copy
public import Gstyle.Proof.Pdg.GraphLayout
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

def localIdentity (identity symbols : w64) : Prop :=
  identity = W64 0 ∨ uint.Z identity % 2 ≠ 0 ∨ uint.Z identity / 2 ≤ uint.Z symbols

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

theorem wp_fieldIdentity_local (identity symbols : w64) (hlocal : localIdentity identity symbols)
    [FuncUnfold fieldIdentity [] fieldIdentity.impl] :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! fieldIdentity)) (Val #identity)) (Val #symbols))
    {{ RET #false; True }} := by
  wp_start
  wp_auto
  wp_if_destruct
  · wp_end
  · wp_if_destruct
    · wp_if_destruct
      · exfalso
        rcases hlocal with h | h | h <;> word
      · wp_end
    · wp_end

def localRootsOwn (roots : Loc) (m : GMap w32 w32) : IProp GF := iprop(
  ⌜roots = map.nil ∧ m = ∅⌝ ∨ roots ↦$□ m)

variable [pdg.query.Assumptions]

theorem wp_localRoot_lookup (roots : Loc) (m : GMap w32 w32) (key : w32)
    (hroot : (m !! key).getD (W32 0) = W32 0) :
    {{ localRootsOwn (GF := GF) roots m }}
      (App (App (Val (map.lookup1 pdg.Ref.ty pdg.Ref.ty)) (Val #roots)) (Val #key))
    {{ RET #(W32 0); localRootsOwn (GF := GF) roots m }} := by
  iintro %Φ Hroots HΦ
  iunfold localRootsOwn at Hroots
  icases Hroots with (⟨%hnil, %hm⟩ | Hroots)
  · subst roots
    wp_pures
    rw [show zero_val w32 = W32 0 from rfl]
    iapply HΦ
    iunfold localRootsOwn
    ileft
    ipureintro
    exact ⟨rfl, hm⟩
  · wp_apply wp_map_lookup1 pdg.Ref.ty pdg.Ref.ty roots m key DFrac.discard $$ Hroots with Hroots
    rw [show zero_val w32 = W32 0 from rfl]
    rw [hroot]
    iapply HΦ
    iunfold localRootsOwn
    iright
    iexact Hroots

variable [graphView.TypeAssumptions]

theorem wp_invalidatesField_local (receiver : Loc) (ref : w32) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"invalidatesField" #receiver)) (Val #ref)) (Val #(W32 0)))
    {{ RET #false; True }} := by
  wp_start
  wp_auto
  wp_end

variable [Value.TypeAssumptions] [State.TypeAssumptions]
variable [pdg.Graph.TypeAssumptions] [pdg.Node.TypeAssumptions] [pdg.Attributes.TypeAssumptions]
variable [pdg.AttributeSet.TypeAssumptions] [pdg.AttributeTables.TypeAssumptions]
variable [pdg.Producer.TypeAssumptions] [pdg.Function.TypeAssumptions] [pdg.Span.TypeAssumptions]
variable [pdg.CountFact.TypeAssumptions] [pdg.Conformance.TypeAssumptions] [pdg.Fact.TypeAssumptions]
variable [pdg.Position.TypeAssumptions] [pdg.CallForm.TypeAssumptions] [pdg.ExitKind.TypeAssumptions]
variable [pdg.ConformanceStatus.TypeAssumptions] [pdg.Category.TypeAssumptions] [pdg.Access.TypeAssumptions]

local instance local_graph_symbols_load (l : Loc) (g : pdg.Graph) (dq : DFrac) :
    AccessStrict (PROP := IProp GF)
      (typedPointsto (structFieldRef pdg.Graph go!"Symbols" l) g.Symbols' dq)
      (typedPointsto (structFieldRef pdg.Graph go!"Symbols" l) g.Symbols' dq)
      (typedPointsto l g dq) (typedPointsto l g dq) := by
  solve_pointsto_access_struct

theorem wp_invalidateFields_local (receiver : Loc) (view : graphView) (graph : pdg.Graph)
    (roots : GMap w32 w32) (state : Loc) (m : StateData) (ref : w32)
    [FuncUnfold fieldIdentity [] fieldIdentity.impl]
    (hlocal : ∀ key value, m !! key = some value →
      (roots !! key).getD (W32 0) = W32 0 ∧ localIdentity value.Identity' graph.Symbols'.len) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗
      (receiver ↦□ view ∗ view.graph' ↦□ graph ∗ localRootsOwn (GF := GF) view.fieldRoots' roots ∗ state ↦$ m) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"invalidateFields" #receiver)) (Val #ref)) (Val #state))
    {{ RET #(); receiver ↦□ view ∗ view.graph' ↦□ graph ∗ localRootsOwn (GF := GF) view.fieldRoots' roots ∗ state ↦$ m }} := by
  wp_start as ⟨Hview, Hgraph, Hroots, Hstate⟩
  wp_auto
  wp_func_lits
  wp_bind (App (App (Val (map.forRange _ _)) _) _)
  iapply (wp_map_for_range (fun _ _ => iprop(∃ (sv : w32) (vv : Value),
    symbol_ptr ↦ sv ∗ value_ptr ↦ vv ∗ v_ptr ↦ receiver ∗ ref_ptr ↦ ref ∗ state_ptr ↦ state ∗
    receiver ↦□ view ∗ view.graph' ↦□ graph ∗ localRootsOwn (GF := GF) view.fieldRoots' roots))) $$ Hstate
  iintro %keys %Hkeys
  isplitl [symbol value v ref state Hview Hgraph Hroots]
  · iexists _, _
    iframe
  isplitl []
  · imodintro
    iintro %i %key %value ⟨%Hkey, %Hlookup⟩
      ⟨%sv, %vv, Hsymbol, Hvalue, Hvptr, Hrptr, Hsptr, Hview, Hgraph, Hroots⟩
    obtain ⟨hroot, hidentity⟩ := hlocal key value Hlookup
    wp_auto
    wp_apply wp_localRoot_lookup view.fieldRoots' roots key hroot $$ [Hroots] with Hroots
    · iframe
    wp_apply wp_invalidatesField_local receiver ref
    try wp_auto
    wp_apply wp_fieldIdentity_local value.Identity' graph.Symbols'.len hidentity
    try wp_auto
    iapply map_body_done
    iexists key, value
    iframe
  iintro ⟨%sv, %vv, Hsymbol, Hvalue, Hvptr, Hrptr, Hsptr, Hview, Hgraph, Hroots⟩ Hstate
  wp_auto
  wp_end

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
