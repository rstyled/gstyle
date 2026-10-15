module
public import Gstyle.Proof.Transfer.SymbolExecution
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



theorem wp_symbol_resolve (receiver : Loc) (s : QuerySnapshot) (fuel : Nat) (ref result : w32)
    (wf : SymbolSnapshotValid s) (hvalid : validSymbolRef s ref)
    (hresolve : symbolResolve s fuel ref = some result) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ querySnapshotOwn receiver s }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"symbol" #receiver)) (Val #ref))
    {{ RET #result; querySnapshotOwn receiver s }} := by
  induction fuel generalizing ref result with
  | zero => simp only [symbolResolve] at hresolve; contradiction
  | succ fuel ih =>
    by_cases hz : ref = W32 0
    · subst ref
      rw [symbolResolve, symbolStep_zero] at hresolve
      have hr := (Option.some.inj hresolve).symm
      subst result
      exact wp_symbol_snapshot_zero receiver s
    · have href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.nodes.length := hvalid.resolve_left hz
      have hf : 0 < uint.Z ref ∧ uint.Z ref ≤ s.fields.length := by rw [wf.fields]; exact href
      have hfi : (uint.Z ref - 1).toNat < s.fields.length := by omega
      let cached := s.fields[(uint.Z ref - 1).toNat]'hfi
      have hcache : s.fields[(uint.Z ref - 1).toNat]? = some cached := List.getElem?_eq_getElem hfi
      by_cases hc : cached = W32 0
      · have hfield : s.fields[(uint.Z ref - 1).toNat]? = some (W32 0) := by rw [hcache, hc]
        have hni : (uint.Z ref - 1).toNat < s.nodes.length := by omega
        let node := s.nodes[(uint.Z ref - 1).toNat]'hni
        have hnode : s.nodes[(uint.Z ref - 1).toNat]? = some node := List.getElem?_eq_getElem hni
        by_cases ha : 0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ s.attributes.length
        · have hai : (uint.Z node.Attributes' - 1).toNat < s.attributes.length := by omega
          let attrs := s.attributes[(uint.Z node.Attributes' - 1).toNat]'hai
          have hrecord : s.attributes[(uint.Z node.Attributes' - 1).toNat]? = some attrs := List.getElem?_eq_getElem hai
          have ha0 : node.Attributes' ≠ W32 0 := by obtain ⟨_, _⟩ := ha; word
          have hbase := symbolStep_present s ref node attrs hz hfield hnode ha0 hrecord
          have halloc := wf.allocation attrs (List.mem_of_getElem? hrecord)
          cases hread : directReadValue node (pdg.rowContents s.accessRows attrs.Reads') with
          | false =>
            by_cases hstatus : attrs.Operator'.Status' = 1#8
            · by_cases hop : pdg.tableText s.strings attrs.Operator'.Value' = go!"parentheses"
              · have hoperand : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length := by rw [wf.operands]; exact href
                have hoi : (uint.Z ref - 1).toNat < s.operands.length := by omega
                let row := s.operands[(uint.Z ref - 1).toNat]'hoi
                have hrow : s.operands[(uint.Z ref - 1).toNat]? = some row := List.getElem?_eq_getElem hoi
                have harity := wf.arity row (List.mem_of_getElem? hrow)
                have hzero : (0 : Nat) < row.arr.length := by omega
                let child := row.arr[0]'hzero
                have hchildref : row.arr[0]? = some child := List.getElem?_eq_getElem hzero
                have hstep : symbolStep s ref = .inr child :=
                  hbase.trans (symbolRecordStep_parentheses s ref child node attrs row hread hstatus hop hrow hchildref)
                rw [symbolResolve, hstep] at hresolve
                have hchild := ih child result (wf.children ref child hstep) hresolve
                exact wp_symbol_snapshot_parentheses receiver s ref child result node attrs row hf hfield href hnode ha hrecord halloc hread hstatus hop hoperand hrow hchildref hchild
              · have hstep : symbolStep s ref = .inl (W32 0) :=
                  hbase.trans (symbolRecordStep_other s ref node attrs hread hstatus hop)
                rw [symbolResolve, hstep] at hresolve
                have hr := (Option.some.inj hresolve).symm
                subst result
                exact wp_symbol_snapshot_known_other receiver s ref node attrs hf hfield href hnode ha hrecord halloc hread hstatus hop
            · have hstep : symbolStep s ref = .inl (W32 0) :=
                hbase.trans (symbolRecordStep_unavailable s ref node attrs hread hstatus)
              rw [symbolResolve, hstep] at hresolve
              have hr := (Option.some.inj hresolve).symm
              subst result
              exact wp_symbol_snapshot_unavailable_operator receiver s ref node attrs hf hfield href hnode ha hrecord halloc hread hstatus
          | true =>
            obtain ⟨entry, hreads, hcategory, hlocation⟩ := directReadValue_true node (pdg.rowContents s.accessRows attrs.Reads') hread
            have hstep : symbolStep s ref = .inl (boundedReadSymbol (sint.Z s.view.kinds'.len) entry.Symbol') :=
              hbase.trans (symbolRecordStep_direct s ref node attrs entry hreads hcategory hlocation)
            rw [symbolResolve, hstep] at hresolve
            have hr := (Option.some.inj hresolve).symm
            subst result
            exact wp_symbol_snapshot_direct receiver s ref node attrs hf hfield href hnode ha hrecord halloc entry hreads hcategory hlocation
        · have hstep := symbolStep_absent s ref node hz hfield hnode ha
          rw [symbolResolve, hstep] at hresolve
          have hr := (Option.some.inj hresolve).symm
          subst result
          exact wp_symbol_snapshot_absent receiver s ref node hf hfield href hnode ha
      · have hstep := symbolStep_cached s ref cached hz hcache hc
        rw [symbolResolve, hstep] at hresolve
        have hr := (Option.some.inj hresolve).symm
        subst result
        exact wp_symbol_snapshot_cached receiver s ref cached hf hcache hc

theorem wp_symbol_ranked (receiver : Loc) (s : QuerySnapshot) (rank : w32 → Nat) (ref : w32)
    (wf : SymbolSnapshotValid s) (hvalid : validSymbolRef s ref)
    (hdecrease : ∀ current child, symbolStep s current = .inr child → rank child < rank current) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ querySnapshotOwn receiver s }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"symbol" #receiver)) (Val #ref))
    {{ (result : w32), RET #result;
       querySnapshotOwn receiver s ∗ ⌜symbolResolve s (rank ref + 1) ref = some result⌝ }} := by
  obtain ⟨result, hresolve⟩ := symbolResolve_rank s rank hdecrease ref
  wp_start_folded as Hsnapshot
  wp_apply wp_symbol_resolve receiver s (rank ref + 1) ref result wf hvalid hresolve $$ [Hsnapshot] with Hsnapshot
  · iexact Hsnapshot
  wp_end

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
