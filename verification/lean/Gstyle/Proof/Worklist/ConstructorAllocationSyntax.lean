module
public import Gstyle.Proof.Worklist.ConstructorSyntax
@[expose] public section
noncomputable section
set_option maxRecDepth 100000
namespace Perennial
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [FfiSyntax] [GoGlobalContext]

/-- The complete lexical sequence that starts with the production symbol range. -/
def constructorSymbolSuffixStart : Expr → Bool
  | App (App (Val _) (Rec _ BAnon _))
      (App (Rec _ (BNamed "$range") (App (Rec _ (BNamed "symbol") _) _)) _) => true
  | _ => false

def constructorSymbolSuffixSite : Expr → Option Expr
  | Rec _ _ body => constructorSymbolSuffixSite body
  | e@(App left right) =>
      if constructorSymbolSuffixStart e then some e else
        (constructorSymbolSuffixSite left).or (constructorSymbolSuffixSite right)
  | If condition yes no =>
      ((constructorSymbolSuffixSite condition).or (constructorSymbolSuffixSite yes)).or
        (constructorSymbolSuffixSite no)
  | _ => none

def constructorSymbolSuffix : Expr :=
  (constructorSymbolSuffixSite constructorBody).getD (Val #())

/-- Replacing the selected boundary leaves every initializer and the return intact. -/
def constructorAllocationContext (tail : Expr) : Expr → Expr
  | Rec name argument body => Rec name argument (constructorAllocationContext tail body)
  | e@(App left right) =>
      if constructorSymbolSuffixStart e then tail else
        App (constructorAllocationContext tail left) (constructorAllocationContext tail right)
  | If condition yes no =>
      If (constructorAllocationContext tail condition) (constructorAllocationContext tail yes)
        (constructorAllocationContext tail no)
  | other => other

theorem constructor_symbol_suffix_found :
    (constructorSymbolSuffixSite constructorBody).isSome = true := by rfl

theorem constructor_allocation_context_exact :
    constructorAllocationContext constructorSymbolSuffix constructorBody = constructorBody := by rfl

/-- The remaining edge loop, field and tracking calls, and original return. -/
def constructorAfterSymbols : Expr :=
  match constructorSymbolSuffix with
  | App (App _ (Rec _ BAnon body)) _ => body
  | _ => Val #()

theorem constructor_symbol_suffix_exact :
    constructorSymbolSuffix =
      App (App (Val exceptionSeq) (Lam BAnon constructorAfterSymbols)) constructorSymbolRange := by rfl

def constructorSymbolSuffixRun (graphLocal viewLocal : Loc) : Expr :=
  App (Val exceptionDo)
    (subst "g" #graphLocal (subst "view" #viewLocal constructorSymbolSuffix))

def constructorAfterSymbolsRawRun (graphLocal viewLocal : Loc) : Expr :=
  subst "g" #graphLocal (subst "view" #viewLocal constructorAfterSymbols)

/-- Compose the raw symbol loop inside the original constructor exception boundary. -/
theorem constructor_symbol_suffix_run_exact (graphLocal viewLocal : Loc) :
    constructorSymbolSuffixRun graphLocal viewLocal =
      App (Val exceptionDo)
        (App (App (Val exceptionSeq) (Lam BAnon (constructorAfterSymbolsRawRun graphLocal viewLocal)))
          (constructorSymbolsRawRun graphLocal viewLocal)) := by rfl

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
