module
public import Gstyle.Proof.Worklist.ConstructorFact
public import Gstyle.Proof.Transfer.SymbolModel
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

/-- Scalar symbols have no cached field binding or recursive parentheses lookup.
Comparisons, assignments, reads, and branch operators remain available. -/
structure ScalarTrackingSymbols [FfiSyntax] (s : QuerySnapshot) : Prop where
  fieldsZero : s.fields = List.replicate s.nodes.length (W32 0)
  noParentheses : ∀ a ∈ s.attributes, decodedFact s a.Operator' ≠ go!"parentheses"
  outcomeBound : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46

def trackingSymbolBound (count : Nat) (symbol : w32) : Prop :=
  symbol = W32 0 ∨ (0 < uint.Z symbol ∧ uint.Z symbol ≤ count)

section execution
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [graphView.TypeAssumptions] [pdg.query.Assumptions]
variable [pdg.Graph.TypeAssumptions] [pdg.Node.TypeAssumptions] [pdg.Attributes.TypeAssumptions]
variable [pdg.AttributeSet.TypeAssumptions] [pdg.AttributeTables.TypeAssumptions]
variable [pdg.Producer.TypeAssumptions] [pdg.Function.TypeAssumptions] [pdg.Span.TypeAssumptions]
variable [pdg.CountFact.TypeAssumptions] [pdg.Conformance.TypeAssumptions] [pdg.Fact.TypeAssumptions]
variable [pdg.Position.TypeAssumptions] [pdg.CallForm.TypeAssumptions] [pdg.ExitKind.TypeAssumptions]
variable [pdg.ConformanceStatus.TypeAssumptions] [pdg.Category.TypeAssumptions]
variable [pdg.Access.TypeAssumptions] [pdg.Tables.TypeAssumptions] [pdg.Extension.TypeAssumptions]
variable [FuncUnfold pdg.listAt [pdg.Ref.ty] (pdg.listAt.impl pdg.Ref.ty)]
variable [FuncUnfold pdg.listAt [pdg.Access.ty] (pdg.listAt.impl pdg.Access.ty)]
variable [FuncUnfold pdg.listAt [pdg.Extension.ty] (pdg.listAt.impl pdg.Extension.ty)]
variable [FuncUnfold directBindingRead [] directBindingRead.impl]
variable [FuncUnfold operatorText [] operatorText.impl]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

/-- Execute the scalar symbol getter with mutable receiver permission. Its real
kind-count check bounds every nonzero result used to index definition rows. -/
theorem wp_trackingSymbol_scalar (receiver : Loc) (view : graphView) (s : QuerySnapshot)
    (ref : w32) (vd fd kd : DFrac) (hgraph : view.graph' = s.view.graph')
    (hdata : ScalarTrackingSymbols s) (hvalid : validSymbolRef s ref) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ receiver ↦{vd} view ∗ constructorGraphOwn s ∗
      view.fields' ↦*{fd} s.fields ∗ view.kinds' ↦*{kd} s.kinds }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"symbol" #receiver)) (Val #ref))
    {{ (symbol : w32), RET #symbol; receiver ↦{vd} view ∗ constructorGraphOwn s ∗
      view.fields' ↦*{fd} s.fields ∗ view.kinds' ↦*{kd} s.kinds ∗
      ⌜trackingSymbolBound s.kinds.length symbol⌝ }} := by
  wp_start_folded as ⟨Hview, #Hsnapshot, Hfields, Hkinds⟩
  ihave %Hklen := ownSlice_len $$ Hkinds
  ihave #Hctx := Hsnapshot
  iunfold constructorGraphOwn at Hctx
  icases Hctx with ⟨Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions,
    HrefRows, HaccessRows, _, Htable, Hstrings⟩
  by_cases hz : ref = W32 0
  · subst ref
    wp_apply wp_symbol_zero receiver
    wp_end
    all_goals iframe Hview Hsnapshot Hfields Hkinds
    all_goals ipureintro; exact Or.inl rfl
  · have href := hvalid.resolve_left hz
    have hf : 0 < uint.Z ref ∧ uint.Z ref ≤ s.fields.length := by
      rw [hdata.fieldsZero, List.length_replicate]; exact href
    have hi : (uint.Z ref - 1).toNat < s.nodes.length := by omega
    have hfield : s.fields[(uint.Z ref - 1).toNat]? = some (W32 0) := by
      simp only [hdata.fieldsZero, List.getElem?_replicate, ite_eq_left hi]
    let node := s.nodes[(uint.Z ref - 1).toNat]'hi
    have hnode : s.nodes[(uint.Z ref - 1).toNat]? = some node := List.getElem?_eq_getElem hi
    by_cases ha : 0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ s.attributes.length
    · have hai : (uint.Z node.Attributes' - 1).toNat < s.attributes.length := by omega
      let a := s.attributes[(uint.Z node.Attributes' - 1).toNat]'hai
      have hrecord : s.attributes[(uint.Z node.Attributes' - 1).toNat]? = some a := List.getElem?_eq_getElem hai
      have halloc := hdata.outcomeBound a (List.mem_of_getElem? hrecord)
      ihave Hvalues := pdg.readonlyRows_at_ref s.references s.referenceRows a.Outcomes' $$ HrefRows
      ihave Hreads := pdg.readonlyRows_at_ref s.accesses s.accessRows a.Reads' $$ HaccessRows
      cases hread : directReadValue node (pdg.rowContents s.accessRows a.Reads') with
      | false =>
        by_cases hs : a.Operator'.Status' = 1#8
        · have hop : pdg.tableText s.strings a.Operator'.Value' ≠ go!"parentheses" := by
            simpa only [decodedFact, hs, ite_true] using hdata.noParentheses a (List.mem_of_getElem? hrecord)
          wp_apply wp_symbol_known_other receiver view s.graph s.fields s.nodes node s.attributes a
            s.references s.accesses s.extensions (pdg.rowContents s.referenceRows a.Outcomes')
            (pdg.rowContents s.accessRows a.Reads') s.table s.strings ref
            vd DFrac.discard fd DFrac.discard DFrac.discard hf hfield href hnode ha hrecord halloc hread hs hop
            $$ [Hview Hfields Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues Hreads Htable Hstrings]
            with ⟨Hview, Hfields, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Hreads, Htable, Hstrings⟩
          · rw [hgraph]; iframe Hview Hfields Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues Hreads Htable Hstrings
          wp_end
          all_goals iframe Hview Hsnapshot Hfields Hkinds
          all_goals ipureintro; exact Or.inl rfl
        · wp_apply wp_symbol_unavailable_operator receiver view s.graph s.fields s.nodes node s.attributes a
            s.references s.accesses s.extensions (pdg.rowContents s.referenceRows a.Outcomes')
            (pdg.rowContents s.accessRows a.Reads') ref vd DFrac.discard fd DFrac.discard
            hf hfield href hnode ha hrecord halloc hread hs
            $$ [Hview Hfields Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues Hreads]
            with ⟨Hview, Hfields, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Hreads⟩
          · rw [hgraph]; iframe Hview Hfields Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues Hreads
          wp_end
          all_goals iframe Hview Hsnapshot Hfields Hkinds
          all_goals ipureintro; exact Or.inl rfl
      | true =>
        obtain ⟨entry, hreads, hc, hl⟩ := directReadValue_true node (pdg.rowContents s.accessRows a.Reads') hread
        isimp only [hreads] at Hreads
        wp_apply wp_symbol_direct receiver view s.graph s.fields s.nodes node s.attributes a
          s.references s.accesses s.extensions (pdg.rowContents s.referenceRows a.Outcomes') entry ref
          vd DFrac.discard fd DFrac.discard hf hfield href hnode ha hrecord halloc hc hl
          $$ [Hview Hfields Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues Hreads]
          with ⟨Hview, Hfields, Hgraph, Hnodes, Hrecords, Hreferences, Haccesses, Hextensions, Hvalues, Hreads⟩
        · rw [hgraph]; iframe Hview Hfields Hgraph Hnodes Hrecords Hreferences Haccesses Hextensions Hvalues Hreads
        wp_end
        all_goals iframe Hview Hsnapshot Hfields Hkinds
        all_goals ipureintro
        all_goals unfold trackingSymbolBound boundedReadSymbol
        all_goals split
        · exact Or.inl rfl
        · by_cases he : entry.Symbol' = W32 0
          · exact Or.inl he
          · right; constructor <;> word
    · wp_apply wp_symbol_absent_attributes receiver view s.graph s.fields s.nodes node s.attributes ref
        vd DFrac.discard fd hf hfield href hnode ha
        $$ [Hview Hfields Hgraph Hnodes Hrecords]
        with ⟨Hview, Hfields, Hgraph, Hnodes, Hrecords⟩
      · rw [hgraph]; iframe Hview Hfields Hgraph Hnodes Hrecords
      wp_end
      all_goals iframe Hview Hsnapshot Hfields Hkinds
      all_goals ipureintro; exact Or.inl rfl

end execution
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
