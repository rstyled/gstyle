module
public import Gstyle.Proof.Worklist.ConstructorTrackingSyntax
@[expose] public section
noncomputable section
set_option maxRecDepth 100000
namespace Perennial
namespace github_com.rstyled.gstyle.domain.runtimeguard
variable [FfiSyntax] [GoGlobalContext]

def trackingWritesHead : Expr → Bool
  | App (Rec _ (BNamed "write") _) _ => true
  | _ => false

/-- Select the actual write range after its writes slice has been evaluated. -/
def trackingWritesSite : Expr → Option Expr
  | Rec _ _ body => trackingWritesSite body
  | e@(App left right) =>
      if trackingWritesHead e then some e else (trackingWritesSite left).or (trackingWritesSite right)
  | If condition yes no =>
      ((trackingWritesSite condition).or (trackingWritesSite yes)).or (trackingWritesSite no)
  | _ => none

def trackingWritesBody : Expr := (trackingWritesSite trackingBody).getD (Val #())
theorem tracking_writes_found : (trackingWritesSite trackingBody).isSome = true := by rfl

/-- Elaboration extracts the selected loop through definitional equality.
The witness contains the source syntax, so unfolding it does not rerun the selector. -/
def trackingWritesLoop : Expr :=
  let selected : {loop : Expr // trackingWritesBody =
      Let "write" (App (Val (GoInstruction (GoAlloc pdg.Access.ty)))
        (App (Val (GoInstruction (GoZeroVal pdg.Access.ty))) (Val #()))) loop} :=
    ⟨_, rfl⟩
  selected.val

theorem tracking_writes_shape : trackingWritesBody =
    Let "write" (App (Val (GoInstruction (GoAlloc pdg.Access.ty)))
      (App (Val (GoInstruction (GoZeroVal pdg.Access.ty))) (Val #()))) trackingWritesLoop := by rfl

def trackingWritesRawRun (definitionsLocal refLocal : Loc) (writes : GoSlice) : Expr :=
  subst "definitions" #definitionsLocal (subst "ref" #refLocal (subst "$range" #writes trackingWritesBody))

end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
