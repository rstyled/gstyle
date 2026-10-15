module
public import Gstyle.Proof.Transfer.BranchModel
public import Perennial.Golang.Theory.Array
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [graphView.TypeAssumptions] [Value.TypeAssumptions] [State.TypeAssumptions] [pdg.query.Assumptions]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

theorem wp_operand (receiver : Loc) (view : graphView) (rows : List (GoArray w32 2))
    (ref : w32) (position : w64) (row : GoArray w32 2) (value : w32)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ rows.length)
    (hp : 0 ≤ sint.Z position ∧ sint.Z position < 2)
    (hrow : rows[(uint.Z ref - 1).toNat]? = some row)
    (hvalue : row.arr[(sint.Z position).toNat]? = some value) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (receiver ↦ view ∗ view.operands' ↦* rows) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"operand" #receiver)) (Val #ref)) (Val #position))
    {{ RET #value; receiver ↦ view ∗ view.operands' ↦* rows }} := by
  wp_start as ⟨Hview, Hrows⟩
  ihave %Hlen := ownSlice_len $$ Hrows
  try wp_auto
  repeat (wp_if_destruct <;> try (exfalso; obtain ⟨_, _⟩ := href; obtain ⟨_, _⟩ := hp; word))
  have Hi : sint.Z (W64 (uint.Z (ref - W32 1))) = uint.Z ref - 1 := by obtain ⟨_, _⟩ := href; word
  rw [Hi]
  rw [ite_eq_left (by obtain ⟨_, _⟩ := href; constructor <;> word : 0 ≤ uint.Z ref - 1 ∧ uint.Z ref - 1 < sint.Z view.operands'.len)]
  icases ownSlice_elem_acc (uint.Z ref - 1) row view.operands' (DFrac.own 1) rows (by omega) hrow $$ Hrows with ⟨Hrow, Hrows⟩
  try wp_auto
  rw [ite_eq_left hp]
  icases array_acc (sliceIndexRef (GoArray w32 2) (uint.Z ref - 1) view.operands') (sint.Z position) (DFrac.own 1) 2 row value hp.1 hvalue $$ Hrow with ⟨Hvalue, Hrow⟩
  try wp_auto
  ihave Hrow := Hrow $$ %value [Hvalue]
  · iexact Hvalue
  rw [list_set_lookup_self row.arr (sint.Z position).toNat value hvalue]
  ihave Hrows := Hrows $$ %row [Hrow]
  · iexact Hrow
  rw [list_set_lookup_self rows (uint.Z ref - 1).toNat row hrow]
  wp_end

theorem wp_operand_zero (receiver : Loc) (position : w64) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"operand" #receiver)) (Val #(W32 0))) (Val #position))
    {{ RET #(W32 0); True }} := by
  wp_start
  try wp_auto
  wp_end

theorem wp_operand_outside (receiver : Loc) (ref : w32) (position : w64)
    (hp : 2 ≤ sint.Z position) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"operand" #receiver)) (Val #ref)) (Val #position))
    {{ RET #(W32 0); True }} := by
  wp_start
  try wp_auto
  wp_if_destruct
  · try wp_auto
    wp_end
  · try wp_auto
    wp_if_destruct
    · wp_end
    · exfalso; word

def ownsBranch (m : Option StateData) (l : Loc) : IProp GF :=
  match m with
  | none => iprop(⌜l = map.nil⌝)
  | some m => iprop(l ↦$ m)

def branchResources (receiver : Loc) (view : graphView) (rows : List (GoArray w32 2))
    (kinds : List GoString) (unstable : List Bool) (extra : IProp GF) : IProp GF :=
  iprop(receiver ↦ view ∗ view.operands' ↦* rows ∗ view.kinds' ↦* kinds ∗ view.unstable' ↦* unstable ∗ extra)

def branchChildContract (receiver : Loc) (view : graphView) (rows : List (GoArray w32 2))
    (kinds : List GoString) (unstable : List Bool) (kind : w32 → GoString) (changing : w32 → Bool)
    (extra : IProp GF) (child : w32) (truth : Bool) (step : StateData → Option StateData) : Prop :=
  ∀ (state : Loc) (m : StateData), validBranchState kinds unstable kind changing m →
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
      (branchResources receiver view rows kinds unstable extra ∗ state ↦$ m) }}
      (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"assume" #receiver)) (Val #child)) (Val #truth)) (Val #state))
    {{ (out : Loc), RET #out;
      branchResources receiver view rows kinds unstable extra ∗ ownsBranch (step m) out ∗
      ⌜validBranchOption kinds unstable kind changing (step m)⌝ }}

variable [FuncUnfold copyState [] copyState.impl]
variable [FuncUnfold integerConstraint [] integerConstraint.impl] [interval.Assumptions]

theorem wp_logical (receiver : Loc) (view : graphView) (rows : List (GoArray w32 2))
    (kinds : List GoString) (unstable : List Bool) (kind : w32 → GoString) (changing : w32 → Bool)
    (extra : IProp GF) (state : Loc) (m : StateData) (ref : w32) (op : GoString) (truth : Bool)
    (row : GoArray w32 2) (leftRef rightRef : w32) (left right : StateData → Option StateData)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ rows.length)
    (hrow : rows[(uint.Z ref - 1).toNat]? = some row)
    (hl : row.arr[0]? = some leftRef) (hr : row.arr[1]? = some rightRef)
    (hlen : unstable.length = kinds.length)
    (hm : validBranchState kinds unstable kind changing m)
    (hleft : branchChildContract receiver view rows kinds unstable kind changing extra leftRef truth left)
    (hright : branchChildContract receiver view rows kinds unstable kind changing extra rightRef truth right) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
      (branchResources receiver view rows kinds unstable extra ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"logical" #receiver)) (Val #ref)) (Val #op)) (Val #truth)) (Val #state))
    {{ (out : Loc), RET #out; ∃ (result : Option StateData),
      branchResources receiver view rows kinds unstable extra ∗ ownsBranch result out ∗
      (if sequentialBranch op truth then True else state ↦$ m) ∗
      ⌜branchResult kind changing op truth left right m result ∧ validBranchOption kinds unstable kind changing result⌝ }} := by
  wp_start as ⟨Hctx, Hstate⟩
  iunfold branchResources at Hctx
  icases Hctx with ⟨Hview, Hrows, Hkinds, Hunstable, Hextra⟩
  try wp_auto
  cases hs : sequentialBranch op truth
  · have hcond := hs
    unfold sequentialBranch at hcond
    rw [hcond]
    try wp_auto
    wp_apply wp_operand receiver view rows ref (W64 0) row leftRef href (by decide) hrow hl $$ [Hview Hrows] with ⟨Hview, Hrows⟩
    · iframe
    wp_apply wp_copyState state m $$ [Hstate] with %leftCopy ⟨Hstate, Hcopy⟩
    · iframe
    wp_apply (hleft leftCopy m hm) $$ [Hview Hrows Hkinds Hunstable Hextra Hcopy] with %leftLoc ⟨Hctx, Hleft, %HleftValid⟩
    · unfold branchResources; iframe
    iunfold branchResources at Hctx
    icases Hctx with ⟨Hview, Hrows, Hkinds, Hunstable, Hextra⟩
    wp_apply wp_operand receiver view rows ref (W64 1) row rightRef href (by decide) hrow hr $$ [Hview Hrows] with ⟨Hview, Hrows⟩
    · iframe
    wp_apply wp_copyState state m $$ [Hstate] with %rightCopy ⟨Hstate, Hcopy⟩
    · iframe
    wp_apply (hright rightCopy m hm) $$ [Hview Hrows Hkinds Hunstable Hextra Hcopy] with %rightLoc ⟨Hctx, Hright, %HrightValid⟩
    · unfold branchResources; iframe
    cases hleftResult : left m with
    | none =>
      iunfold ownsBranch at Hleft
      icases Hleft with %HleftNil
      subst leftLoc
      cases hrightResult : right m with
      | none =>
        iunfold ownsBranch at Hright
        icases Hright with %HrightNil
        subst rightLoc
        try wp_auto
        wp_end
        iexists none
        try simp [hs]
        unfold ownsBranch
        iframe
        ipureintro
        simp [branchResult, hs, branchJoin, hleftResult, hrightResult, validBranchOption]
      | some n =>
        iunfold ownsBranch at Hright
        ihave %HrightNotNil := ownMap_not_nil rightLoc n (DFrac.own 1) $$ Hright
        try wp_auto
        wp_if_destruct
        · contradiction
        · wp_apply wp_join_left_nil receiver rightLoc n $$ [Hright] with %out ⟨Hright, Hout⟩
          · iframe
          wp_end
          iexists (some n)
          try simp [hs]
          unfold ownsBranch
          iframe
          ipureintro
          simpa [branchResult, hs, branchJoin, hleftResult, hrightResult, validBranchOption] using HrightValid
    | some p =>
      iunfold ownsBranch at Hleft
      ihave %HleftNotNil := ownMap_not_nil leftLoc p (DFrac.own 1) $$ Hleft
      simp [HleftNotNil]
      try wp_auto
      cases hrightResult : right m with
      | none =>
        iunfold ownsBranch at Hright
        icases Hright with %HrightNil
        subst rightLoc
        wp_apply wp_join_right_nil receiver leftLoc p $$ [Hleft] with %out ⟨Hleft, Hout⟩
        · iframe
        wp_end
        iexists (some p)
        try simp [hs]
        unfold ownsBranch
        iframe
        ipureintro
        simpa [branchResult, hs, branchJoin, hleftResult, hrightResult, validBranchOption] using HleftValid
      | some n =>
        iunfold ownsBranch at Hright
        iunfold branchResources at Hctx
        icases Hctx with ⟨Hview, Hrows, Hkinds, Hunstable, Hextra⟩
        have hp : validBranchState kinds unstable kind changing p := by simpa [hleftResult, validBranchOption] using HleftValid
        have hn : validBranchState kinds unstable kind changing n := by simpa [hrightResult, validBranchOption] using HrightValid
        have hmeta : ∀ key, (p !! key).isSome ∨ (n !! key).isSome →
            (0 < uint.Z key ∧ uint.Z key ≤ kinds.length) ∧
            kinds[(uint.Z key - 1).toNat]? = some (kind key) ∧
            unstable[(uint.Z key - 1).toNat]? = some (changing key) := by
          intro key hk
          rcases hk with hk | hk
          · exact hp key hk
          · exact hn key hk
        wp_apply wp_join receiver view kinds unstable kind changing leftLoc rightLoc p n hlen hmeta $$ [Hview Hkinds Hunstable Hleft Hright]
          with %out ⟨%om, Hview, Hkinds, Hunstable, Hleft, Hright, Hout, %Hlookup⟩
        · iframe
        wp_end
        iexists (some om)
        try simp [hs]
        unfold ownsBranch branchResources
        iframe
        ipureintro
        constructor
        · simpa [branchResult, hs, branchJoin, hleftResult, hrightResult] using Hlookup
        · exact joined_branch_valid kinds unstable kind changing p n om hp hn Hlookup
  · have hcond := hs
    unfold sequentialBranch at hcond
    rw [hcond]
    try wp_auto
    wp_apply wp_operand receiver view rows ref (W64 0) row leftRef href (by decide) hrow hl $$ [Hview Hrows] with ⟨Hview, Hrows⟩
    · iframe
    wp_apply (hleft state m hm) $$ [Hview Hrows Hkinds Hunstable Hextra Hstate] with %leftLoc ⟨Hctx, Hleft, %HleftValid⟩
    · unfold branchResources; iframe
    cases hleftResult : left m with
    | none =>
      iunfold ownsBranch at Hleft
      icases Hleft with %HleftNil
      subst leftLoc
      try wp_auto
      wp_end
      iexists none
      simp [hs]
      unfold ownsBranch
      iframe
      ipureintro
      simp [branchResult, hs, hleftResult, validBranchOption]
    | some n =>
      iunfold ownsBranch at Hleft
      ihave %HleftNotNil := ownMap_not_nil leftLoc n (DFrac.own 1) $$ Hleft
      wp_if_destruct
      · contradiction
      · iunfold branchResources at Hctx
        icases Hctx with ⟨Hview, Hrows, Hkinds, Hunstable, Hextra⟩
        wp_apply wp_operand receiver view rows ref (W64 1) row rightRef href (by decide) hrow hr $$ [Hview Hrows] with ⟨Hview, Hrows⟩
        · iframe
        wp_apply (hright leftLoc n (by simpa [hleftResult, validBranchOption] using HleftValid)) $$ [Hview Hrows Hkinds Hunstable Hextra Hleft] with %rightLoc ⟨Hctx, Hright, %HrightValid⟩
        · unfold branchResources; iframe
        wp_end
        iexists (right n)
        try simp [hs]
        iframe
        ipureintro
        simpa [branchResult, hs, hleftResult] using HrightValid

theorem wp_logical_sound (receiver : Loc) (view : graphView) (rows : List (GoArray w32 2))
    (kinds : List GoString) (unstable : List Bool) (kind : w32 → GoString) (changing : w32 → Bool)
    (extra : IProp GF) (state : Loc) (m : StateData) (ref : w32) (op : GoString) (truth : Bool)
    (row : GoArray w32 2) (leftRef rightRef : w32) (left right : StateData → Option StateData)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ rows.length)
    (hrow : rows[(uint.Z ref - 1).toNat]? = some row)
    (hl : row.arr[0]? = some leftRef) (hr : row.arr[1]? = some rightRef)
    (hlen : unstable.length = kinds.length)
    (hm : validBranchState kinds unstable kind changing m)
    (hleft : branchChildContract receiver view rows kinds unstable kind changing extra leftRef truth left)
    (hright : branchChildContract receiver view rows kinds unstable kind changing extra rightRef truth right)
    (env : w32 → Int) (lhs rhs : Bool)
    (hop : op = go!"&&" ∨ op = go!"||")
    (htruth : (if op = go!"&&" then lhs && rhs else lhs || rhs) = truth)
    (hin : stateDenotes kind changing m env)
    (hleftSound : ∀ m, stateDenotes kind changing m env → lhs = truth → branchDenotes kind changing (left m) env)
    (hrightSound : ∀ m, stateDenotes kind changing m env → rhs = truth → branchDenotes kind changing (right m) env) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ isPkgInit (PROP := IProp GF) pkg_id.github_com.rstyled.gstyle.domain.runtimeguard.interval ∗
      (branchResources receiver view rows kinds unstable extra ∗ state ↦$ m) }}
      (App (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"logical" #receiver)) (Val #ref)) (Val #op)) (Val #truth)) (Val #state))
    {{ (out : Loc), RET #out; ∃ (result : Option StateData),
      branchResources receiver view rows kinds unstable extra ∗ ownsBranch result out ∗
      (if sequentialBranch op truth then True else state ↦$ m) ∗
      ⌜branchDenotes kind changing result env ∧ validBranchOption kinds unstable kind changing result⌝ }} := by
  wp_start_folded as Hpre
  wp_apply wp_logical receiver view rows kinds unstable kind changing extra state m ref op truth row leftRef rightRef left right href hrow hl hr hlen hm hleft hright $$ [Hpre]
    with %out ⟨%result, Hctx, Hresult, Hsource, %Hspec⟩
  · iexact Hpre
  wp_end
  iexists result
  iframe
  ipureintro
  exact ⟨branch_result_sound kind changing op truth left right m result env lhs rhs hop htruth hin hleftSound hrightSound Hspec.1, Hspec.2⟩

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
