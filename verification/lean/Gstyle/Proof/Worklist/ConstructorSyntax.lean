module
public import Gstyle.Code.github_com.rstyled.gstyle.domain.runtimeguard
@[expose] public section
noncomputable section
set_option maxHeartbeats 0
set_option maxRecDepth 100000
namespace Perennial
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [FfiSyntax] [GoGlobalContext]

/-- The range expression includes its real local allocation and production body. -/
def constructorRangeSite (name : String) : Expr → Option Expr
  | e@(App (Rec _ (BNamed "$range") (App (Rec _ (BNamed binding) _) _)) _) =>
      if binding = name then some e else none
  | Rec _ _ body => constructorRangeSite name body
  | App left right => (constructorRangeSite name left).or (constructorRangeSite name right)
  | If condition yes no =>
      ((constructorRangeSite name condition).or (constructorRangeSite name yes)).or (constructorRangeSite name no)
  | _ => none

def constructorBody : Expr :=
  match newGraphView.impl with
  | RecV _ _ body => body
  | _ => Val #()

def constructorSymbolRange : Expr := (constructorRangeSite "symbol" constructorBody).getD (Val #())
def constructorEdgeRange : Expr := (constructorRangeSite "edge" constructorBody).getD (Val #())

theorem constructor_symbols_found : (constructorRangeSite "symbol" constructorBody).isSome = true := by rfl
theorem constructor_edges_found : (constructorRangeSite "edge" constructorBody).isSome = true := by rfl

def constructorSymbolsRawRun (graphLocal viewLocal : Loc) : Expr :=
  subst "g" #graphLocal (subst "view" #viewLocal constructorSymbolRange)

def constructorSymbolsRun (graphLocal viewLocal : Loc) : Expr :=
  App (Val exceptionDo) (constructorSymbolsRawRun graphLocal viewLocal)
def constructorEdgesRawRun (graphLocal viewLocal : Loc) : Expr :=
  subst "g" #graphLocal (subst "view" #viewLocal constructorEdgeRange)

def constructorEdgesRun (graphLocal viewLocal : Loc) : Expr :=
  App (Val exceptionDo) (constructorEdgesRawRun graphLocal viewLocal)

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
