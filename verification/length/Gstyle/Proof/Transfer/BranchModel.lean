module
public import Gstyle.Proof.StateMap.Join
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [FfiSyntax]

def sequentialBranch (op : GoString) (truth : Bool) : Bool := decide (decide (op = go!"&&") = truth)

def branchJoin (kind : w32 → GoString) (changing : w32 → Bool)
    (left right out : Option StateData) : Prop :=
  match left, right with
  | none, none => out = none
  | some m, none => out = some m
  | none, some m => out = some m
  | some p, some n => ∃ m, out = some m ∧ ∀ key, m !! key = joinLookup kind changing p n key

def branchResult (kind : w32 → GoString) (changing : w32 → Bool)
    (op : GoString) (truth : Bool) (left right : StateData → Option StateData)
    (m : StateData) (out : Option StateData) : Prop :=
  if sequentialBranch op truth then out = (left m).bind right
  else branchJoin kind changing (left m) (right m) out

def stateDenotes (kind : w32 → GoString) (changing : w32 → Bool)
    (m : StateData) (env : w32 → Int) : Prop :=
  ∀ key, interval.contains (bounds (bindingValue (kind key) (changing key) key m)) (env key)

def branchDenotes (kind : w32 → GoString) (changing : w32 → Bool)
    (m : Option StateData) (env : w32 → Int) : Prop :=
  match m with
  | none => False
  | some m => stateDenotes kind changing m env

theorem branch_join_sound (kind : w32 → GoString) (changing : w32 → Bool)
    (left right out : Option StateData) (env : w32 → Int)
    (hout : branchJoin kind changing left right out)
    (hin : branchDenotes kind changing left env ∨ branchDenotes kind changing right env) :
    branchDenotes kind changing out env := by
  cases left with
  | none =>
    cases right with
    | none => simp [branchDenotes] at hin
    | some n => simpa [branchJoin] using hout ▸ hin.resolve_left (by simp [branchDenotes])
  | some p =>
    cases right with
    | none => simpa [branchJoin] using hout ▸ hin.resolve_right (by simp [branchDenotes])
    | some n =>
      obtain ⟨m, rfl, hlookup⟩ := hout
      intro key
      apply state_join_sound kind changing p n m key (env key) hlookup
      cases hin with
      | inl hp => exact Or.inl (hp key)
      | inr hn => exact Or.inr (hn key)

theorem logical_truth_cases (op : GoString) (truth lhs rhs : Bool)
    (hop : op = go!"&&" ∨ op = go!"||")
    (htruth : (if op = go!"&&" then lhs && rhs else lhs || rhs) = truth) :
    if sequentialBranch op truth then lhs = truth ∧ rhs = truth else lhs = truth ∨ rhs = truth := by
  rcases hop with rfl | rfl
  all_goals cases truth <;> cases lhs <;> cases rhs <;> simp_all [sequentialBranch]

theorem branch_result_sound (kind : w32 → GoString) (changing : w32 → Bool)
    (op : GoString) (truth : Bool) (left right : StateData → Option StateData)
    (m : StateData) (out : Option StateData) (env : w32 → Int) (lhs rhs : Bool)
    (hop : op = go!"&&" ∨ op = go!"||")
    (htruth : (if op = go!"&&" then lhs && rhs else lhs || rhs) = truth)
    (hin : stateDenotes kind changing m env)
    (hleft : ∀ m, stateDenotes kind changing m env → lhs = truth → branchDenotes kind changing (left m) env)
    (hright : ∀ m, stateDenotes kind changing m env → rhs = truth → branchDenotes kind changing (right m) env)
    (hout : branchResult kind changing op truth left right m out) :
    branchDenotes kind changing out env := by
  have hc := logical_truth_cases op truth lhs rhs hop htruth
  cases hs : sequentialBranch op truth
  · simp [hs, branchResult] at hout
    simp [hs] at hc
    apply branch_join_sound kind changing (left m) (right m) out env hout
    rcases hc with hl | hr
    · exact Or.inl (hleft m hin hl)
    · exact Or.inr (hright m hin hr)
  · simp [hs, branchResult] at hout
    simp [hs] at hc
    subst out
    have hl := hleft m hin hc.1
    cases hm : left m with
    | none => simp [hm, branchDenotes] at hl
    | some n => simpa [hm] using hright n (by simpa [hm, branchDenotes] using hl) hc.2

def validBranchState (kinds : List GoString) (unstable : List Bool)
    (kind : w32 → GoString) (changing : w32 → Bool) (m : StateData) : Prop :=
  ∀ key, (m !! key).isSome →
    (0 < uint.Z key ∧ uint.Z key ≤ kinds.length) ∧
    kinds[(uint.Z key - 1).toNat]? = some (kind key) ∧
    unstable[(uint.Z key - 1).toNat]? = some (changing key)

def validBranchOption (kinds : List GoString) (unstable : List Bool)
    (kind : w32 → GoString) (changing : w32 → Bool) (m : Option StateData) : Prop :=
  match m with
  | none => True
  | some m => validBranchState kinds unstable kind changing m

theorem joined_branch_valid (kinds : List GoString) (unstable : List Bool)
    (kind : w32 → GoString) (changing : w32 → Bool) (p n m : StateData)
    (hp : validBranchState kinds unstable kind changing p)
    (hn : validBranchState kinds unstable kind changing n)
    (hout : ∀ key, m !! key = joinLookup kind changing p n key) :
    validBranchState kinds unstable kind changing m := by
  intro key hk
  have hdomain := joinLookup_domain kind changing p n key
  rw [← hout key] at hdomain
  rw [hdomain] at hk
  cases hpresent : (p !! key).isSome <;> cases hnpresent : (n !! key).isSome
  · simp [hpresent, hnpresent] at hk
  · exact hn key hnpresent
  · exact hp key hpresent
  · exact hp key hpresent

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
