module
public import Gstyle.Proof.Transfer.ComparisonOneSidedSound
public import Gstyle.Proof.Transfer.BranchSnapshot
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


structure AssumeOperators where
  inverseLoc : Loc
  inverseTable : GMap GoString GoString
  reverseLoc : Loc
  reverseTable : GMap GoString GoString

def assumeOperatorsOwn (o : AssumeOperators) : IProp GF := iprop(
  globalAddr inverse ↦□ o.inverseLoc ∗ o.inverseLoc ↦${DFrac.discard} o.inverseTable ∗
  globalAddr reverse ↦□ o.reverseLoc ∗ o.reverseLoc ↦${DFrac.discard} o.reverseTable)

instance assumeOperatorsOwn_persistent (o : AssumeOperators) : Persistent (assumeOperatorsOwn (GF := GF) o) := by
  unfold assumeOperatorsOwn
  infer_instance

theorem constraint_state_valid (s : QuerySnapshot) (kind : w32 → GoString) (changing : w32 → Bool)
    (m : StateData) (key : w32) (active : Bool) (k op : GoString) (u : Bool) (b : Value)
    (hm : validBranchState s.kinds s.unstable kind changing m)
    (metadata : ConstraintMetadata s key k u active)
    (hk : k = kind key) (hu : u = changing key) :
    validBranchState s.kinds s.unstable kind changing (constraintState active k u key m op b) := by
  unfold constraintState
  split
  · rename_i ha
    have hactive : active = true := by cases active <;> simp_all
    intro other ho
    by_cases he : key = other
    · subst other
      cases metadata with
      | present hr hkind hunstable _ => simpa only [hk, hu] using And.intro hr (And.intro hkind hunstable)
      | absent _ hf => simp [hf] at hactive
    · apply hm other
      simpa [GMap.lookup_insert_eq_iff, he, Ne.symm he] using ho
  · exact hm

theorem comparison_result_valid (s : QuerySnapshot) (kind : w32 → GoString) (changing : w32 → Bool)
    (m : StateData) (c : NumericComparison)
    (hm : validBranchState s.kinds s.unstable kind changing m)
    (hl : ConstraintMetadata s c.leftKey c.leftKind c.leftChanging c.leftActive)
    (hr : ConstraintMetadata s c.rightKey c.rightKind c.rightChanging c.rightActive)
    (hkl : c.leftKind = kind c.leftKey) (hkr : c.rightKind = kind c.rightKey)
    (hcl : c.leftChanging = changing c.leftKey) (hcr : c.rightChanging = changing c.rightKey) :
    validBranchState s.kinds s.unstable kind changing (comparisonResultMap c m) := by
  have left := constraint_state_valid s kind changing m c.leftKey c.leftActive c.leftKind c.operator c.leftChanging c.rightValue hm hl hkl hcl
  have right := constraint_state_valid s kind changing (comparisonLeftMap c m) c.rightKey c.rightActive c.rightKind c.reversed c.rightChanging c.leftValue left hr hkr hcr
  unfold comparisonResultMap
  split <;> assumption

inductive NumericExecution (s : QuerySnapshot) (o : AssumeOperators)
    (kindOf : w32 → GoString) (changingOf : w32 → Bool) :
    StateData → w32 → GoString → Bool → Option StateData → Prop where
  | leftTrue (m : StateData)
    (ref left right key : w32) (row : GoArray w32 2) (kind op : GoString) (lv rv : Value) (changing active : Bool)
    (sf lf rf : Nat) (wf : ValueSnapshotValid s)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some left) (hright : row.arr[1]? = some right)
    (hlvalid : validSymbolRef s left) (hrvalid : validSymbolRef s right)
    (hlvalue : valueResolve s m sf lf left = some lv)
    (hrvalue : valueResolve s m sf rf right = some rv)
    (hlsymbol : symbolResolve s sf left = some key)
    (hrsymbol : symbolResolve s sf right = some (W32 0))
    (hkey : key ≠ W32 0)
    (metadata : ConstraintMetadata s key kind changing active)
    (ht : s.tracked.length = s.kinds.length)
    (hnil : rv.Nil' = false)
    (hkind : kind = kindOf key) (hchanging : changing = changingOf key) : NumericExecution s o kindOf changingOf m ref op true (if active && constraintRejected kind changing key m op rv then none else some (constraintState active kind changing key m op rv))
  | leftFalse (m : StateData)
    (ref left right key : w32) (row : GoArray w32 2) (kind op : GoString) (lv rv : Value) (changing active : Bool)
    (sf lf rf : Nat) (normal : GoString) (wf : ValueSnapshotValid s)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some left) (hright : row.arr[1]? = some right)
    (hlvalid : validSymbolRef s left) (hrvalid : validSymbolRef s right)
    (hlvalue : valueResolve s m sf lf left = some lv)
    (hrvalue : valueResolve s m sf rf right = some rv)
    (hlsymbol : symbolResolve s sf left = some key)
    (hrsymbol : symbolResolve s sf right = some (W32 0))
    (hkey : key ≠ W32 0)
    (metadata : ConstraintMetadata s key kind changing active)
    (ht : s.tracked.length = s.kinds.length)
    (hnil : rv.Nil' = false)
    (hlookup : (o.inverseTable !! op).getD (zero_val GoString) = normal)
    (hkind : kind = kindOf key) (hchanging : changing = changingOf key) : NumericExecution s o kindOf changingOf m ref op false (if active && constraintRejected kind changing key m normal rv then none else some (constraintState active kind changing key m normal rv))
  | rightTrue (m : StateData)
    (ref left right key : w32) (row : GoArray w32 2) (kind op : GoString) (lv rv : Value) (changing active : Bool)
    (sf lf rf : Nat) (normal : GoString) (wf : ValueSnapshotValid s)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some left) (hright : row.arr[1]? = some right)
    (hlvalid : validSymbolRef s left) (hrvalid : validSymbolRef s right)
    (hlvalue : valueResolve s m sf lf left = some lv)
    (hrvalue : valueResolve s m sf rf right = some rv)
    (hlsymbol : symbolResolve s sf left = some (W32 0))
    (hrsymbol : symbolResolve s sf right = some key)
    (hkey : key ≠ W32 0)
    (metadata : ConstraintMetadata s key kind changing active)
    (ht : s.tracked.length = s.kinds.length)
    (hnil : lv.Nil' = false)
    (hlookup : (o.reverseTable !! op).getD (zero_val GoString) = normal)
    (hkind : kind = kindOf key) (hchanging : changing = changingOf key) : NumericExecution s o kindOf changingOf m ref op true (if active && constraintRejected kind changing key m normal lv then none else some (constraintState active kind changing key m normal lv))
  | rightFalse (m : StateData)
    (ref left right key : w32) (row : GoArray w32 2) (kind op : GoString) (lv rv : Value) (changing active : Bool)
    (sf lf rf : Nat) (normal negated : GoString) (wf : ValueSnapshotValid s)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some left) (hright : row.arr[1]? = some right)
    (hlvalid : validSymbolRef s left) (hrvalid : validSymbolRef s right)
    (hlvalue : valueResolve s m sf lf left = some lv)
    (hrvalue : valueResolve s m sf rf right = some rv)
    (hlsymbol : symbolResolve s sf left = some (W32 0))
    (hrsymbol : symbolResolve s sf right = some key)
    (hkey : key ≠ W32 0)
    (metadata : ConstraintMetadata s key kind changing active)
    (ht : s.tracked.length = s.kinds.length)
    (hnil : lv.Nil' = false)
    (hlookup : (o.reverseTable !! negated).getD (zero_val GoString) = normal)
    (hnegate : (o.inverseTable !! op).getD (zero_val GoString) = negated)
    (hkind : kind = kindOf key) (hchanging : changing = changingOf key) : NumericExecution s o kindOf changingOf m ref op false (if active && constraintRejected kind changing key m normal lv then none else some (constraintState active kind changing key m normal lv))
  | bothTrue (m : StateData)
    (ref left right : w32) (row : GoArray w32 2) (c : NumericComparison)
    (sf lf rf : Nat) (wf : ValueSnapshotValid s)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some left) (hright : row.arr[1]? = some right)
    (hlvalid : validSymbolRef s left) (hrvalid : validSymbolRef s right)
    (hlvalue : valueResolve s m sf lf left = some c.leftValue)
    (hrvalue : valueResolve s m sf rf right = some c.rightValue)
    (hlsymbol : symbolResolve s sf left = some c.leftKey)
    (hrsymbol : symbolResolve s sf right = some c.rightKey)
    (hleftNonzero : c.leftKey ≠ W32 0) (hrightNonzero : c.rightKey ≠ W32 0)
    (hleftMetadata : ConstraintMetadata s c.leftKey c.leftKind c.leftChanging c.leftActive)
    (hrightMetadata : ConstraintMetadata s c.rightKey c.rightKind c.rightChanging c.rightActive)
    (ht : s.tracked.length = s.kinds.length)
    (hnill : c.leftValue.Nil' = false) (hnilr : c.rightValue.Nil' = false)
    (hreverse : (o.reverseTable !! c.operator).getD (zero_val GoString) = c.reversed)
    (hkindl : c.leftKind = kindOf c.leftKey) (hkindr : c.rightKind = kindOf c.rightKey)
    (hchangingl : c.leftChanging = changingOf c.leftKey) (hchangingr : c.rightChanging = changingOf c.rightKey) : NumericExecution s o kindOf changingOf m ref c.operator true (if comparisonRejected c m then none else some (comparisonResultMap c m))
  | bothFalse (m : StateData)
    (ref left right : w32) (row : GoArray w32 2) (c : NumericComparison)
    (sf lf rf : Nat) (rawOperator : GoString) (wf : ValueSnapshotValid s)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hleft : row.arr[0]? = some left) (hright : row.arr[1]? = some right)
    (hlvalid : validSymbolRef s left) (hrvalid : validSymbolRef s right)
    (hlvalue : valueResolve s m sf lf left = some c.leftValue)
    (hrvalue : valueResolve s m sf rf right = some c.rightValue)
    (hlsymbol : symbolResolve s sf left = some c.leftKey)
    (hrsymbol : symbolResolve s sf right = some c.rightKey)
    (hleftNonzero : c.leftKey ≠ W32 0) (hrightNonzero : c.rightKey ≠ W32 0)
    (hleftMetadata : ConstraintMetadata s c.leftKey c.leftKind c.leftChanging c.leftActive)
    (hrightMetadata : ConstraintMetadata s c.rightKey c.rightKind c.rightChanging c.rightActive)
    (ht : s.tracked.length = s.kinds.length)
    (hnill : c.leftValue.Nil' = false) (hnilr : c.rightValue.Nil' = false)
    (hreverse : (o.reverseTable !! c.operator).getD (zero_val GoString) = c.reversed)
    (hnegate : (o.inverseTable !! rawOperator).getD (zero_val GoString) = c.operator)
    (hkindl : c.leftKind = kindOf c.leftKey) (hkindr : c.rightKind = kindOf c.rightKey)
    (hchangingl : c.leftChanging = changingOf c.leftKey) (hchangingr : c.rightChanging = changingOf c.rightKey) : NumericExecution s o kindOf changingOf m ref rawOperator false (if comparisonRejected c m then none else some (comparisonResultMap c m))

theorem wp_numeric_execution (receiver : Loc) (s : QuerySnapshot) (o : AssumeOperators)
    (kindOf : w32 → GoString) (changingOf : w32 → Bool) (state : Loc) (m : StateData)
    (ref : w32) (op : GoString) (truth : Bool) (result : Option StateData)
    (hm : validBranchState s.kinds s.unstable kindOf changingOf m)
    (execution : NumericExecution s o kindOf changingOf m ref op truth result)
    (hparse : parseInt64Contract (GF := GF)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
      (querySnapshotOwn receiver s ∗ assumeOperatorsOwn o ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"comparison" #receiver)) (Val #ref)) (Val #op)) (Val #truth)) (Val #state))
    {{ (out : Loc), RET #out; querySnapshotOwn receiver s ∗ assumeOperatorsOwn o ∗ ownsBranch result out ∗
      ⌜validBranchOption s.kinds s.unstable kindOf changingOf result⌝ }} := by
  cases execution with
  | leftTrue left right key row kind op lv rv changing active sf lf rf wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hkey metadata ht hnil hkind hchanging =>
    have hv := constraint_state_valid s kindOf changingOf m key active kind op changing rv hm metadata hkind hchanging
    wp_start_folded as ⟨#Hsnapshot, #Hoperators, Hstate⟩
    ihave #Hctx := Hoperators
    iunfold assumeOperatorsOwn at Hctx
    icases Hctx with ⟨HinvGlobal, Hinv, HrevGlobal, Hrev⟩
    wp_apply wp_comparison_left_numeric_true receiver s state m ref left right key row kind op lv rv changing active sf lf rf wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hkey metadata ht hnil hparse $$ [Hsnapshot Hstate] with ⟨Hsnapshot, Hstate⟩
    · iframe Hsnapshot Hstate
      all_goals try iframe HinvGlobal Hinv
      all_goals try iframe HrevGlobal Hrev
    cases rejected : active && constraintRejected kind changing key m op rv <;> simp only [rejected, Bool.false_eq_true, ite_false, ite_true, ownsBranch]
    all_goals wp_end
    all_goals iframe Hsnapshot Hoperators
    all_goals try iframe Hstate
    all_goals ipureintro
    all_goals simp [validBranchOption, hv]

  | leftFalse left right key row kind op lv rv changing active sf lf rf normal wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hkey metadata ht hnil hlookup hkind hchanging =>
    have hv := constraint_state_valid s kindOf changingOf m key active kind normal changing rv hm metadata hkind hchanging
    wp_start_folded as ⟨#Hsnapshot, #Hoperators, Hstate⟩
    ihave #Hctx := Hoperators
    iunfold assumeOperatorsOwn at Hctx
    icases Hctx with ⟨HinvGlobal, Hinv, HrevGlobal, Hrev⟩
    wp_apply wp_comparison_left_numeric_false receiver s state m ref left right key row kind op lv rv changing active sf lf rf o.inverseLoc o.inverseTable normal wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hkey metadata ht hnil hlookup hparse $$ [Hsnapshot Hstate] with ⟨_, _, Hsnapshot, Hstate⟩
    · iframe Hsnapshot Hstate
      all_goals try iframe HinvGlobal Hinv
      all_goals try iframe HrevGlobal Hrev
    cases rejected : active && constraintRejected kind changing key m normal rv <;> simp only [rejected, Bool.false_eq_true, ite_false, ite_true, ownsBranch]
    all_goals wp_end
    all_goals iframe Hsnapshot Hoperators
    all_goals try iframe Hstate
    all_goals ipureintro
    all_goals simp [validBranchOption, hv]

  | rightTrue left right key row kind op lv rv changing active sf lf rf normal wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hkey metadata ht hnil hlookup hkind hchanging =>
    have hv := constraint_state_valid s kindOf changingOf m key active kind normal changing lv hm metadata hkind hchanging
    wp_start_folded as ⟨#Hsnapshot, #Hoperators, Hstate⟩
    ihave #Hctx := Hoperators
    iunfold assumeOperatorsOwn at Hctx
    icases Hctx with ⟨HinvGlobal, Hinv, HrevGlobal, Hrev⟩
    wp_apply wp_comparison_right_numeric_true receiver s state m ref left right key row kind op lv rv changing active sf lf rf o.reverseLoc o.reverseTable normal wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hkey metadata ht hnil hlookup hparse $$ [Hsnapshot Hstate] with ⟨_, _, Hsnapshot, Hstate⟩
    · iframe Hsnapshot Hstate
      all_goals try iframe HinvGlobal Hinv
      all_goals try iframe HrevGlobal Hrev
    cases rejected : active && constraintRejected kind changing key m normal lv <;> simp only [rejected, Bool.false_eq_true, ite_false, ite_true, ownsBranch]
    all_goals wp_end
    all_goals iframe Hsnapshot Hoperators
    all_goals try iframe Hstate
    all_goals ipureintro
    all_goals simp [validBranchOption, hv]

  | rightFalse left right key row kind op lv rv changing active sf lf rf normal negated wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hkey metadata ht hnil hlookup hnegate hkind hchanging =>
    have hv := constraint_state_valid s kindOf changingOf m key active kind normal changing lv hm metadata hkind hchanging
    wp_start_folded as ⟨#Hsnapshot, #Hoperators, Hstate⟩
    ihave #Hctx := Hoperators
    iunfold assumeOperatorsOwn at Hctx
    icases Hctx with ⟨HinvGlobal, Hinv, HrevGlobal, Hrev⟩
    wp_apply wp_comparison_right_numeric_false receiver s state m ref left right key row kind op lv rv changing active sf lf rf o.reverseLoc o.reverseTable normal negated o.inverseLoc o.inverseTable wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hkey metadata ht hnil hlookup hnegate hparse $$ [Hsnapshot Hstate] with ⟨_, _, _, _, Hsnapshot, Hstate⟩
    · iframe Hsnapshot Hstate
      all_goals try iframe HinvGlobal Hinv
      all_goals try iframe HrevGlobal Hrev
    cases rejected : active && constraintRejected kind changing key m normal lv <;> simp only [rejected, Bool.false_eq_true, ite_false, ite_true, ownsBranch]
    all_goals wp_end
    all_goals iframe Hsnapshot Hoperators
    all_goals try iframe Hstate
    all_goals ipureintro
    all_goals simp [validBranchOption, hv]

  | bothTrue left right row c sf lf rf wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hleftNonzero hrightNonzero hleftMetadata hrightMetadata ht hnill hnilr hreverse hkindl hkindr hchangingl hchangingr =>
    have hv := comparison_result_valid s kindOf changingOf m c hm hleftMetadata hrightMetadata hkindl hkindr hchangingl hchangingr
    wp_start_folded as ⟨#Hsnapshot, #Hoperators, Hstate⟩
    ihave #Hctx := Hoperators
    iunfold assumeOperatorsOwn at Hctx
    icases Hctx with ⟨HinvGlobal, Hinv, HrevGlobal, Hrev⟩
    wp_apply wp_comparison_both_numeric_true receiver s state m ref left right row c sf lf rf o.reverseLoc o.reverseTable wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hleftNonzero hrightNonzero hleftMetadata hrightMetadata ht hnill hnilr hreverse hparse $$ [Hsnapshot Hstate] with ⟨_, _, Hsnapshot, Hstate⟩
    · iframe Hsnapshot Hstate
      all_goals try iframe HinvGlobal Hinv
      all_goals try iframe HrevGlobal Hrev
    cases rejected : comparisonRejected c m <;> simp only [rejected, Bool.false_eq_true, ite_false, ite_true, ownsBranch]
    all_goals wp_end
    all_goals iframe Hsnapshot Hoperators
    all_goals try iframe Hstate
    all_goals ipureintro
    all_goals simp [validBranchOption, hv]

  | bothFalse left right row c sf lf rf rawOperator wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hleftNonzero hrightNonzero hleftMetadata hrightMetadata ht hnill hnilr hreverse hnegate hkindl hkindr hchangingl hchangingr =>
    have hv := comparison_result_valid s kindOf changingOf m c hm hleftMetadata hrightMetadata hkindl hkindr hchangingl hchangingr
    wp_start_folded as ⟨#Hsnapshot, #Hoperators, Hstate⟩
    ihave #Hctx := Hoperators
    iunfold assumeOperatorsOwn at Hctx
    icases Hctx with ⟨HinvGlobal, Hinv, HrevGlobal, Hrev⟩
    wp_apply wp_comparison_both_numeric_false receiver s state m ref left right row c sf lf rf op o.inverseLoc o.inverseTable o.reverseLoc o.reverseTable wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hleftNonzero hrightNonzero hleftMetadata hrightMetadata ht hnill hnilr hreverse hnegate hparse $$ [Hsnapshot Hstate] with ⟨_, _, _, _, Hsnapshot, Hstate⟩
    · iframe Hsnapshot Hstate
      all_goals try iframe HinvGlobal Hinv
      all_goals try iframe HrevGlobal Hrev
    cases rejected : comparisonRejected c m <;> simp only [rejected, Bool.false_eq_true, ite_false, ite_true, ownsBranch]
    all_goals wp_end
    all_goals iframe Hsnapshot Hoperators
    all_goals try iframe Hstate
    all_goals ipureintro
    all_goals simp [validBranchOption, hv]

theorem numeric_execution_valid (s : QuerySnapshot) (o : AssumeOperators)
    (kindOf : w32 → GoString) (changingOf : w32 → Bool) (m : StateData)
    (ref : w32) (op : GoString) (truth : Bool) (result : Option StateData)
    (hm : validBranchState s.kinds s.unstable kindOf changingOf m)
    (execution : NumericExecution s o kindOf changingOf m ref op truth result) :
    validBranchOption s.kinds s.unstable kindOf changingOf result := by
  cases execution with
  | leftTrue left right key row kind op lv rv changing active sf lf rf wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hkey metadata ht hnil hkind hchanging =>
    have hv := constraint_state_valid s kindOf changingOf m key active kind op changing rv hm metadata hkind hchanging
    cases rejected : active && constraintRejected kind changing key m op rv <;> simp [rejected, validBranchOption, hv]
  | leftFalse left right key row kind op lv rv changing active sf lf rf normal wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hkey metadata ht hnil hlookup hkind hchanging =>
    have hv := constraint_state_valid s kindOf changingOf m key active kind normal changing rv hm metadata hkind hchanging
    cases rejected : active && constraintRejected kind changing key m normal rv <;> simp [rejected, validBranchOption, hv]
  | rightTrue left right key row kind op lv rv changing active sf lf rf normal wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hkey metadata ht hnil hlookup hkind hchanging =>
    have hv := constraint_state_valid s kindOf changingOf m key active kind normal changing lv hm metadata hkind hchanging
    cases rejected : active && constraintRejected kind changing key m normal lv <;> simp [rejected, validBranchOption, hv]
  | rightFalse left right key row kind op lv rv changing active sf lf rf normal negated wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hkey metadata ht hnil hlookup hnegate hkind hchanging =>
    have hv := constraint_state_valid s kindOf changingOf m key active kind normal changing lv hm metadata hkind hchanging
    cases rejected : active && constraintRejected kind changing key m normal lv <;> simp [rejected, validBranchOption, hv]
  | bothTrue left right row c sf lf rf wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hleftNonzero hrightNonzero hleftMetadata hrightMetadata ht hnill hnilr hreverse hkindl hkindr hchangingl hchangingr =>
    have hv := comparison_result_valid s kindOf changingOf m c hm hleftMetadata hrightMetadata hkindl hkindr hchangingl hchangingr
    cases rejected : comparisonRejected c m <;> simp [rejected, validBranchOption, hv]
  | bothFalse left right row c sf lf rf rawOperator wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hleftNonzero hrightNonzero hleftMetadata hrightMetadata ht hnill hnilr hreverse hnegate hkindl hkindr hchangingl hchangingr =>
    have hv := comparison_result_valid s kindOf changingOf m c hm hleftMetadata hrightMetadata hkindl hkindr hchangingl hchangingr
    cases rejected : comparisonRejected c m <;> simp [rejected, validBranchOption, hv]

def valueIntervalMeaning (s : QuerySnapshot) (kind : w32 → GoString) (changing : w32 → Bool)
    (env values : w32 → Int) : Prop :=
  ∀ m sf fuel ref v, stateDenotes kind changing m env → valueResolve s m sf fuel ref = some v →
    interval.contains (bounds v) (values ref)

def symbolIntervalMeaning (s : QuerySnapshot) (env values : w32 → Int) : Prop :=
  ∀ sf ref key, symbolResolve s sf ref = some key → key ≠ W32 0 → values ref = env key

def numericTruth (s : QuerySnapshot) (ref : w32) (op : GoString) (truth : Bool) (values : w32 → Int) : Prop :=
  ∀ row left right, s.operands[(uint.Z ref - 1).toNat]? = some row →
    row.arr[0]? = some left → row.arr[1]? = some right → comparisonBranch truth op (values left) (values right)

theorem numeric_execution_sound (s : QuerySnapshot) (o : AssumeOperators)
    (kindOf : w32 → GoString) (changingOf : w32 → Bool) (m : StateData)
    (ref : w32) (op : GoString) (truth : Bool) (result : Option StateData)
    (execution : NumericExecution s o kindOf changingOf m ref op truth result)
    (env values : w32 → Int) (hin : stateDenotes kindOf changingOf m env)
    (hvalues : valueIntervalMeaning s kindOf changingOf env values)
    (hsymbols : symbolIntervalMeaning s env values)
    (hword : ∀ key, -9223372036854775808 ≤ env key ∧ env key ≤ 9223372036854775807)
    (htruth : numericTruth s ref op truth values)
    (hinverse : ∀ op, (o.inverseTable !! op).getD (zero_val GoString) = inverseOperator op)
    (hreverse : ∀ op, (o.reverseTable !! op).getD (zero_val GoString) = reverseOperator op) :
    branchDenotes kindOf changingOf result env := by
  cases execution with
  | leftTrue left right key row kind op lv rv changing active sf lf rf wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hkey metadata ht hnil hkind hchanging =>
    have hmeaning := hsymbols sf left key hlsymbol hkey
    have relations := normalizedBranchRelations true op (values left) (values right) (htruth row left right hrow hleft hright)
    have hrel : integerRelation op (env key) (values right) := by
      simpa only [normalizedOperator, ite_true, hmeaning] using relations.1
    have hs := numeric_bound_state_sound kindOf changingOf m env key active op rv (values right) hin (hword key)
      (hvalues m sf rf right rv hin hrvalue) hrel
    have hs' : (active && constraintRejected kind changing key m op rv) = false ∧
      stateDenotes kindOf changingOf (constraintState active kind changing key m op rv) env := by
      simpa only [hkind, hchanging, stateRepresents, stateDenotes] using hs
    simpa only [hs'.1, Bool.false_eq_true, ite_false, branchDenotes] using hs'.2

  | leftFalse left right key row kind op lv rv changing active sf lf rf normal wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hkey metadata ht hnil hlookup hkind hchanging =>
    have hmeaning := hsymbols sf left key hlsymbol hkey
    have relations := normalizedBranchRelations false op (values left) (values right) (htruth row left right hrow hleft hright)
    have hnormal : normal = inverseOperator op := hlookup.symm.trans (hinverse op)
    have hrel : integerRelation normal (env key) (values right) := by
      simpa only [normalizedOperator, Bool.false_eq_true, ite_false, hmeaning, ← hnormal] using relations.1
    have hs := numeric_bound_state_sound kindOf changingOf m env key active normal rv (values right) hin (hword key)
      (hvalues m sf rf right rv hin hrvalue) hrel
    have hs' : (active && constraintRejected kind changing key m normal rv) = false ∧
      stateDenotes kindOf changingOf (constraintState active kind changing key m normal rv) env := by
      simpa only [hkind, hchanging, stateRepresents, stateDenotes] using hs
    simpa only [hs'.1, Bool.false_eq_true, ite_false, branchDenotes] using hs'.2

  | rightTrue left right key row kind op lv rv changing active sf lf rf normal wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hkey metadata ht hnil hlookup hkind hchanging =>
    have hmeaning := hsymbols sf right key hrsymbol hkey
    have relations := normalizedBranchRelations true op (values left) (values right) (htruth row left right hrow hleft hright)
    have hnormal : normal = reverseOperator op := hlookup.symm.trans (hreverse _)
    have hrel : integerRelation normal (env key) (values left) := by
      simpa only [normalizedOperator, ite_true, hmeaning, ← hnormal] using relations.2
    have hs := numeric_bound_state_sound kindOf changingOf m env key active normal lv (values left) hin (hword key)
      (hvalues m sf lf left lv hin hlvalue) hrel
    have hs' : (active && constraintRejected kind changing key m normal lv) = false ∧
      stateDenotes kindOf changingOf (constraintState active kind changing key m normal lv) env := by
      simpa only [hkind, hchanging, stateRepresents, stateDenotes] using hs
    simpa only [hs'.1, Bool.false_eq_true, ite_false, branchDenotes] using hs'.2

  | rightFalse left right key row kind op lv rv changing active sf lf rf normal negated wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hkey metadata ht hnil hlookup hnegate hkind hchanging =>
    have hmeaning := hsymbols sf right key hrsymbol hkey
    have relations := normalizedBranchRelations false op (values left) (values right) (htruth row left right hrow hleft hright)
    have hnegated : negated = inverseOperator op := hnegate.symm.trans (hinverse op)
    have hnormal : normal = reverseOperator negated := hlookup.symm.trans (hreverse _)
    have hrel : integerRelation normal (env key) (values left) := by
      simpa only [normalizedOperator, Bool.false_eq_true, ite_false, hmeaning, ← hnormal, ← hnegated] using relations.2
    have hs := numeric_bound_state_sound kindOf changingOf m env key active normal lv (values left) hin (hword key)
      (hvalues m sf lf left lv hin hlvalue) hrel
    have hs' : (active && constraintRejected kind changing key m normal lv) = false ∧
      stateDenotes kindOf changingOf (constraintState active kind changing key m normal lv) env := by
      simpa only [hkind, hchanging, stateRepresents, stateDenotes] using hs
    simpa only [hs'.1, Bool.false_eq_true, ite_false, branchDenotes] using hs'.2

  | bothTrue left right row c sf lf rf wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hleftNonzero hrightNonzero hleftMetadata hrightMetadata ht hnill hnilr hreverse hkindl hkindr hchangingl hchangingr =>
    have hlmeaning := hsymbols sf left c.leftKey hlsymbol hleftNonzero
    have hrmeaning := hsymbols sf right c.rightKey hrsymbol hrightNonzero
    have relations := normalizedBranchRelations true c.operator (values left) (values right) (htruth row left right hrow hleft hright)
    have hr : c.reversed = reverseOperator c.operator := hreverse.symm.trans (by simpa using ‹∀ op, (o.reverseTable !! op).getD (zero_val GoString) = reverseOperator op› c.operator)
    have rel : integerRelation c.operator (env c.leftKey) (env c.rightKey) ∧ integerRelation c.reversed (env c.rightKey) (env c.leftKey) := by
      simpa only [normalizedOperator, ite_true, ← hr, hlmeaning, hrmeaning] using relations
    have hs := comparison_numeric_state_sound c m kindOf changingOf env hkindl hkindr hchangingl hchangingr hin
      (by simpa only [hlmeaning] using hvalues m sf lf left c.leftValue hin hlvalue)
      (by simpa only [hrmeaning] using hvalues m sf rf right c.rightValue hin hrvalue)
      (hword c.leftKey) (hword c.rightKey) rel.1 rel.2
    simpa only [hs.1, Bool.false_eq_true, ite_false, branchDenotes, stateDenotes, stateRepresents] using hs.2

  | bothFalse left right row c sf lf rf rawOperator wf href hrow hleft hright hlvalid hrvalid hlvalue hrvalue hlsymbol hrsymbol hleftNonzero hrightNonzero hleftMetadata hrightMetadata ht hnill hnilr hreverse hnegate hkindl hkindr hchangingl hchangingr =>
    have hlmeaning := hsymbols sf left c.leftKey hlsymbol hleftNonzero
    have hrmeaning := hsymbols sf right c.rightKey hrsymbol hrightNonzero
    have relations := normalizedBranchRelations false op (values left) (values right) (htruth row left right hrow hleft hright)
    have hn : c.operator = inverseOperator op := hnegate.symm.trans (hinverse op)
    have hr : c.reversed = reverseOperator c.operator := hreverse.symm.trans (by simpa using ‹∀ op, (o.reverseTable !! op).getD (zero_val GoString) = reverseOperator op› c.operator)
    have rel : integerRelation c.operator (env c.leftKey) (env c.rightKey) ∧ integerRelation c.reversed (env c.rightKey) (env c.leftKey) := by
      simpa only [normalizedOperator, Bool.false_eq_true, ite_false, ← hn, ← hr, hlmeaning, hrmeaning] using relations
    have hs := comparison_numeric_state_sound c m kindOf changingOf env hkindl hkindr hchangingl hchangingr hin
      (by simpa only [hlmeaning] using hvalues m sf lf left c.leftValue hin hlvalue)
      (by simpa only [hrmeaning] using hvalues m sf rf right c.rightValue hin hrvalue)
      (hword c.leftKey) (hword c.rightKey) rel.1 rel.2
    simpa only [hs.1, Bool.false_eq_true, ite_false, branchDenotes, stateDenotes, stateRepresents] using hs.2

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
