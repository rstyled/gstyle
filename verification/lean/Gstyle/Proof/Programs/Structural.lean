module
public import Gstyle.Proof.Programs.Contract
@[expose] public section
namespace Perennial.GstylePrograms

instance (r : Reported) (x : Int) : Decidable (r.contains x) :=
  inferInstanceAs (Decidable (_ ∧ _))

/-- Input domains use mathematical endpoints, so strict comparisons cannot overflow. -/
structure Domain where
  lower : Int
  upper : Int
  deriving DecidableEq, Repr

def Domain.contains (d : Domain) (x : Int) : Prop := d.lower ≤ x ∧ x ≤ d.upper

def initialDomain : Domain := ⟨-(2^63), 2^63 - 1⟩

def Comparison.inverse : Comparison → Comparison
  | .lt => .ge | .le => .gt | .gt => .le | .ge => .lt | .eq => .ne | .ne => .eq

theorem inverse_holds (op : Comparison) (x bound : Int) :
    op.inverse.holds x bound ↔ ¬ op.holds x bound := by
  cases op <;> simp [Comparison.inverse, Comparison.holds] <;> omega

def Domain.restrict (d : Domain) (c : Condition) : Domain :=
  match c.op with
  | .lt => ⟨d.lower, min d.upper (c.bound - 1)⟩
  | .le => ⟨d.lower, min d.upper c.bound⟩
  | .gt => ⟨max d.lower (c.bound + 1), d.upper⟩
  | .ge => ⟨max d.lower c.bound, d.upper⟩
  | .eq => ⟨max d.lower c.bound, min d.upper c.bound⟩
  | .ne => d

theorem restrict_sound (d : Domain) (c : Condition) (x : Int)
    (hx : d.contains x) (hc : c.op.holds x c.bound) :
    (d.restrict c).contains x := by
  cases c with
  | mk op bound =>
    cases op <;> simp_all [Domain.restrict, Domain.contains, Comparison.holds] <;> omega

/-- Cover both endpoints of a nonempty interval. Unknown report bounds impose no constraint. -/
def Domain.covered (d : Domain) (r : Reported) : Bool :=
  decide (d.upper < d.lower) ||
    ((Bool.not r.hasLower || decide (r.lower ≤ d.lower)) &&
     (Bool.not r.hasUpper || decide (d.upper ≤ r.upper)))

theorem covered_sound (d : Domain) (r : Reported) (x : Int)
    (hc : d.covered r = true) (hx : d.contains x) : r.contains x := by
  cases hl : r.hasLower <;> cases hu : r.hasUpper
  all_goals simp_all [Domain.covered, Domain.contains, Reported.contains] <;> omega

/-- A terminating structural checker, with no syntax-depth or constant alphabet bound. -/
def checkTree : Program → Domain → Reported → Bool
  | p, d, r =>
    if d.upper < d.lower then true else
    match p with
    | .result .input => d.covered r
    | .result (.literal value) => decide (r.contains value)
    | .branch c yes no =>
      checkTree yes (d.restrict c) r &&
      checkTree no (d.restrict ⟨c.op.inverse, c.bound⟩) r

/-- No hypothesis assumes that the analyzer's answer or a child answer is sound. -/
theorem checkTree_sound (p : Program) (d : Domain) (r : Reported) (x : Int)
    (checked : checkTree p d r = true) (hx : d.contains x) :
    r.contains (p.eval x) := by
  induction p generalizing d with
  | result value =>
    have nonempty : ¬d.upper < d.lower := by rcases hx with ⟨_, _⟩; omega
    cases value with
    | input => exact covered_sound d r x (by simpa [checkTree, nonempty] using checked) hx
    | literal value => simpa [checkTree, nonempty, Program.eval, Expression.eval] using checked
  | branch c yes no ihYes ihNo =>
    have nonempty : ¬d.upper < d.lower := by rcases hx with ⟨_, _⟩; omega
    have children := Bool.and_eq_true_iff.mp (by simpa [checkTree, nonempty] using checked)
    by_cases hc : c.op.holds x c.bound
    · simpa [Program.eval, hc] using ihYes (d.restrict c) children.1 (restrict_sound d c x hx hc)
    · have hn : c.op.inverse.holds x c.bound := (inverse_holds c.op x c.bound).mpr hc
      simpa [Program.eval, hc] using ihNo (d.restrict ⟨c.op.inverse, c.bound⟩) children.2
        (restrict_sound d ⟨c.op.inverse, c.bound⟩ x hx hn)

def checkPrecision : Program → Reported → Bool
  | .result (.literal n), r =>
    r.hasLower && r.hasUpper && decide (r.lower = n) && decide (r.upper = n)
  | _, _ => true

theorem checkPrecision_exact (p : Program) (r : Reported)
    (checked : checkPrecision p r = true) : PreciseLiteral p r := by
  intro n hp
  subst p
  simpa [checkPrecision, Bool.and_eq_true, and_assoc] using checked

def checkReport (p : Program) (r : Reported) : Bool :=
  checkTree p initialDomain r && checkPrecision p r

/-- One theorem checks reports for all finite branch trees and all integer constants. -/
theorem checkReport_sound (p : Program) (r : Reported)
    (checked : checkReport p r = true) : Sound p r ∧ PreciseLiteral p r := by
  obtain ⟨tree, precision⟩ := Bool.and_eq_true_iff.mp checked
  constructor
  · intro x hx
    apply checkTree_sound p initialDomain r x tree
    rcases hx with ⟨hl, hu⟩
    constructor <;> simp [initialDomain] <;> omega
  · exact checkPrecision_exact p r precision

/-- Compose raw checked answers with the existing certificate and interval contracts. -/
def certificateOfCheck (name : String) (p : Program) (r : Reported) (source : String)
    (identity : p.source name = source)
    (bounds : Signed64 r.lower ∧ Signed64 r.upper)
    (checked : checkReport p r = true) : Certificate :=
  ⟨name, p, r, source, identity, bounds, (checkReport_sound p r checked).1,
    (checkReport_sound p r checked).2⟩

section interval
variable [FfiSyntax]

theorem checked_interval_sound (p : Program) (r : Reported)
    (bounds : Signed64 r.lower ∧ Signed64 r.upper)
    (checked : checkReport p r = true) (x : Int) (hx : Signed64 x) :
    github_com.rstyled.gstyle.domain.runtimeguard.interval.contains r.interval (p.eval x) := by
  exact (reported_interval r bounds.1 bounds.2 _).mpr ((checkReport_sound p r checked).1 x hx)

end interval

end Perennial.GstylePrograms
