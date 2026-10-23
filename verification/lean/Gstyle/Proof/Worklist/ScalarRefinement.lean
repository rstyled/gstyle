module
public import Gstyle.Proof.Worklist.ScalarBlock
public import Gstyle.Proof.Worklist.Decision
public import Gstyle.Proof.Worklist.StepModel
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [ext : FfiSyntax]

/-- Rejected edges have no state; every retained scalar edge has no pointer identity. -/
def scalarOption : Option StateData → Prop
  | none => True
  | some m => scalarState m

theorem scalarState_empty : scalarState (∅ : StateData) := by
  intro key value h
  simp at h

theorem scalarState_update (m : StateData) (key : w32) (v : Value)
    (hm : scalarState m) (hv : v.Identity' = W64 0) : scalarState (<[key := v]>m) := by
  intro other value h
  by_cases he : key = other
  · subst other
    simp only [GMap.lookup_insert] at h
    cases h
    exact hv
  · simp only [GMap.lookup_insert_eq_iff, he, Ne.symm he] at h
    exact hm other value h

theorem baseKind_identity (kind : GoString) : (baseKind kind).Identity' = W64 0 := by
  unfold baseKind
  split <;> rfl

theorem bindingValue_scalar (kind : GoString) (changing : Bool) (key : w32) (m : StateData)
    (hm : scalarState m) (hn : kind ≠ go!"\"pointer\"") :
    (bindingValue kind changing key m).Identity' = W64 0 := by
  cases changing
  · simp only [bindingValue, Bool.false_eq_true, ite_false]
    cases h : m !! key with
    | none => simpa only [h, Option.getD_none, defaultKind, hn, ite_false] using baseKind_identity kind
    | some v => exact hm key v h
  · exact baseKind_identity kind

theorem numericConstrained_identity (v bound : Value) (op : GoString) :
    (numericConstrained v op bound).Identity' = v.Identity' := by
  unfold numericConstrained
  split <;> unfold lengthConstrained <;> split <;> rfl

theorem constraintState_scalar (active : Bool) (kind : GoString) (changing : Bool)
    (key : w32) (m : StateData) (op : GoString) (bound : Value)
    (hm : scalarState m) (hn : kind ≠ go!"\"pointer\"") :
    scalarState (constraintState active kind changing key m op bound) := by
  unfold constraintState
  split
  · apply scalarState_update m key _ hm
    exact (numericConstrained_identity _ _ _).trans (bindingValue_scalar kind changing key m hm hn)
  · exact hm

theorem merged_scalar (left right : Value) (hl : left.Identity' = W64 0) :
    (merged left right).Identity' = W64 0 := by
  simp only [merged, mergedPointers, mergedBounds, hl]
  split <;> rfl

theorem joinedState_scalar (kind : w32 → GoString) (changing : w32 → Bool)
    (p n : StateData) (hp : scalarState p) (hn : scalarState n) :
    scalarState (joinedState kind changing p n) := by
  intro key value hv
  change joinLookup kind changing p n key = some value at hv
  cases hl : p !! key with
  | none =>
    cases hr : n !! key with
    | none => simp [joinLookup, hl, hr] at hv
    | some v =>
      simp only [joinLookup, hl, hr, Option.map_some, Option.some.injEq] at hv
      subst value
      exact merged_scalar v _ (hn key v hr)
  | some v =>
    simp only [joinLookup, hl, Option.some.injEq] at hv
    subst value
    exact merged_scalar v _ (hp key v hl)

theorem joinedInput_scalar (kind : w32 → GoString) (changing : w32 → Bool)
    (p : Option StateData) (n : StateData) (hp : scalarOption p) (hn : scalarState n) :
    scalarState (joinedInput kind changing p n) := by
  cases p with
  | none => exact hn
  | some p => exact joinedState_scalar kind changing p n hp hn

theorem comparisonResultMap_scalar (c : NumericComparison) (m : StateData)
    (hm : scalarState m) (hl : c.leftKind ≠ go!"\"pointer\"")
    (hr : c.rightKind ≠ go!"\"pointer\"") : scalarState (comparisonResultMap c m) := by
  have hleft := constraintState_scalar c.leftActive c.leftKind c.leftChanging c.leftKey m
    c.operator c.rightValue hm hl
  have hright := constraintState_scalar c.rightActive c.rightKind c.rightChanging c.rightKey
    (comparisonLeftMap c m) c.reversed c.leftValue hleft hr
  unfold comparisonResultMap
  split <;> assumption

/-- Every numeric execution constructor preserves scalar storage. The global
    kind restriction excludes pointer defaults; it does not assume a result. -/
theorem numericExecution_scalar (s : QuerySnapshot) (o : AssumeOperators)
    (kindOf : w32 → GoString) (changingOf : w32 → Bool) (m : StateData)
    (ref : w32) (op : GoString) (truth : Bool) (result : Option StateData)
    (hm : scalarState m) (hnonpointer : ∀ key, kindOf key ≠ go!"\"pointer\"")
    (execution : NumericExecution s o kindOf changingOf m ref op truth result) :
    scalarOption result := by
  cases execution with
  | leftTrue left right key row kind op lv rv changing active sf lf rf wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hkey metadata ht hnil hkind hchanging =>
    have hv := constraintState_scalar active kind changing key m op rv hm (by rw [hkind]; exact hnonpointer key)
    cases rejected : active && constraintRejected kind changing key m op rv <;> simp [rejected, scalarOption, hv]
  | leftFalse left right key row kind op lv rv changing active sf lf rf normal wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hkey metadata ht hnil hlookup hkind hchanging =>
    have hv := constraintState_scalar active kind changing key m normal rv hm (by rw [hkind]; exact hnonpointer key)
    cases rejected : active && constraintRejected kind changing key m normal rv <;> simp [rejected, scalarOption, hv]
  | rightTrue left right key row kind op lv rv changing active sf lf rf normal wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hkey metadata ht hnil hlookup hkind hchanging =>
    have hv := constraintState_scalar active kind changing key m normal lv hm (by rw [hkind]; exact hnonpointer key)
    cases rejected : active && constraintRejected kind changing key m normal lv <;> simp [rejected, scalarOption, hv]
  | rightFalse left right key row kind op lv rv changing active sf lf rf normal negated wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hkey metadata ht hnil hlookup hnegate hkind hchanging =>
    have hv := constraintState_scalar active kind changing key m normal lv hm (by rw [hkind]; exact hnonpointer key)
    cases rejected : active && constraintRejected kind changing key m normal lv <;> simp [rejected, scalarOption, hv]
  | bothTrue left right row c sf lf rf wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hleftNonzero hrightNonzero hleftMetadata hrightMetadata ht hnill hnilr hreverse hkindl hkindr hchangingl hchangingr =>
    have hv := comparisonResultMap_scalar c m hm (by rw [hkindl]; exact hnonpointer c.leftKey) (by rw [hkindr]; exact hnonpointer c.rightKey)
    cases rejected : comparisonRejected c m <;> simp [rejected, scalarOption, hv]
  | bothFalse left right row c sf lf rf rawOperator wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hleftNonzero hrightNonzero hleftMetadata hrightMetadata ht hnill hnilr hreverse hnegate hkindl hkindr hchangingl hchangingr =>
    have hv := comparisonResultMap_scalar c m hm (by rw [hkindl]; exact hnonpointer c.leftKey) (by rw [hkindr]; exact hnonpointer c.rightKey)
    cases rejected : comparisonRejected c m <;> simp [rejected, scalarOption, hv]

  | neither => exact hm

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
variable [FuncUnfold exactInteger [] exactInteger.impl]
variable [FuncUnfold integerConstraint [] integerConstraint.impl]
variable [FuncUnfold lengthConstraint [] lengthConstraint.impl]
variable [FuncUnfold emptyRange [] emptyRange.impl]
variable [interval.Assumptions]
variable [FuncUnfold compareExactOrder [] compareExactOrder.impl]
variable [FuncUnfold compareExactIntegers [] compareExactIntegers.impl]
variable [FuncUnfold booleanContradiction [] booleanContradiction.impl]
variable [FuncUnfold copyState [] copyState.impl]

/-- Induction follows finite production-assume execution, including both logical
    branches. No child scalar or semantic contract is assumed. -/
theorem finiteAssume_scalar (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool) (fuel : Nat) (ref : w32) (truth : Bool)
    (step : StateData → Option StateData)
    (execution : FiniteAssume s o kind changing fuel ref truth step)
    (hnonpointer : ∀ key, kind key ≠ go!"\"pointer\"")
    (m : StateData) (hm : validBranchState s.kinds s.unstable kind changing m)
    (hscalar : scalarState m) : scalarOption (step m) := by
  induction fuel generalizing ref truth step m with
  | zero => cases execution
  | succ fuel ih =>
    cases execution
    case conflict => trivial
    case parentheses =>
      rename_i child row hc hvalid hop href hrow hguard childExecution
      exact ih _ _ _ childExecution _ hm hscalar
    case negation =>
      rename_i child row hc hvalid hop href hrow hguard childExecution
      exact ih _ _ _ childExecution _ hm hscalar
    case logical =>
      rename_i leftRef rightRef op left right hlogical row hl hr hvalid hop href hrow leftExecution rightExecution hguard
      have hsleft := ih _ _ _ leftExecution m hm hscalar
      have hsright := ih _ _ _ rightExecution m hm hscalar
      unfold logicalStep
      split
      · cases hlm : left m with
        | none => trivial
        | some n =>
          have hv := finiteAssume_valid s o kind changing fuel leftRef truth left leftExecution m hm
          have hn : validBranchState s.kinds s.unstable kind changing n := by
            simpa only [hlm, validBranchOption] using hv
          have hs : scalarState n := by simpa only [hlm, scalarOption] using hsleft
          simpa only [hlm, Option.bind_some] using ih _ _ _ rightExecution n hn hs
      · cases hlm : left m <;> cases hrm : right m <;>
          simp only [hlm, hrm, scalarOption] at hsleft hsright
        · trivial
        · exact hsright
        · exact hsleft
        · exact joinedState_scalar kind changing _ _ hsleft hsright
    case comparison =>
      rename_i op hparentheses hnot hand hor hvalid hop hguard leaf
      exact numericExecution_scalar s o kind changing m ref op truth (step m) hscalar hnonpointer (leaf m hm)

end proof

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
