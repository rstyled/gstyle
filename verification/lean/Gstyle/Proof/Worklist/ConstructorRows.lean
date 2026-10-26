module
public import Gstyle.Proof.Transfer.ValueResolution
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [FfiSyntax]

def constructorRefValid (count : Nat) (ref : w32) : Prop :=
  ref = W32 0 ∨ (0 < uint.Z ref ∧ uint.Z ref ≤ count)

/-- Shape and reference safety are maintained while the real edge loop writes slots. -/
def constructorRowsValid (count : Nat) (rows : List (GoArray w32 2)) : Prop :=
  rows.length = count ∧ ∀ row ∈ rows, row.arr.length = 2 ∧
    ∀ child ∈ row.arr, constructorRefValid count child

def constructorOperandSet (rows : List (GoArray w32 2)) (edge : pdg.Edge) : List (GoArray w32 2) :=
  let i := (uint.Z edge.Target' - 1).toNat
  let row := rows[i]?.getD (zero_val (GoArray w32 2))
  rows.set i (array.mk 2 (row.arr.set (uint.Z edge.Position' - 1).toNat edge.Source'))

theorem constructorRowsValid_zero (count : Nat) :
    constructorRowsValid count (List.replicate count (zero_val (GoArray w32 2))) := by
  constructor
  · simp
  · intro row hrow
    have hr : row = zero_val (GoArray w32 2) := (List.mem_replicate.mp hrow).2
    subst row
    constructor
    · rfl
    · intro child hc
      have hz : child = W32 0 := by
        change child ∈ [W32 0, W32 0] at hc
        simpa using hc
      exact Or.inl hz

theorem constructorRowsValid_set (count : Nat) (rows : List (GoArray w32 2)) (edge : pdg.Edge)
    (hrows : constructorRowsValid count rows)
    (htarget : 0 < uint.Z edge.Target' ∧ uint.Z edge.Target' ≤ count)
    (hsource : constructorRefValid count edge.Source') :
    constructorRowsValid count (constructorOperandSet rows edge) := by
  rcases hrows with ⟨hlen, hrows⟩
  have hi : (uint.Z edge.Target' - 1).toNat < rows.length := by omega
  obtain ⟨row, hr⟩ : ∃ row, rows[(uint.Z edge.Target' - 1).toNat]? = some row :=
    ⟨_, List.getElem?_eq_getElem hi⟩
  have hrow := hrows row (List.mem_of_getElem? hr)
  simp only [constructorOperandSet, hr, Option.getD_some]
  constructor
  · simpa using hlen
  · intro other ho
    rcases List.mem_or_eq_of_mem_set ho with hm | heq
    · exact hrows other hm
    · subst other
      constructor
      · simpa using hrow.1
      · intro child hc
        rcases List.mem_or_eq_of_mem_set hc with hm | heq
        · exact hrow.2 child hm
        · subst child; exact hsource

theorem constructorRowsValid_fold (count : Nat) (rows : List (GoArray w32 2)) (edges : List pdg.Edge)
    (hrows : constructorRowsValid count rows)
    (hedges : ∀ edge ∈ edges, (0 < uint.Z edge.Target' ∧ uint.Z edge.Target' ≤ count) ∧
      constructorRefValid count edge.Source') :
    constructorRowsValid count (edges.foldl constructorOperandSet rows) := by
  induction edges generalizing rows with
  | nil => exact hrows
  | cons edge edges ih =>
    have he := hedges edge (by simp)
    apply ih (constructorOperandSet rows edge) (constructorRowsValid_set count rows edge hrows he.1 he.2)
    intro e hm
    exact hedges e (by simp [hm])

/-- Both resolver child checks follow from the operand data built by the edge loop. -/
theorem constructorValueChild_valid (s : QuerySnapshot) (hrows : constructorRowsValid s.nodes.length s.operands)
    (ref : w32) : validSymbolRef s (valueChild s ref) := by
  unfold valueChild
  cases hr : s.operands[(uint.Z ref - 1).toNat]? with
  | none => exact Or.inl rfl
  | some row =>
    have hrow := hrows.2 row (List.mem_of_getElem? hr)
    simp only [Option.getD_some]
    cases hc : row.arr[0]? with
    | none => exact Or.inl rfl
    | some child =>
      simp only [Option.getD_some]
      exact hrow.2 child (List.mem_of_getElem? hc)

theorem constructorSymbolRecordChild_valid (s : QuerySnapshot) (ref : w32)
    (node : pdg.Node) (attrs : pdg.AttributeSet) (child : w32)
    (hrows : constructorRowsValid s.nodes.length s.operands)
    (hstep : symbolRecordStep s ref node attrs = .inr child) : validSymbolRef s child := by
  let operator := if attrs.Operator'.Status' = 1#8 then pdg.tableText s.strings attrs.Operator'.Value' else go!""
  change (if directReadValue node (pdg.rowContents s.accessRows attrs.Reads') then
      Sum.inl (boundedReadSymbol (sint.Z s.view.kinds'.len)
        ((pdg.rowContents s.accessRows attrs.Reads').head?.getD (zero_val pdg.Access)).Symbol')
    else if operator = go!"parentheses" then Sum.inr (valueChild s ref) else Sum.inl (W32 0)) = Sum.inr child at hstep
  split at hstep
  · cases hstep
  · by_cases hp : operator = go!"parentheses"
    · rw [ite_eq_left hp] at hstep
      cases hstep
      exact constructorValueChild_valid s hrows ref
    · rw [ite_eq_right hp] at hstep
      cases hstep

/-- No resolver postcondition is supplied: array shape and edge reference bounds derive it. -/
theorem constructorValueSnapshotValid (s : QuerySnapshot)
    (hfields : s.fields.length = s.nodes.length)
    (hrows : constructorRowsValid s.nodes.length s.operands)
    (hunstable : s.unstable.length = s.kinds.length)
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46) :
    ValueSnapshotValid s := by
  refine ⟨⟨hfields, hrows.1, ?_, halloc, ?_⟩, hunstable, ?_⟩
  · intro row hm
    exact (hrows.2 row hm).1
  · intro ref child hstep
    unfold symbolStep at hstep
    dsimp only at hstep
    split at hstep
    · cases hstep
    · split at hstep
      · cases hstep
      · split at hstep
        · cases hstep
        · split at hstep
          · cases hstep
          · split at hstep
            · cases hstep
            · exact constructorSymbolRecordChild_valid s ref _ _ child hrows hstep
  · intro ref _
    exact constructorValueChild_valid s hrows ref

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
