module
public import Gstyle.Proof.Worklist.SolveSyntax
@[expose] public section
noncomputable section
set_option maxHeartbeats 0
set_option maxRecDepth 100000
namespace Perennial
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [FfiSyntax] [GoGlobalContext]

/-- Select the allocation and lexical continuation of the actual analysis local. -/
def analysisSite : Expr → Option Expr
  | e@(App (Rec _ (BNamed "$r0") (App (Rec _ (BNamed "a") _) _)) _) => some e
  | Rec _ _ body => analysisSite body
  | App left right => (analysisSite left).or (analysisSite right)
  | If condition yes no => ((analysisSite condition).or (analysisSite yes)).or (analysisSite no)
  | _ => none

def solveAnalysisBody : Expr := (analysisSite solveBody).getD (Val #())

/-- The cursor site is the hole in the source context, not a copied setup body. -/
def cursorExpression : Expr → Option Expr
  | e@(App (Rec _ (BNamed "$r0") (App (Rec _ (BNamed "cursor") _) _)) _) => some e
  | Rec _ _ body => cursorExpression body
  | App left right => (cursorExpression left).or (cursorExpression right)
  | If condition yes no => ((cursorExpression condition).or (cursorExpression yes)).or (cursorExpression no)
  | _ => none

def cursorHead : Expr → Bool
  | Rec _ (BNamed name) (App (Rec _ (BNamed inner) _) _) => name == "$r0" && inner == "cursor"
  | _ => false

def cursorStart : Expr → Bool
  | App head _ => cursorHead head
  | _ => false

/-- Follow lexical continuations; initializers and completed statements stay intact. -/
def setupContext (tail : Expr) : Expr → Expr
  | App (Rec name argument body) initial =>
      if cursorHead (Rec name argument body) then tail else
        App (Rec name argument (setupContext tail body)) initial
  | App (App sequence (Rec name argument body)) statement =>
      App (App sequence (Rec name argument (setupContext tail body)))
        (if cursorStart statement then tail else statement)
  | other => other

theorem setupContext_terminal (tail : Expr) (value : val) (argument : Expr) :
    setupContext tail (App (Val value) argument) = App (Val value) argument := by rfl

def solveSetupBody : Expr := setupContext (Val executeVal) solveAnalysisBody

theorem solve_analysis_found : (analysisSite solveBody).isSome = true := by rfl

theorem solve_setup_cursor_found : (cursorExpression solveAnalysisBody).isSome = true := by rfl

theorem solve_setup_stops_before_cursor : (cursorExpression solveSetupBody).isSome = false := by rfl

/-- Replacing the hole with its selected source expression recovers the whole body. -/
theorem solve_setup_context_exact :
    setupContext ((cursorExpression solveAnalysisBody).getD (Val #())) solveAnalysisBody =
      solveAnalysisBody := by rfl

/-- Execute the source allocation, entry-map store and enqueue, stopping at the
    cursor boundary. The original return remains in the source context. -/
def solveSetupRawRun (graphLocal viewLocal : Loc) : Expr :=
  subst "g" #graphLocal (subst "view" #viewLocal solveSetupBody)

def solveSetupRun (graphLocal viewLocal : Loc) : Expr :=
  App (Val exceptionDo) (solveSetupRawRun graphLocal viewLocal)

/-- The whole solve function with only its cursor continuation replaced. -/
def solveWithCursorTail (tail : Expr) : val :=
  match solve.impl with
  | RecV name argument (App label body) => RecV name argument (App label (setupContext tail body))
  | other => other

theorem solve_setup_function_exact :
    solveWithCursorTail ((cursorExpression solveAnalysisBody).getD (Val #())) = solve.impl := by rfl

theorem solve_setup_function_shape : ∃ name argument label body,
    solve.impl = RecV name argument (App label body) := by exact ⟨_, _, _, _, rfl⟩

def solveSetupEntry (graph : Loc) : Expr :=
  App (Val (solveWithCursorTail (Val executeVal))) (Val #graph)

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
