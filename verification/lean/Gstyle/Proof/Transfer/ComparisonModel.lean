module
public import Gstyle.Proof.Transfer.QuerySnapshot
public import Gstyle.Proof.Transfer.NumericModel
@[expose] public section
noncomputable section
namespace Perennial
namespace github_com.rstyled.gstyle.domain.runtimeguard

def comparisonOperator (op : GoString) : Prop :=
  op = go!"<" ∨ op = go!"<=" ∨ op = go!">" ∨ op = go!">=" ∨ op = go!"==" ∨ op = go!"!="

def inverseOperator (op : GoString) : GoString :=
  if op = go!"<" then go!">=" else
  if op = go!"<=" then go!">" else
  if op = go!">" then go!"<=" else
  if op = go!">=" then go!"<" else
  if op = go!"==" then go!"!=" else
  if op = go!"!=" then go!"==" else go!""

def reverseOperator (op : GoString) : GoString :=
  if op = go!"<" then go!">" else
  if op = go!"<=" then go!">=" else
  if op = go!">" then go!"<" else
  if op = go!">=" then go!"<=" else
  if op = go!"==" then go!"==" else
  if op = go!"!=" then go!"!=" else go!""

def integerRelation (op : GoString) (left right : Int) : Prop :=
  if op = go!"<" then left < right else
  if op = go!"<=" then left ≤ right else
  if op = go!">" then left > right else
  if op = go!">=" then left ≥ right else
  if op = go!"==" then left = right else
  if op = go!"!=" then left ≠ right else True

theorem inverseOperator_relation (op : GoString) (left right : Int) (h : comparisonOperator op) :
    integerRelation (inverseOperator op) left right ↔ ¬integerRelation op left right := by
  rcases h with h | h | h | h | h | h
  all_goals subst op
  all_goals simp [inverseOperator, integerRelation]
  all_goals try omega

theorem reverseOperator_relation (op : GoString) (left right : Int) (h : comparisonOperator op) :
    integerRelation (reverseOperator op) right left ↔ integerRelation op left right := by
  rcases h with h | h | h | h | h | h
  all_goals subst op
  all_goals simp [reverseOperator, integerRelation]
  all_goals try omega

theorem inverseOperator_involution (op : GoString) (h : comparisonOperator op) :
    inverseOperator (inverseOperator op) = op := by
  rcases h with h | h | h | h | h | h
  all_goals subst op; try decide

theorem reverseOperator_involution (op : GoString) (h : comparisonOperator op) :
    reverseOperator (reverseOperator op) = op := by
  rcases h with h | h | h | h | h | h
  all_goals subst op; try decide

def normalizedOperator (truth : Bool) (op : GoString) : GoString :=
  if truth then op else inverseOperator op

theorem normalizedOperator_relation (truth : Bool) (op : GoString) (left right : Int)
    (h : comparisonOperator op) :
    integerRelation (normalizedOperator truth op) left right ↔
      if truth then integerRelation op left right else ¬integerRelation op left right := by
  cases truth with
  | false =>
    simp only [normalizedOperator, Bool.false_eq_true, ite_false]
    try exact inverseOperator_relation op left right h
  | true => simp only [normalizedOperator, ite_true]

theorem inverse_reverse_commute (op : GoString) (h : comparisonOperator op) :
    inverseOperator (reverseOperator op) = reverseOperator (inverseOperator op) := by
  rcases h with h | h | h | h | h | h
  all_goals subst op; try decide

theorem unknownOperator_relation (op : GoString) (left right : Int) (h : ¬comparisonOperator op) :
    integerRelation op left right ∧ inverseOperator op = go!"" ∧ reverseOperator op = go!"" := by
  simp only [comparisonOperator, not_or] at h
  rcases h with ⟨h1, h2, h3, h4, h5, h6⟩
  simp only [integerRelation, inverseOperator, reverseOperator, h1, h2, h3, h4, h5, h6, ite_false]
  try trivial

theorem reversedNormalizedOperator_relation (truth : Bool) (op : GoString) (left right : Int)
    (h : comparisonOperator op) :
    integerRelation (reverseOperator (normalizedOperator truth op)) right left ↔
      if truth then integerRelation op left right else ¬integerRelation op left right := by
  rcases h with h | h | h | h | h | h
  all_goals subst op; cases truth
  all_goals simp [normalizedOperator, inverseOperator, reverseOperator, integerRelation]
  all_goals try omega

def comparisonBranch (truth : Bool) (op : GoString) (left right : Int) : Prop :=
  comparisonOperator op → if truth then integerRelation op left right else ¬integerRelation op left right

theorem normalizedBranchRelations (truth : Bool) (op : GoString) (left right : Int)
    (hbranch : comparisonBranch truth op left right) :
    integerRelation (normalizedOperator truth op) left right ∧
      integerRelation (reverseOperator (normalizedOperator truth op)) right left := by
  by_cases hknown : comparisonOperator op
  · have hcondition : if truth then integerRelation op left right else ¬integerRelation op left right := hbranch hknown
    exact ⟨(normalizedOperator_relation truth op left right hknown).mpr hcondition,
      (reversedNormalizedOperator_relation truth op left right hknown).mpr hcondition⟩
  · simp only [comparisonOperator, not_or] at hknown
    rcases hknown with ⟨h1, h2, h3, h4, h5, h6⟩
    cases truth <;> simp [normalizedOperator, inverseOperator, reverseOperator, integerRelation, h1, h2, h3, h4, h5, h6]

inductive ConstraintMetadata [FfiSyntax] (s : QuerySnapshot) (key : w32) (kind : GoString)
    (changing active : Bool) : Prop where
  | present
      (hk : 0 < uint.Z key ∧ uint.Z key ≤ s.kinds.length)
      (hkind : s.kinds[(uint.Z key - 1).toNat]? = some kind)
      (hunstable : s.unstable[(uint.Z key - 1).toNat]? = some changing)
      (htracked : s.tracked[(uint.Z key - 1).toNat]? = some active) :
      ConstraintMetadata s key kind changing active
  | absent (hout : key = W32 0 ∨ s.kinds.length < uint.Z key) (ha : active = false) :
      ConstraintMetadata s key kind changing active

structure NumericComparison [FfiSyntax] where
  leftKey : w32
  rightKey : w32
  leftKind : GoString
  rightKind : GoString
  leftChanging : Bool
  rightChanging : Bool
  leftActive : Bool
  rightActive : Bool
  operator : GoString
  reversed : GoString
  leftValue : Value
  rightValue : Value

def comparisonLeftRejected [FfiSyntax] (c : NumericComparison) (m : StateData) : Bool :=
  c.leftActive && constraintRejected c.leftKind c.leftChanging c.leftKey m c.operator c.rightValue

def comparisonLeftMap [FfiSyntax] (c : NumericComparison) (m : StateData) : StateData :=
  constraintState c.leftActive c.leftKind c.leftChanging c.leftKey m c.operator c.rightValue

def comparisonRightRejected [FfiSyntax] (c : NumericComparison) (m : StateData) : Bool :=
  c.rightActive && constraintRejected c.rightKind c.rightChanging c.rightKey (comparisonLeftMap c m) c.reversed c.leftValue

def comparisonRightMap [FfiSyntax] (c : NumericComparison) (m : StateData) : StateData :=
  constraintState c.rightActive c.rightKind c.rightChanging c.rightKey (comparisonLeftMap c m) c.reversed c.leftValue

def comparisonRejected [FfiSyntax] (c : NumericComparison) (m : StateData) : Bool :=
  comparisonLeftRejected c m || comparisonRightRejected c m

def comparisonResultMap [FfiSyntax] (c : NumericComparison) (m : StateData) : StateData :=
  if comparisonLeftRejected c m then comparisonLeftMap c m else comparisonRightMap c m

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
