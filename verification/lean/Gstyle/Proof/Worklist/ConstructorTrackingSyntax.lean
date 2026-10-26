module
public import Gstyle.Code.github_com.rstyled.gstyle.domain.runtimeguard
@[expose] public section
noncomputable section
set_option maxRecDepth 100000
namespace Perennial
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [FfiSyntax] [GoGlobalContext]

def trackingBody : Expr :=
  match graphView.trackBindings.impl with
  | RecV _ _ body => body
  | _ => Val #()

/-- Select the real queue cursor allocation and its complete loop. -/
def trackingCursorSite : Expr → Option Expr
  | e@(App (Rec _ (BNamed "$r0") (App (Rec _ (BNamed "cursor") _) _)) _) => some e
  | Rec _ _ body => trackingCursorSite body
  | App left right => (trackingCursorSite left).or (trackingCursorSite right)
  | If condition yes no =>
      ((trackingCursorSite condition).or (trackingCursorSite yes)).or (trackingCursorSite no)
  | _ => none

def trackingCursorBody : Expr := (trackingCursorSite trackingBody).getD (Val #())

theorem tracking_cursor_found : (trackingCursorSite trackingBody).isSome = true := by rfl

/-- This expression is selected from the translation, including its cursor local. -/
def trackingCursorRun (queueLocal : Loc) : Expr :=
  App (Val exceptionDo) (subst "queue" #queueLocal trackingCursorBody)

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
