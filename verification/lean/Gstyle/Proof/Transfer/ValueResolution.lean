module
public import Gstyle.Proof.Transfer.ValueEvaluation
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

def valueChild [FfiSyntax] (s : QuerySnapshot) (ref : w32) : w32 :=
  ((s.operands[(uint.Z ref - 1).toNat]?.getD (zero_val (GoArray w32 2))).arr[0]?).getD (W32 0)

structure ValueSnapshotValid [FfiSyntax] (s : QuerySnapshot) : Prop where
  symbols : SymbolSnapshotValid s
  unstable : s.unstable.length = s.kinds.length
  children : ∀ ref, 0 < uint.Z ref ∧ uint.Z ref ≤ s.nodes.length → validSymbolRef s (valueChild s ref)

def valueResolve [FfiSyntax] (s : QuerySnapshot) (m : StateData) (sf : Nat) : Nat → w32 → Option Value
  | 0, _ => none
  | fuel + 1, ref =>
    if ref = W32 0 then some (zero_val Value) else
    match symbolResolve s sf ref with
    | none => none
    | some key =>
      if key ≠ W32 0 then
        if uint.Z key ≤ s.kinds.length then
          some (bindingValue (s.kinds[(uint.Z key - 1).toNat]?.getD (go!""))
            (s.unstable[(uint.Z key - 1).toNat]?.getD false) key m)
        else some (zero_val Value)
      else
        let op := decodedFact s (snapshotRecord s ref).Operator'
        let constant := snapshotConstant s ref
        if op = go!"parentheses" then valueResolve s m sf fuel (valueChild s ref)
        else if op = go!"conversion" then
          if constant.Integer' then some constant
          else (valueResolve s m sf fuel (valueChild s ref)).map (conversionResult (snapshotFact s ref (go!"type")))
        else
          let candidate := lengthCandidate s ref
          if candidate.Length' then
            (valueResolve s m sf fuel (valueChild s ref)).map (lengthResult candidate)
          else some candidate

theorem valueChild_row [FfiSyntax] (s : QuerySnapshot) (ref : w32) (wf : ValueSnapshotValid s)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.nodes.length) :
    ∃ row, s.operands[(uint.Z ref - 1).toNat]? = some row ∧ row.arr[0]? = some (valueChild s ref) := by
  have hi : (uint.Z ref - 1).toNat < s.operands.length := by rw [wf.symbols.operands]; omega
  let row := s.operands[(uint.Z ref - 1).toNat]'hi
  have hr : s.operands[(uint.Z ref - 1).toNat]? = some row := List.getElem?_eq_getElem hi
  have ha := wf.symbols.arity row (List.mem_of_getElem? hr)
  have hz : (0 : Nat) < row.arr.length := by omega
  have hc : row.arr[0]? = some row.arr[0] := List.getElem?_eq_getElem hz
  refine ⟨row, hr, ?_⟩
  simp only [valueChild, hr, Option.getD_some, hc]

theorem valueResolve_evaluation [FfiSyntax] (s : QuerySnapshot) (m : StateData) (sf fuel : Nat)
    (ref : w32) (result : Value) (wf : ValueSnapshotValid s) (hvalid : validSymbolRef s ref)
    (hresult : valueResolve s m sf fuel ref = some result) : ValueEvaluation s m sf ref result := by
  induction fuel generalizing ref result with
  | zero => simp only [valueResolve] at hresult; cases hresult
  | succ fuel ih =>
    by_cases hz : ref = W32 0
    · subst ref
      simp only [valueResolve, ite_true] at hresult
      cases hresult
      exact .zero
    · have href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.nodes.length := hvalid.resolve_left hz
      rw [valueResolve, ite_eq_right hz] at hresult
      cases hs : symbolResolve s sf ref with
      | none => simp only [hs] at hresult; cases hresult
      | some key =>
        rw [hs] at hresult
        dsimp only at hresult
        by_cases hk : key = W32 0
        · subst key
          rw [ite_eq_right (by simp)] at hresult
          try dsimp only at hresult
          obtain ⟨row, hrow, hchildref⟩ := valueChild_row s ref wf href
          have hc := wf.children ref href
          by_cases hp : decodedFact s (snapshotRecord s ref).Operator' = go!"parentheses"
          · rw [ite_eq_left hp] at hresult
            exact .parentheses ref (valueChild s ref) row result href hs hp hrow hchildref (ih _ _ hc hresult)
          · rw [ite_eq_right hp] at hresult
            by_cases hv : decodedFact s (snapshotRecord s ref).Operator' = go!"conversion"
            · rw [ite_eq_left hv] at hresult
              cases hn : (snapshotConstant s ref).Integer' with
              | false =>
                simp only [hn, Bool.false_eq_true, ite_false] at hresult
                cases hr : valueResolve s m sf fuel (valueChild s ref) with
                | none => simp only [hr, Option.map_none] at hresult; cases hresult
                | some source =>
                  simp only [hr, Option.map_some] at hresult
                  cases hresult
                  exact .conversionChild ref (valueChild s ref) row source href hs hv hn hrow hchildref (ih _ _ hc hr)
              | true =>
                simp only [hn, ite_true] at hresult
                cases hresult
                exact .conversionConstant ref href hs hv hn
            · rw [ite_eq_right hv] at hresult
              cases hl : (lengthCandidate s ref).Length' with
              | false =>
                simp only [hl, Bool.false_eq_true, ite_false] at hresult
                cases hresult
                exact .plain ref href hs hp hv hl
              | true =>
                simp only [hl, ite_true] at hresult
                cases hr : valueResolve s m sf fuel (valueChild s ref) with
                | none => simp only [hr, Option.map_none] at hresult; cases hresult
                | some source =>
                  simp only [hr, Option.map_some] at hresult
                  cases hresult
                  exact .lengthChild ref (valueChild s ref) row source href hs hp hv hl hrow hchildref (ih _ _ hc hr)
        · rw [ite_eq_left hk] at hresult
          by_cases hb : uint.Z key ≤ s.kinds.length
          · rw [ite_eq_left hb] at hresult
            have hpos : 0 < uint.Z key := by word
            have hi : (uint.Z key - 1).toNat < s.kinds.length := by omega
            have hui : (uint.Z key - 1).toNat < s.unstable.length := by rw [wf.unstable]; exact hi
            have hkind : s.kinds[(uint.Z key - 1).toNat]? = some (s.kinds[(uint.Z key - 1).toNat]?.getD (go!"")) := by
              rw [List.getElem?_eq_getElem hi]; rfl
            have hu : s.unstable[(uint.Z key - 1).toNat]? = some (s.unstable[(uint.Z key - 1).toNat]?.getD false) := by
              rw [List.getElem?_eq_getElem hui]; rfl
            cases hresult
            exact .bound ref key _ _ href hs ⟨hpos, hb⟩ hkind hu
          · rw [ite_eq_right hb] at hresult
            cases hresult
            exact .outOfRange ref key href hs (by omega)

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

theorem wp_value_resolve (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (sf fuel : Nat) (ref : w32) (result : Value) (wf : ValueSnapshotValid s)
    (hvalid : validSymbolRef s ref) (hparse : parseInt64Contract (GF := GF))
    (hresult : valueResolve s m sf fuel ref = some result) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"value" #receiver)) (Val #ref)) (Val #state))
    {{ RET #result; querySnapshotOwn receiver s ∗ state ↦$ m }} := by
  exact wp_value_evaluation receiver s state m sf ref result wf.symbols wf.unstable hparse
    (valueResolve_evaluation s m sf fuel ref result wf hvalid hresult)

end proof

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
