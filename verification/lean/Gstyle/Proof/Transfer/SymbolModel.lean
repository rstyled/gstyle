module
public import Gstyle.Proof.Transfer.QuerySnapshot
public import Gstyle.Proof.Transfer.SymbolRead
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [FfiSyntax]

def symbolRecordStep (s : QuerySnapshot) (ref : w32) (node : pdg.Node) (attrs : pdg.AttributeSet) : Sum w32 w32 :=
  let reads := pdg.rowContents s.accessRows attrs.Reads'
  if directReadValue node reads then
    .inl (boundedReadSymbol (sint.Z s.view.kinds'.len) (reads.head?.getD (zero_val pdg.Access)).Symbol')
  else
    let op := if attrs.Operator'.Status' = 1#8 then pdg.tableText s.strings attrs.Operator'.Value' else go!""
    if op = go!"parentheses" then
      .inr ((s.operands[(uint.Z ref - 1).toNat]?.getD (zero_val (GoArray w32 2))).arr[0]?.getD (W32 0))
    else .inl (W32 0)

def symbolStep (s : QuerySnapshot) (ref : w32) : Sum w32 w32 :=
  if ref = W32 0 then .inl (W32 0) else
  let i := (uint.Z ref - 1).toNat
  let cached := s.fields[i]?.getD (W32 0)
  if cached ≠ W32 0 then .inl cached else
  match s.nodes[i]? with
  | none => .inl (W32 0)
  | some node =>
    if node.Attributes' = W32 0 then .inl (W32 0) else
    match s.attributes[(uint.Z node.Attributes' - 1).toNat]? with
    | none => .inl (W32 0)
    | some attrs => symbolRecordStep s ref node attrs

def symbolResolve (s : QuerySnapshot) : Nat → w32 → Option w32
  | 0, _ => none
  | fuel + 1, ref =>
    match symbolStep s ref with
    | .inl value => some value
    | .inr child => symbolResolve s fuel child

theorem directReadValue_true (node : pdg.Node) (reads : List pdg.Access)
    (hread : directReadValue node reads = true) :
    ∃ entry, reads = [entry] ∧ node.Category' = W8 1 ∧ entry.Location' = W32 0 := by
  cases reads with
  | nil => simp only [directReadValue, Bool.false_eq_true] at hread
  | cons entry rest =>
    cases rest with
    | nil =>
      change decide (node.Category' = W8 1 ∧ entry.Location' = W32 0) = true at hread
      exact ⟨entry, rfl, of_decide_eq_true hread⟩
    | cons second tail => simp only [directReadValue, Bool.false_eq_true] at hread

theorem symbolRecordStep_direct (s : QuerySnapshot) (ref : w32) (node : pdg.Node) (a : pdg.AttributeSet)
    (entry : pdg.Access) (hreads : pdg.rowContents s.accessRows a.Reads' = [entry])
    (hc : node.Category' = W8 1) (hl : entry.Location' = W32 0) :
    symbolRecordStep s ref node a = .inl (boundedReadSymbol (sint.Z s.view.kinds'.len) entry.Symbol') := by
  have ht : directReadValue node [entry] = true := by
    unfold directReadValue
    exact decide_eq_true ⟨hc, hl⟩
  simp [symbolRecordStep, hreads, ht]

theorem symbolRecordStep_unavailable (s : QuerySnapshot) (ref : w32) (node : pdg.Node) (a : pdg.AttributeSet)
    (hr : directReadValue node (pdg.rowContents s.accessRows a.Reads') = false)
    (hs : a.Operator'.Status' ≠ 1#8) : symbolRecordStep s ref node a = .inl (W32 0) := by
  simp [symbolRecordStep, hr, hs]

theorem symbolRecordStep_other (s : QuerySnapshot) (ref : w32) (node : pdg.Node) (a : pdg.AttributeSet)
    (hr : directReadValue node (pdg.rowContents s.accessRows a.Reads') = false)
    (hs : a.Operator'.Status' = 1#8) (hop : pdg.tableText s.strings a.Operator'.Value' ≠ go!"parentheses") :
    symbolRecordStep s ref node a = .inl (W32 0) := by
  simp [symbolRecordStep, hr, hs, hop]

theorem symbolRecordStep_parentheses (s : QuerySnapshot) (ref child : w32) (node : pdg.Node)
    (a : pdg.AttributeSet) (row : GoArray w32 2)
    (hr : directReadValue node (pdg.rowContents s.accessRows a.Reads') = false)
    (hs : a.Operator'.Status' = 1#8) (hop : pdg.tableText s.strings a.Operator'.Value' = go!"parentheses")
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row) (hchild : row.arr[0]? = some child) :
    symbolRecordStep s ref node a = .inr child := by
  simp only [symbolRecordStep, hr, Bool.false_eq_true, ite_false, hs, ite_true, hop, hrow, Option.getD_some, hchild]

structure SymbolSnapshotValid (s : QuerySnapshot) : Prop where
  fields : s.fields.length = s.nodes.length
  operands : s.operands.length = s.nodes.length
  arity : ∀ row ∈ s.operands, row.arr.length = 2
  allocation : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46
  children : ∀ ref child, symbolStep s ref = .inr child → validSymbolRef s child

theorem symbolStep_cached (s : QuerySnapshot) (ref value : w32)
    (hz : ref ≠ W32 0) (hc : s.fields[(uint.Z ref - 1).toNat]? = some value) (hv : value ≠ W32 0) :
    symbolStep s ref = .inl value := by
  simp only [symbolStep, hz, ite_false, hc, Option.getD_some]
  rw [ite_eq_left hv]

theorem symbolStep_absent (s : QuerySnapshot) (ref : w32) (node : pdg.Node)
    (hz : ref ≠ W32 0) (hc : s.fields[(uint.Z ref - 1).toNat]? = some (W32 0))
    (hn : s.nodes[(uint.Z ref - 1).toNat]? = some node)
    (ha : ¬ (0 < uint.Z node.Attributes' ∧ uint.Z node.Attributes' ≤ s.attributes.length)) :
    symbolStep s ref = .inl (W32 0) := by
  simp only [symbolStep, hz, ite_false, hc, Option.getD_some, ne_eq, not_true_eq_false, hn]
  by_cases ha0 : node.Attributes' = W32 0
  · rw [ite_eq_left ha0]
  · rw [ite_eq_right ha0]
    have hpos : 0 < uint.Z node.Attributes' := by word
    have hnone : s.attributes[(uint.Z node.Attributes' - 1).toNat]? = none :=
      List.getElem?_eq_none (by omega)
    rw [hnone]

theorem symbolStep_present (s : QuerySnapshot) (ref : w32) (node : pdg.Node) (attrs : pdg.AttributeSet)
    (hz : ref ≠ W32 0) (hc : s.fields[(uint.Z ref - 1).toNat]? = some (W32 0))
    (hn : s.nodes[(uint.Z ref - 1).toNat]? = some node) (ha0 : node.Attributes' ≠ W32 0)
    (ha : s.attributes[(uint.Z node.Attributes' - 1).toNat]? = some attrs) :
    symbolStep s ref = symbolRecordStep s ref node attrs := by
  simp only [symbolStep, hz, ite_false, hc, Option.getD_some, ne_eq, not_true_eq_false, hn, ha0, ha]

theorem symbolStep_zero (s : QuerySnapshot) : symbolStep s (W32 0) = .inl (W32 0) := by
  simp only [symbolStep, ite_true]

theorem symbolResolve_zero (s : QuerySnapshot) (fuel : Nat) :
    symbolResolve s (fuel + 1) (W32 0) = some (W32 0) := by
  rw [symbolResolve, symbolStep_zero]

theorem symbolResolve_terminal (s : QuerySnapshot) (fuel : Nat) (ref value : w32)
    (hstep : symbolStep s ref = .inl value) :
    symbolResolve s (fuel + 1) ref = some value := by
  rw [symbolResolve, hstep]

theorem symbolResolve_child (s : QuerySnapshot) (fuel : Nat) (ref child : w32)
    (hstep : symbolStep s ref = .inr child) :
    symbolResolve s (fuel + 1) ref = symbolResolve s fuel child := by
  rw [symbolResolve, hstep]

theorem symbolResolve_add (s : QuerySnapshot) (fuel extra : Nat) (ref value : w32)
    (hresult : symbolResolve s fuel ref = some value) :
    symbolResolve s (fuel + extra) ref = some value := by
  induction fuel generalizing ref with
  | zero => cases hresult
  | succ fuel ih =>
    cases hstep : symbolStep s ref with
    | inl terminal =>
      simpa only [Nat.succ_add, symbolResolve, hstep] using hresult
    | inr child =>
      rw [symbolResolve, hstep] at hresult
      rw [Nat.succ_add, symbolResolve, hstep]
      try exact ih child hresult

theorem symbolResolve_deterministic (s : QuerySnapshot) (first second : Nat) (ref a b : w32)
    (ha : symbolResolve s first ref = some a) (hb : symbolResolve s second ref = some b) : a = b := by
  have hfirst := symbolResolve_add s first second ref a ha
  have hsecond := symbolResolve_add s second first ref b hb
  rw [Nat.add_comm second first, hfirst] at hsecond
  exact Option.some.inj hsecond

theorem symbolResolve_rank (s : QuerySnapshot) (rank : w32 → Nat)
    (hdecrease : ∀ ref child, symbolStep s ref = .inr child → rank child < rank ref)
    (ref : w32) : ∃ value, symbolResolve s (rank ref + 1) ref = some value := by
  have solve : ∀ n ref, rank ref = n → ∃ value, symbolResolve s (n + 1) ref = some value := by
    intro n
    induction n using Nat.strongRecOn with
    | ind n ih =>
      intro current hrank
      cases hstep : symbolStep s current with
      | inl value => exact ⟨value, symbolResolve_terminal s n current value hstep⟩
      | inr child =>
        have hlt : rank child < n := by have := hdecrease current child hstep; omega
        obtain ⟨value, hvalue⟩ := ih (rank child) hlt child rfl
        have hpad := symbolResolve_add s (rank child + 1) (n - (rank child + 1)) child value hvalue
        have hn : rank child + 1 + (n - (rank child + 1)) = n := by omega
        rw [hn] at hpad
        refine ⟨value, ?_⟩
        rw [symbolResolve, hstep]
        try exact hpad
  exact solve (rank ref) ref rfl

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
