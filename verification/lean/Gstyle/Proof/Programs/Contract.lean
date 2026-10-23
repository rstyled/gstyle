module
public import Gstyle.Proof.Interval.Base
@[expose] public section
namespace Perennial.GstylePrograms
open github_com.rstyled.gstyle.domain.runtimeguard

/-- The complete input domain of the generated Go functions. -/
def Signed64 (x : Int) : Prop := -(2^63) ≤ x ∧ x < 2^63

instance (x : Int) : Decidable (Signed64 x) :=
  inferInstanceAs (Decidable (_ ∧ _))

inductive Comparison where
  | lt | le | gt | ge | eq | ne
  deriving DecidableEq, Repr

def Comparison.source : Comparison → String
  | .lt => "<" | .le => "<=" | .gt => ">" | .ge => ">=" | .eq => "==" | .ne => "!="

def Comparison.holds : Comparison → Int → Int → Prop
  | .lt, x, y => x < y | .le, x, y => x ≤ y
  | .gt, x, y => x > y | .ge, x, y => x ≥ y
  | .eq, x, y => x = y | .ne, x, y => x ≠ y

instance (op : Comparison) (x y : Int) : Decidable (op.holds x y) := by
  cases op <;> unfold Comparison.holds <;> infer_instance

inductive Expression where
  | input
  | literal (value : Int)
  deriving DecidableEq, Repr

def Expression.eval : Expression → Int → Int
  | .input, x => x | .literal value, _ => value

def Expression.source : Expression → String
  | .input => "x" | .literal value => toString value

structure Condition where
  op : Comparison
  bound : Int
  deriving DecidableEq, Repr

inductive Program where
  | result (value : Expression)
  | branch (condition : Condition) (yes no : Program)
  deriving DecidableEq, Repr

def Program.eval : Program → Int → Int
  | .result value, x => value.eval x
  | .branch c yes no, x => if c.op.holds x c.bound then yes.eval x else no.eval x

def Program.body : Program → String
  | .result value => "result = " ++ value.source ++ "\n"
  | .branch c yes no => "if x " ++ c.op.source ++ " " ++ toString c.bound ++ " {\n" ++
      yes.body ++ "} else {\n" ++ no.body ++ "}\n"

def Program.source (name : String) (p : Program) : String :=
  "func " ++ name ++ "(x int64) int64 {\nvar result int64\n_ = *new(int)\n" ++ p.body ++
    "observe(result)\nreturn result\n}\n"

/-- Raw numeric fields returned by the production runtime analysis. -/
structure Reported where
  lower : Int
  upper : Int
  hasLower : Bool
  hasUpper : Bool
  deriving DecidableEq, Repr

def Reported.contains (r : Reported) (x : Int) : Prop :=
  (r.hasLower = true → r.lower ≤ x) ∧ (r.hasUpper = true → x ≤ r.upper)

/-- Every possible result, not just sampled executions, fits the actual report. -/
def Sound (p : Program) (r : Reported) : Prop :=
  ∀ x : Int, Signed64 x → r.contains (p.eval x)

/-- Precision is mandatory for direct constants: reporting unknown everywhere fails. -/
def PreciseLiteral (p : Program) (r : Reported) : Prop :=
  ∀ n, p = .result (.literal n) →
    r.hasLower = true ∧ r.hasUpper = true ∧ r.lower = n ∧ r.upper = n

structure Certificate where
  name : String
  program : Program
  report : Reported
  source : String
  sourceMatches : program.source name = source
  bounds : Signed64 report.lower ∧ Signed64 report.upper
  sound : Sound program report
  precision : PreciseLiteral program report

/-- Enumerate every syntax tree up to depth, over the supplied finite alphabets. -/
def enumerate (leaves : List Expression) (tests : List Condition) : Nat → List Program
  | 0 => leaves.map Program.result
  | depth + 1 => leaves.map Program.result ++ tests.flatMap (fun c =>
      (enumerate leaves tests depth).flatMap (fun yes =>
        (enumerate leaves tests depth).map (Program.branch c yes)))

/-- Admission is structural and independent of the analyzer's success. -/
def Admitted (leaves : List Expression) (tests : List Condition) : Nat → Program → Prop
  | _, .result e => e ∈ leaves
  | 0, .branch .. => False
  | depth + 1, .branch c yes no =>
    c ∈ tests ∧ Admitted leaves tests depth yes ∧ Admitted leaves tests depth no

theorem enumerate_complete (leaves : List Expression) (tests : List Condition)
    (depth : Nat) (p : Program) :
    p ∈ enumerate leaves tests depth ↔ Admitted leaves tests depth p := by
  induction depth generalizing p with
  | zero => cases p <;> simp [enumerate, Admitted]
  | succ depth ih =>
    cases p with
    | result value => simp [enumerate, Admitted, List.mem_flatMap]
    | branch c yes no =>
      simp only [enumerate, List.mem_append, List.mem_map, List.mem_flatMap,
        Program.branch.injEq, ih, Admitted]
      constructor
      · rintro (⟨e, he, impossible⟩ | ⟨c, hc, yes, hy, no, hn, rfl, rfl, rfl⟩)
        · cases impossible
        · exact ⟨hc, hy, hn⟩
      · rintro ⟨hc, hy, hn⟩
        exact Or.inr ⟨c, hc, yes, hy, no, hn, rfl, rfl, rfl⟩

/-- Coverage equality prevents a certificate collection from omitting hard cases. -/
theorem certified_programs (certificates : List Certificate) (programs : List Program)
    (coverage : certificates.map Certificate.program = programs)
    (p : Program) (hp : p ∈ programs) :
    ∃ c ∈ certificates, c.program = p ∧ Sound p c.report ∧ PreciseLiteral p c.report := by
  rw [← coverage] at hp
  obtain ⟨c, hc, he⟩ := List.mem_map.mp hp
  exact ⟨c, hc, he, he ▸ c.sound, he ▸ c.precision⟩

theorem certified_admitted (certificates : List Certificate)
    (leaves : List Expression) (tests : List Condition) (depth : Nat)
    (coverage : certificates.map Certificate.program = enumerate leaves tests depth)
    (p : Program) (hp : Admitted leaves tests depth p) :
    ∃ c ∈ certificates, c.program = p ∧ Sound p c.report ∧ PreciseLiteral p c.report := by
  exact certified_programs certificates _ coverage p ((enumerate_complete leaves tests depth p).mpr hp)

section interval
variable [FfiSyntax]

/-- The certificate uses the same interval meaning as the existing analyzer proofs. -/
def Reported.interval (r : Reported) : interval.Range :=
  { Lower' := W64 r.lower, Upper' := W64 r.upper,
    HasLower' := r.hasLower, HasUpper' := r.hasUpper }

theorem reported_interval (r : Reported) (hl : Signed64 r.lower) (hu : Signed64 r.upper) (x : Int) :
    interval.contains r.interval x ↔ r.contains x := by
  have hlo : sint.Z (W64 r.lower) = r.lower := by obtain ⟨_, _⟩ := hl; word
  have hup : sint.Z (W64 r.upper) = r.upper := by obtain ⟨_, _⟩ := hu; word
  simp [interval.contains, Reported.interval, Reported.contains, hlo, hup]

theorem Certificate.interval_sound (c : Certificate) (x : Int) (hx : Signed64 x) :
    interval.contains c.report.interval (c.program.eval x) := by
  exact (reported_interval c.report c.bounds.1 c.bounds.2 _).mpr (c.sound x hx)

end interval
end Perennial.GstylePrograms
