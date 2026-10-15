module
public import Gstyle.Proof.Transfer.AssumeNumeric
public import Gstyle.Proof.Transfer.AssumeDispatch
public import Gstyle.Proof.Transfer.Boolean
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

variable [FuncUnfold integerConstraint [] integerConstraint.impl]
variable [FuncUnfold lengthConstraint [] lengthConstraint.impl]
variable [FuncUnfold exactInteger [] exactInteger.impl]
variable [FuncUnfold emptyRange [] emptyRange.impl]
variable [interval.Assumptions]



def joinedState (kind : w32 → GoString) (changing : w32 → Bool) (p n : StateData) : StateData :=
  ⟨joinLookup kind changing p n, by
    obtain ⟨pk, hp⟩ := p.finite
    obtain ⟨nk, hn⟩ := n.finite
    refine ⟨pk ++ nk, fun key hk => ?_⟩
    have hd := joinLookup_domain kind changing p n key
    rw [hd] at hk
    cases hpk : (p !! key).isSome
    · have hnk : (n !! key).isSome := by simpa [hpk] using hk
      exact List.mem_append_right _ (hn key hnk)
    · exact List.mem_append_left _ (hp key hpk)⟩

def joinOptions (kind : w32 → GoString) (changing : w32 → Bool) : Option StateData → Option StateData → Option StateData
  | none, right => right
  | left, none => left
  | some p, some n => some (joinedState kind changing p n)

def logicalStep (kind : w32 → GoString) (changing : w32 → Bool)
    (op : GoString) (truth : Bool) (left right : StateData → Option StateData) (m : StateData) : Option StateData :=
  if sequentialBranch op truth then (left m).bind right else joinOptions kind changing (left m) (right m)

theorem logicalStep_result (kind : w32 → GoString) (changing : w32 → Bool)
    (op : GoString) (truth : Bool) (left right : StateData → Option StateData) (m : StateData) :
    branchResult kind changing op truth left right m (logicalStep kind changing op truth left right m) := by
  unfold branchResult logicalStep
  split
  · rfl
  · cases left m <;> cases right m <;> simp [joinOptions, branchJoin]
    exact fun _ => rfl

theorem branchResult_unique (kind : w32 → GoString) (changing : w32 → Bool)
    (op : GoString) (truth : Bool) (left right : StateData → Option StateData) (m : StateData) (out : Option StateData)
    (h : branchResult kind changing op truth left right m out) :
    out = logicalStep kind changing op truth left right m := by
  unfold branchResult at h
  unfold logicalStep
  split at h <;> rename_i hs
  · simpa [hs] using h
  · simp only [hs, Bool.false_eq_true, ite_false]
    cases hl : left m <;> cases hr : right m <;> simp only [hl, hr, branchJoin] at h
    · exact h
    · exact h
    · exact h
    · obtain ⟨om, rfl, ho⟩ := h
      congr 1
      exact GMap.ext ho

inductive FiniteAssume (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool) :
    Nat → w32 → Bool → (StateData → Option StateData) → Prop where
  | conflict (fuel : Nat) (ref : w32) (truth : Bool)
      (hvalid : validSymbolRef s ref)
      (hconflict : booleanConflict (snapshotFact s ref (go!"boolean")) truth = true) :
      FiniteAssume s o kind changing (fuel + 1) ref truth (fun _ => none)
  | parentheses (fuel : Nat) (ref child : w32) (truth : Bool) (step : StateData → Option StateData)
      (hvalid : validSymbolRef s ref)
      (hguard : booleanConflict (snapshotFact s ref (go!"boolean")) truth = false)
      (hop : decodedFact s (snapshotRecord s ref).Operator' = go!"parentheses")
      (row : GoArray w32 2)
      (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
      (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row) (hc : row.arr[0]? = some child)
      (childExecution : FiniteAssume s o kind changing fuel child truth step) :
      FiniteAssume s o kind changing (fuel + 1) ref truth step
  | negation (fuel : Nat) (ref child : w32) (truth : Bool) (step : StateData → Option StateData)
      (hvalid : validSymbolRef s ref)
      (hguard : booleanConflict (snapshotFact s ref (go!"boolean")) truth = false)
      (hop : decodedFact s (snapshotRecord s ref).Operator' = go!"!")
      (row : GoArray w32 2)
      (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
      (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row) (hc : row.arr[0]? = some child)
      (childExecution : FiniteAssume s o kind changing fuel child (!truth) step) :
      FiniteAssume s o kind changing (fuel + 1) ref truth step
  | logical (fuel : Nat) (ref leftRef rightRef : w32) (truth : Bool) (op : GoString)
      (left right : StateData → Option StateData)
      (hvalid : validSymbolRef s ref)
      (hguard : booleanConflict (snapshotFact s ref (go!"boolean")) truth = false)
      (hop : decodedFact s (snapshotRecord s ref).Operator' = op)
      (hlogical : op = go!"&&" ∨ op = go!"||")
      (row : GoArray w32 2)
      (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
      (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
      (hl : row.arr[0]? = some leftRef) (hr : row.arr[1]? = some rightRef)
      (leftExecution : FiniteAssume s o kind changing fuel leftRef truth left)
      (rightExecution : FiniteAssume s o kind changing fuel rightRef truth right) :
      FiniteAssume s o kind changing (fuel + 1) ref truth (logicalStep kind changing op truth left right)
  | comparison (fuel : Nat) (ref : w32) (truth : Bool) (op : GoString) (step : StateData → Option StateData)
      (hvalid : validSymbolRef s ref)
      (hguard : booleanConflict (snapshotFact s ref (go!"boolean")) truth = false)
      (hop : decodedFact s (snapshotRecord s ref).Operator' = op)
      (hparentheses : op ≠ go!"parentheses") (hnot : op ≠ go!"!")
      (hand : op ≠ go!"&&") (hor : op ≠ go!"||")
      (leaf : ∀ m, validBranchState s.kinds s.unstable kind changing m → NumericExecution s o kind changing m ref op truth (step m)) :
      FiniteAssume s o kind changing (fuel + 1) ref truth step

theorem finiteAssume_positive (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool) (fuel : Nat) (ref : w32) (truth : Bool)
    (step : StateData → Option StateData) (h : FiniteAssume s o kind changing fuel ref truth step) : 0 < fuel := by
  cases h <;> omega

theorem finiteAssume_self_cycle (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool) (ref : w32)
    (hguard : ∀ truth, booleanConflict (snapshotFact s ref (go!"boolean")) truth = false)
    (hop : decodedFact s (snapshotRecord s ref).Operator' = go!"parentheses")
    (hchild : ∀ row child, s.operands[(uint.Z ref - 1).toNat]? = some row → row.arr[0]? = some child → child = ref)
    (fuel : Nat) (truth : Bool) (step : StateData → Option StateData) :
    ¬FiniteAssume s o kind changing fuel ref truth step := by
  induction fuel generalizing truth step with
  | zero => intro h; cases h
  | succ fuel ih =>
    intro h
    cases h
    case conflict => rename_i hv hc; rw [hguard] at hc; contradiction
    case parentheses =>
      rename_i child row hc hv ho hr hrow hg childExecution
      have heq := hchild row child hrow hc
      subst child
      exact ih _ _ childExecution
    case negation => rename_i child row hc hv ho hr hrow hg childExecution; rw [hop] at ho; cases ho
    case logical =>
      rename_i leftRef rightRef op left right hl row hleft hright hv ho hr hrow le re hg
      rcases hl with rfl | rfl <;> rw [hop] at ho <;> cases ho
    case comparison => rename_i op hp hn ha hor hv ho hg leaf; exact hp (ho.symm.trans hop)

theorem logicalStep_valid (s : QuerySnapshot) (kind : w32 → GoString) (changing : w32 → Bool)
    (op : GoString) (truth : Bool) (left right : StateData → Option StateData) (m : StateData)
    (hm : validBranchState s.kinds s.unstable kind changing m)
    (hl : ∀ m, validBranchState s.kinds s.unstable kind changing m → validBranchOption s.kinds s.unstable kind changing (left m))
    (hr : ∀ m, validBranchState s.kinds s.unstable kind changing m → validBranchOption s.kinds s.unstable kind changing (right m)) :
    validBranchOption s.kinds s.unstable kind changing (logicalStep kind changing op truth left right m) := by
  have hleft := hl m hm
  have hright := hr m hm
  unfold logicalStep
  split
  · cases h : left m with
    | none => simp [h, validBranchOption]
    | some n => simpa [h] using hr n (by simpa [h, validBranchOption] using hleft)
  · cases h1 : left m <;> cases h2 : right m <;> simp only [h1, h2, validBranchOption] at hleft hright
    · trivial
    · exact hright
    · exact hleft
    · exact joined_branch_valid s.kinds s.unstable kind changing _ _ _ hleft hright (fun _ => rfl)

theorem finiteAssume_valid (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool) (fuel : Nat) (ref : w32) (truth : Bool)
    (step : StateData → Option StateData)
    (execution : FiniteAssume s o kind changing fuel ref truth step)
    (m : StateData) (hm : validBranchState s.kinds s.unstable kind changing m) :
    validBranchOption s.kinds s.unstable kind changing (step m) := by
  induction fuel generalizing ref truth step m with
  | zero => cases execution
  | succ fuel ih =>
    cases execution
    case conflict => trivial
    case parentheses => rename_i child row hc hvalid hop href hrow hguard childExecution; exact ih _ _ _ childExecution _ hm
    case negation => rename_i child row hc hvalid hop href hrow hguard childExecution; exact ih _ _ _ childExecution _ hm
    case logical =>
      rename_i leftRef rightRef op left right hlogical row hl hr hvalid hop href hrow leftExecution rightExecution hguard
      exact logicalStep_valid s kind changing op truth left right m hm
        (fun m hm => ih _ _ _ leftExecution m hm) (fun m hm => ih _ _ _ rightExecution m hm)
    case comparison =>
      rename_i op hparentheses hnot hand hor hvalid hop hguard leaf
      exact numeric_execution_valid s o kind changing m ref op truth (step m) hm (leaf m hm)

structure AssumeMeaning (s : QuerySnapshot) (values : w32 → Int) (eval : w32 → Bool) : Prop where
  booleanFacts : ∀ ref,
    (snapshotFact s ref (go!"boolean") = go!"true" → eval ref = true) ∧
    (snapshotFact s ref (go!"boolean") = go!"false" → eval ref = false)
  parentheses : ∀ ref row child, s.operands[(uint.Z ref - 1).toNat]? = some row → row.arr[0]? = some child →
    decodedFact s (snapshotRecord s ref).Operator' = go!"parentheses" → eval ref = eval child
  negation : ∀ ref row child, s.operands[(uint.Z ref - 1).toNat]? = some row → row.arr[0]? = some child →
    decodedFact s (snapshotRecord s ref).Operator' = go!"!" → eval ref = !(eval child)
  logical : ∀ ref row left right op, s.operands[(uint.Z ref - 1).toNat]? = some row →
    row.arr[0]? = some left → row.arr[1]? = some right →
    decodedFact s (snapshotRecord s ref).Operator' = op → (op = go!"&&" ∨ op = go!"||") →
    eval ref = (if op = go!"&&" then eval left && eval right else eval left || eval right)
  comparison : ∀ ref op truth, decodedFact s (snapshotRecord s ref).Operator' = op → eval ref = truth →
    numericTruth s ref op truth values

theorem booleanConflict_sound (s : QuerySnapshot) (values : w32 → Int) (eval : w32 → Bool)
    (meaning : AssumeMeaning s values eval) (ref : w32) (truth : Bool)
    (h : booleanConflict (snapshotFact s ref (go!"boolean")) truth = true) : eval ref ≠ truth := by
  obtain h | h := of_decide_eq_true h
  · have ht := (meaning.booleanFacts ref).1 h.1
    rw [ht, h.2]; decide
  · have ht := (meaning.booleanFacts ref).2 h.1
    rw [ht, h.2]; decide

theorem finiteAssume_sound (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool) (fuel : Nat) (ref : w32) (truth : Bool)
    (step : StateData → Option StateData)
    (execution : FiniteAssume s o kind changing fuel ref truth step)
    (env values : w32 → Int) (eval : w32 → Bool)
    (meaning : AssumeMeaning s values eval)
    (hvalues : valueIntervalMeaning s kind changing env values)
    (hsymbols : symbolIntervalMeaning s env values)
    (hword : ∀ key, -9223372036854775808 ≤ env key ∧ env key ≤ 9223372036854775807)
    (hinverse : ∀ op, (o.inverseTable !! op).getD (zero_val GoString) = inverseOperator op)
    (hreverse : ∀ op, (o.reverseTable !! op).getD (zero_val GoString) = reverseOperator op)
    (m : StateData) (hm : validBranchState s.kinds s.unstable kind changing m)
    (hin : stateDenotes kind changing m env) (htruth : eval ref = truth) :
    branchDenotes kind changing (step m) env := by
  induction fuel generalizing ref truth step m with
  | zero => cases execution
  | succ fuel ih =>
    cases execution
    case conflict =>
      rename_i hvalid hconflict
      exact False.elim (booleanConflict_sound s values eval meaning ref truth hconflict htruth)
    case parentheses =>
      rename_i child row hc hvalid hop href hrow hguard childExecution
      have heq := meaning.parentheses ref row child hrow hc hop
      exact ih child truth step childExecution m hm hin (heq.symm.trans htruth)
    case negation =>
      rename_i child row hc hvalid hop href hrow hguard childExecution
      have heq := meaning.negation ref row child hrow hc hop
      have hchild : eval child = !truth := by simpa only [Bool.not_not] using congrArg Bool.not (heq.symm.trans htruth)
      exact ih child (!truth) step childExecution m hm hin hchild
    case logical =>
      rename_i leftRef rightRef op left right hlogical row hl hr hvalid hop href hrow leftExecution rightExecution hguard
      have heval := meaning.logical ref row leftRef rightRef op hrow hl hr hop hlogical
      have casesTruth := logical_truth_cases op truth (eval leftRef) (eval rightRef) hlogical (heval.symm.trans htruth)
      unfold logicalStep
      cases hs : sequentialBranch op truth
      · simp only [hs, Bool.false_eq_true, ite_false] at casesTruth ⊢
        have hj : branchJoin kind changing (left m) (right m) (joinOptions kind changing (left m) (right m)) := by
          cases left m <;> cases right m <;> simp [branchJoin, joinOptions]
          exact fun _ => rfl
        apply branch_join_sound kind changing (left m) (right m) _ env hj
        rcases casesTruth with ht | ht
        · exact Or.inl (ih leftRef truth left leftExecution m hm hin ht)
        · exact Or.inr (ih rightRef truth right rightExecution m hm hin ht)
      · simp only [hs, ite_true] at casesTruth ⊢
        have hleft := ih leftRef truth left leftExecution m hm hin casesTruth.1
        have hvalid := finiteAssume_valid s o kind changing fuel leftRef truth left leftExecution m hm
        cases hlm : left m with
        | none => simp [hlm, branchDenotes] at hleft
        | some n =>
          have hv : validBranchState s.kinds s.unstable kind changing n := by simpa [hlm, validBranchOption] using hvalid
          have hn : stateDenotes kind changing n env := by simpa [hlm, branchDenotes] using hleft
          simpa [hlm] using ih rightRef truth right rightExecution n hv hn casesTruth.2
    case comparison =>
      rename_i op hparentheses hnot hand hor hvalid hop hguard leaf
      exact numeric_execution_sound s o kind changing m ref op truth (step m) (leaf m hm)
        env values hin hvalues hsymbols hword (meaning.comparison ref op truth hop htruth) hinverse hreverse

variable [FuncUnfold booleanContradiction [] booleanContradiction.impl]
variable [FuncUnfold copyState [] copyState.impl]

theorem wp_finite_assume (receiver : Loc) (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool) (fuel : Nat) (ref : w32) (truth : Bool)
    (step : StateData → Option StateData)
    (execution : FiniteAssume s o kind changing fuel ref truth step)
    (wf : ValueSnapshotValid s) (hparse : parseInt64Contract (GF := GF))
    (state : Loc) (m : StateData) (hm : validBranchState s.kinds s.unstable kind changing m) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
      (querySnapshotOwn receiver s ∗ assumeOperatorsOwn o ∗ state ↦$ m) }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"assume" #receiver)) (Val #ref)) (Val #truth)) (Val #state))
    {{ (out : Loc), RET #out; querySnapshotOwn receiver s ∗ assumeOperatorsOwn o ∗ ownsBranch (step m) out ∗
      ⌜validBranchOption s.kinds s.unstable kind changing (step m)⌝ }} := by
  induction fuel generalizing ref truth step state m with
  | zero => cases execution
  | succ fuel ih =>
    cases execution
    case conflict =>
      rename_i hvalid hconflict
      wp_start as ⟨#Hsnapshot, #Hoperators, Hstate⟩
      try wp_auto
      wp_apply wp_fact_snapshot receiver s ref (go!"boolean") hvalid wf.symbols.allocation $$ [Hsnapshot] with Hsnapshot
      · iframe Hsnapshot
      try wp_auto
      wp_apply wp_booleanContradiction (snapshotFact s ref (go!"boolean")) truth
      try wp_auto
      rw [hconflict]
      try wp_auto
      wp_end
      all_goals iframe Hsnapshot Hoperators
      all_goals simp only [ownsBranch]
      all_goals ipureintro; simp [validBranchOption]
    case parentheses =>
      rename_i child row hc hvalid hop href hrow hguard childExecution
      exact wp_assume_parentheses receiver s ref child truth state
        (iprop(assumeOperatorsOwn o ∗ state ↦$ m))
        (fun out => iprop(querySnapshotOwn receiver s ∗ assumeOperatorsOwn o ∗ ownsBranch (step m) out ∗
          ⌜validBranchOption s.kinds s.unstable kind changing (step m)⌝))
        hvalid wf.symbols.allocation hguard hop row href hrow hc
        (ih child truth step childExecution state m hm)
    case negation =>
      rename_i child row hc hvalid hop href hrow hguard childExecution
      exact wp_assume_negation receiver s ref child truth state
        (iprop(assumeOperatorsOwn o ∗ state ↦$ m))
        (fun out => iprop(querySnapshotOwn receiver s ∗ assumeOperatorsOwn o ∗ ownsBranch (step m) out ∗
          ⌜validBranchOption s.kinds s.unstable kind changing (step m)⌝))
        hvalid wf.symbols.allocation hguard hop row href hrow hc
        (ih child (!truth) step childExecution state m hm)
    case logical =>
      rename_i leftRef rightRef op left right hlogical row hl hr hvalid hop href hrow leftExecution rightExecution hguard
      have leftContract : branchChildContract (GF := GF) receiver s.view s.operands s.kinds s.unstable kind changing
          (iprop(querySnapshotOwn receiver s ∗ assumeOperatorsOwn o)) leftRef truth left DFrac.discard := by
        intro childState cm hcm
        wp_start_folded as ⟨Hresources, Hstate⟩
        iunfold branchResources at Hresources
        icases Hresources with ⟨#Hview, #Hrows, #Hkinds, #Hunstable, #Hsnapshot, #Hoperators⟩
        wp_apply ih leftRef truth left leftExecution childState cm hcm $$ [Hstate] with %out ⟨#Hsnapshot, #Hoperators, Hresult, %Hvalid⟩
        · iframe Hsnapshot Hoperators Hstate
        wp_end
        unfold branchResources
        iframe Hview Hrows Hkinds Hunstable Hsnapshot Hoperators Hresult
        ipureintro; exact Hvalid
      have rightContract : branchChildContract (GF := GF) receiver s.view s.operands s.kinds s.unstable kind changing
          (iprop(querySnapshotOwn receiver s ∗ assumeOperatorsOwn o)) rightRef truth right DFrac.discard := by
        intro childState cm hcm
        wp_start_folded as ⟨Hresources, Hstate⟩
        iunfold branchResources at Hresources
        icases Hresources with ⟨#Hview, #Hrows, #Hkinds, #Hunstable, #Hsnapshot, #Hoperators⟩
        wp_apply ih rightRef truth right rightExecution childState cm hcm $$ [Hstate] with %out ⟨#Hsnapshot, #Hoperators, Hresult, %Hvalid⟩
        · iframe Hsnapshot Hoperators Hstate
        wp_end
        unfold branchResources
        iframe Hview Hrows Hkinds Hunstable Hsnapshot Hoperators Hresult
        ipureintro; exact Hvalid
      apply wp_assume_logical receiver s ref truth state op
        (iprop(assumeOperatorsOwn o ∗ state ↦$ m))
        (fun out => iprop(querySnapshotOwn receiver s ∗ assumeOperatorsOwn o ∗ ownsBranch (logicalStep kind changing op truth left right m) out ∗
          ⌜validBranchOption s.kinds s.unstable kind changing (logicalStep kind changing op truth left right m)⌝))
        hvalid wf.symbols.allocation hguard hop hlogical
      wp_start_folded as ⟨#Hsnapshot, #Hoperators, Hstate⟩
      have Hlogical := wp_logical (GF := GF) receiver s.view s.operands s.kinds s.unstable kind changing
          (iprop(querySnapshotOwn receiver s ∗ assumeOperatorsOwn o)) state m ref op truth row leftRef rightRef left right
          href hrow hl hr wf.unstable hm leftContract rightContract
      have Hunique := branchResult_unique kind changing op truth left right m
      ihave #Hctx := Hsnapshot
      iunfold querySnapshotOwn at Hctx
      icases Hctx with ⟨Hview, _, _, Hrows, Hkinds, Hunstable, _, _, _, _, _, _, _, _, _, _, _⟩
      wp_apply Hlogical $$ [Hstate]
        with %out ⟨%result, Hresources, Hresult, Hsource, %Hspec⟩
      · unfold branchResources; iframe Hview Hrows Hkinds Hunstable Hsnapshot Hoperators Hstate
      have heq := Hunique result Hspec.1
      subst result
      wp_end
      iunfold branchResources at Hresources
      icases Hresources with ⟨_, _, _, _, Hsnapshot, Hoperators⟩
      iframe Hsnapshot Hoperators Hresult
      ipureintro; exact Hspec.2
    case comparison =>
      rename_i op hparentheses hnot hand hor hvalid hop hguard leaf
      exact wp_assume_comparison receiver s ref truth state op
        (iprop(assumeOperatorsOwn o ∗ state ↦$ m))
        (fun out => iprop(querySnapshotOwn receiver s ∗ assumeOperatorsOwn o ∗ ownsBranch (step m) out ∗
          ⌜validBranchOption s.kinds s.unstable kind changing (step m)⌝))
        hvalid wf.symbols.allocation hguard hop hparentheses hnot hand hor
        (wp_numeric_execution receiver s o kind changing state m ref op truth (step m) hm (leaf m hm) hparse)

theorem wp_finite_assume_sound (receiver : Loc) (s : QuerySnapshot) (o : AssumeOperators)
    (kind : w32 → GoString) (changing : w32 → Bool) (fuel : Nat) (ref : w32) (truth : Bool)
    (step : StateData → Option StateData)
    (execution : FiniteAssume s o kind changing fuel ref truth step)
    (wf : ValueSnapshotValid s) (hparse : parseInt64Contract (GF := GF))
    (state : Loc) (m : StateData) (hm : validBranchState s.kinds s.unstable kind changing m)
    (env values : w32 → Int) (eval : w32 → Bool)
    (meaning : AssumeMeaning s values eval)
    (hvalues : valueIntervalMeaning s kind changing env values)
    (hsymbols : symbolIntervalMeaning s env values)
    (hword : ∀ key, -9223372036854775808 ≤ env key ∧ env key ≤ 9223372036854775807)
    (hinverse : ∀ op, (o.inverseTable !! op).getD (zero_val GoString) = inverseOperator op)
    (hreverse : ∀ op, (o.reverseTable !! op).getD (zero_val GoString) = reverseOperator op)
    (hin : stateDenotes kind changing m env) (htruth : eval ref = truth) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
      (querySnapshotOwn receiver s ∗ assumeOperatorsOwn o ∗ state ↦$ m) }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"assume" #receiver)) (Val #ref)) (Val #truth)) (Val #state))
    {{ (out : Loc), RET #out; querySnapshotOwn receiver s ∗ assumeOperatorsOwn o ∗ ownsBranch (step m) out ∗
      ⌜validBranchOption s.kinds s.unstable kind changing (step m) ∧ branchDenotes kind changing (step m) env⌝ }} := by
  have sound := finiteAssume_sound s o kind changing fuel ref truth step execution env values eval meaning
    hvalues hsymbols hword hinverse hreverse m hm hin htruth
  wp_start_folded as Hinputs
  wp_apply wp_finite_assume receiver s o kind changing fuel ref truth step execution wf hparse state m hm
    $$ [Hinputs] with %out ⟨Hsnapshot, Hoperators, Hresult, %Hvalid⟩
  · iexact Hinputs
  wp_end

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
