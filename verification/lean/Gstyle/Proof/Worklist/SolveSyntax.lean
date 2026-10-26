module
public import Gstyle.Code.github_com.rstyled.gstyle.domain.runtimeguard
@[expose] public section
noncomputable section
set_option maxHeartbeats 0
set_option maxRecDepth 100000
namespace Perennial
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [FfiSyntax] [GoGlobalContext]

/-- Select the generated cursor initializer and body together. Opaque library
    values are leaves. No replacement initializer or solver body is supplied. -/
def cursorSite : Expr → Option (Expr × Expr)
  | App (Rec _ (BNamed "$r0") (App (Rec _ (BNamed "cursor") body) _)) initial => some (initial, body)
  | Rec _ _ body => cursorSite body
  | App left right => (cursorSite left).or (cursorSite right)
  | If condition yes no => ((cursorSite condition).or (cursorSite yes)).or (cursorSite no)
  | _ => none

def solveBody : Expr :=
  match solve.impl with
  | RecV _ _ body => body
  | _ => Val #()

def solveCursorInitializer : Expr := ((cursorSite solveBody).getD (Val #(), Val #())).1

def solveCursorBody : Expr := ((cursorSite solveBody).getD (Val #(), Val #())).2

def solveLoop : Expr :=
  match solveCursorBody with
  | App (App _ (Rec _ _ loop)) _ => loop
  | _ => Val #()

theorem solve_cursor_found : (cursorSite solveBody).isSome = true := by rfl

theorem solve_cursor_initializer : solveCursorInitializer = Val #(W64 0) := by rfl

theorem solve_loop_shape : ∃ condition body post,
    solveLoop = App (App (App (Val doFor) condition) body) post := by
  exact ⟨_, _, _, rfl⟩

/-- Execute the actual initializer and actual cursor tail. Only the locations
    of the already allocated locals come from the surrounding proof boundary. -/
def solveCursorRun (receiver graphLocal cursor : Loc) : Expr :=
  subst "a" #receiver (subst "g" #graphLocal
    (subst "cursor" #cursor (Let "$r0" solveCursorInitializer solveCursorBody)))

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
