module
public import Gstyle.Proof.Transfer.ConversionExecution
public import Gstyle.Proof.StateMap.Binding
public import Gstyle.Proof.Transfer.SymbolResolution
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard
def lengthCandidate [FfiSyntax] (s : QuerySnapshot) (ref : w32) : Value :=
  { (snapshotConstant s ref) with Length' := decide ((snapshotRecord s ref).Callee'.Status' = 1#8) && decide (pdg.tableText s.strings (snapshotRecord s ref).Callee'.Value' = go!"len") }

section proof
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics] [graphView.TypeAssumptions] [pdg.query.Assumptions]
variable [pdg.Graph.TypeAssumptions] [pdg.Node.TypeAssumptions] [pdg.Attributes.TypeAssumptions]
variable [pdg.AttributeSet.TypeAssumptions] [pdg.AttributeTables.TypeAssumptions]
variable [pdg.Producer.TypeAssumptions] [pdg.Function.TypeAssumptions] [pdg.Span.TypeAssumptions]
variable [pdg.CountFact.TypeAssumptions] [pdg.Conformance.TypeAssumptions] [pdg.Fact.TypeAssumptions]
variable [pdg.Position.TypeAssumptions] [pdg.CallForm.TypeAssumptions] [pdg.ExitKind.TypeAssumptions]
variable [pdg.ConformanceStatus.TypeAssumptions] [pdg.Category.TypeAssumptions]
variable [FuncUnfold pdg.listAt [pdg.Ref.ty] (pdg.listAt.impl pdg.Ref.ty)]
variable [FuncUnfold pdg.listAt [pdg.Access.ty] (pdg.listAt.impl pdg.Access.ty)]
variable [FuncUnfold pdg.listAt [pdg.Extension.ty] (pdg.listAt.impl pdg.Extension.ty)]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

variable [pdg.Access.TypeAssumptions] [pdg.Tables.TypeAssumptions] [pdg.Extension.TypeAssumptions]
variable [FuncUnfold directBindingRead [] directBindingRead.impl]
variable [FuncUnfold operatorText [] operatorText.impl]


variable [FuncUnfold extension [] extension.impl]

variable [Value.TypeAssumptions] [State.TypeAssumptions]

theorem wp_value_binding_dispatch (receiver : Loc) (ref symbol : w32) (state : Loc)
    (constant result : Value) (resources : IProp GF)
    (href : ref ≠ W32 0) (hsymbol : symbol ≠ W32 0)
    (hattrs :
      {{ isPkgInit (PROP := IProp GF) pkg ∗ resources }}
        (App (Val #(methods (go.PointerType graphView.ty) go!"attrs" #receiver)) (Val #ref))
      {{ (attrs : pdg.Attributes), RET #attrs; resources }})
    (hconstant :
      {{ isPkgInit (PROP := IProp GF) pkg ∗ resources }}
        (App (Val #(methods (go.PointerType graphView.ty) go!"constantValue" #receiver)) (Val #ref))
      {{ RET #constant; resources }})
    (hresolve :
      {{ isPkgInit (PROP := IProp GF) pkg ∗ resources }}
        (App (Val #(methods (go.PointerType graphView.ty) go!"symbol" #receiver)) (Val #ref))
      {{ RET #symbol; resources }})
    (hbinding :
      {{ isPkgInit (PROP := IProp GF) pkg ∗ resources }}
        (App (App (Val #(methods (go.PointerType graphView.ty) go!"binding" #receiver)) (Val #symbol)) (Val #state))
      {{ RET #result; resources }}) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ resources }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"value" #receiver)) (Val #ref)) (Val #state))
    {{ RET #result; resources }} := by
  wp_start as Hresources
  try wp_auto
  wp_if_destruct
  · exfalso; simp_all
  · wp_apply hattrs $$ [Hresources] with %attrs Hresources
    · iexact Hresources
    try wp_auto
    wp_apply hconstant $$ [Hresources] with Hresources
    · iexact Hresources
    try wp_auto
    wp_apply hresolve $$ [Hresources] with Hresources
    · iexact Hresources
    try wp_auto
    wp_if_destruct
    · exfalso; simp_all
    · wp_apply hbinding $$ [Hresources] with Hresources
      · iexact Hresources
      wp_end

theorem wp_binding_snapshot (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (key : w32) (kind : GoString) (changing : Bool)
    (hk : 0 < uint.Z key ∧ uint.Z key ≤ s.kinds.length)
    (hkind : s.kinds[(uint.Z key - 1).toNat]? = some kind)
    (hlen : s.unstable.length = s.kinds.length)
    (hunstable : s.unstable[(uint.Z key - 1).toNat]? = some changing) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"binding" #receiver)) (Val #key)) (Val #state))
    {{ RET #(bindingValue kind changing key m); querySnapshotOwn receiver s ∗ state ↦$ m }} := by
  wp_start_folded as ⟨#Hsnapshot, Hstate⟩
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, _, _, _, Hkinds, Hunstable, _, _, _, _, _, _, _, _, _, _, _⟩
  wp_apply wp_binding receiver s.view s.kinds s.unstable state m key kind changing hk hkind hlen hunstable DFrac.discard
    $$ [Hview Hkinds Hunstable Hstate] with ⟨Hview, Hkinds, Hunstable, Hstate⟩
  · iframe Hview Hkinds Hunstable Hstate
  wp_end
  all_goals iframe Hsnapshot Hstate

theorem wp_value_cached (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (ref key : w32) (kind : GoString) (changing : Bool)
    (hvalid : validSymbolRef s ref)
    (hf : 0 < uint.Z ref ∧ uint.Z ref ≤ s.fields.length)
    (hcache : s.fields[(uint.Z ref - 1).toNat]? = some key)
    (hk : 0 < uint.Z key ∧ uint.Z key ≤ s.kinds.length)
    (hkind : s.kinds[(uint.Z key - 1).toNat]? = some kind)
    (hlen : s.unstable.length = s.kinds.length)
    (hunstable : s.unstable[(uint.Z key - 1).toNat]? = some changing)
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46)
    (hparse : parseInt64Contract (GF := GF)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"value" #receiver)) (Val #ref)) (Val #state))
    {{ RET #(bindingValue kind changing key m); querySnapshotOwn receiver s ∗ state ↦$ m }} := by
  have hnonzero : ref ≠ W32 0 := by obtain ⟨_, _⟩ := hf; word
  have hkey : key ≠ W32 0 := by obtain ⟨_, _⟩ := hk; word
  apply wp_value_binding_dispatch receiver ref key state (snapshotConstant s ref)
    (bindingValue kind changing key m) iprop(querySnapshotOwn receiver s ∗ state ↦$ m) hnonzero hkey
  · wp_start_folded as ⟨Hsnapshot, Hstate⟩
    wp_apply wp_attrs_snapshot receiver s ref hvalid halloc $$ [Hsnapshot] with %attrs ⟨Hsnapshot, %hmetadata⟩
    · iexact Hsnapshot
    wp_end
  · wp_start_folded as ⟨Hsnapshot, Hstate⟩
    wp_apply wp_constantValue_snapshot receiver s ref hvalid halloc hparse $$ [Hsnapshot] with Hsnapshot
    · iexact Hsnapshot
    wp_end
  · wp_start_folded as ⟨Hsnapshot, Hstate⟩
    wp_apply wp_symbol_snapshot_cached receiver s ref key hf hcache hkey $$ [Hsnapshot] with Hsnapshot
    · iexact Hsnapshot
    wp_end
  · exact wp_binding_snapshot receiver s state m key kind changing hk hkind hlen hunstable

theorem wp_value_parentheses_dispatch (receiver : Loc) (s : QuerySnapshot) (ref child : w32)
    (state : Loc) (row : GoArray w32 2) (symbolFuel : Nat) (result : Value) (extra : IProp GF)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hvalid : validSymbolRef s ref)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hchildref : row.arr[0]? = some child)
    (hop : decodedFact s (snapshotRecord s ref).Operator' = go!"parentheses")
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46)
    (hparse : parseInt64Contract (GF := GF))
    (wf : SymbolSnapshotValid s)
    (hresolve : symbolResolve s symbolFuel ref = some (W32 0))
    (hchild :
      {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ extra) }}
        (App (App (Val #(methods (go.PointerType graphView.ty) go!"value" #receiver)) (Val #child)) (Val #state))
      {{ RET #result; querySnapshotOwn receiver s ∗ extra }}) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ extra) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"value" #receiver)) (Val #ref)) (Val #state))
    {{ RET #result; querySnapshotOwn receiver s ∗ extra }} := by
  wp_start as ⟨Hsnapshot, Hextra⟩
  try wp_auto
  wp_if_destruct
  · exfalso; obtain ⟨_, _⟩ := href; word
  · wp_apply wp_attrs_snapshot receiver s ref hvalid halloc $$ [Hsnapshot] with %attrs ⟨Hsnapshot, %hmetadata⟩
    · iexact Hsnapshot
    try wp_auto
    wp_apply wp_constantValue_snapshot receiver s ref hvalid halloc hparse $$ [Hsnapshot] with Hsnapshot
    · iexact Hsnapshot
    try wp_auto
    wp_apply wp_symbol_resolve receiver s symbolFuel ref (W32 0) wf hvalid hresolve $$ [Hsnapshot] with #Hsnapshot
    · iexact Hsnapshot
    ihave #Hctx := Hsnapshot
    iunfold querySnapshotOwn at Hctx
    icases Hctx with ⟨Hview, Hgraph, _, Hrows, _, _, _, _, _, _, _, _, _, _, _, _, _⟩
    try wp_auto
    wp_apply wp_operatorText_snapshot receiver s attrs.Operator' $$ [Hsnapshot] with Hsnapshot
    · iframe Hsnapshot
    rw [hmetadata.1, hop]
    try wp_auto
    wp_apply wp_operand_readonly receiver s.view s.operands ref (W64 0) row child href (by decide) hrow hchildref DFrac.discard DFrac.discard
      $$ [Hview Hrows] with ⟨Hview, Hrows⟩
    · iframe Hview Hrows
    try wp_auto
    wp_apply hchild $$ [Hsnapshot Hextra] with ⟨Hsnapshot, Hextra⟩
    · iframe Hsnapshot Hextra
    wp_end

theorem wp_value_conversion_constant (receiver : Loc) (s : QuerySnapshot) (ref : w32)
    (state : Loc) (symbolFuel : Nat) (extra : IProp GF)
    (href : ref ≠ W32 0)
    (hvalid : validSymbolRef s ref)
    (hop : decodedFact s (snapshotRecord s ref).Operator' = go!"conversion")
    (hinteger : (snapshotConstant s ref).Integer' = true)
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46)
    (hparse : parseInt64Contract (GF := GF))
    (wf : SymbolSnapshotValid s)
    (hresolve : symbolResolve s symbolFuel ref = some (W32 0))
 :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ extra) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"value" #receiver)) (Val #ref)) (Val #state))
    {{ RET #(snapshotConstant s ref); querySnapshotOwn receiver s ∗ extra }} := by
  wp_start as ⟨Hsnapshot, Hextra⟩
  try wp_auto
  wp_if_destruct
  · exfalso; simp_all
  · wp_apply wp_attrs_snapshot receiver s ref hvalid halloc $$ [Hsnapshot] with %attrs ⟨Hsnapshot, %hmetadata⟩
    · iexact Hsnapshot
    try wp_auto
    wp_apply wp_constantValue_snapshot receiver s ref hvalid halloc hparse $$ [Hsnapshot] with Hsnapshot
    · iexact Hsnapshot
    try wp_auto
    wp_apply wp_symbol_resolve receiver s symbolFuel ref (W32 0) wf hvalid hresolve $$ [Hsnapshot] with #Hsnapshot
    · iexact Hsnapshot
    ihave #Hctx := Hsnapshot
    iunfold querySnapshotOwn at Hctx
    icases Hctx with ⟨Hview, Hgraph, _, Hrows, _, _, _, _, _, _, _, _, _, _, _, _, _⟩
    try wp_auto
    wp_apply wp_operatorText_snapshot receiver s attrs.Operator' $$ [Hsnapshot] with Hsnapshot
    · iframe Hsnapshot
    rw [hmetadata.1, hop]
    try wp_auto
    wp_apply wp_conversionValue_integer receiver ref state (snapshotConstant s ref) hinteger
    wp_end

theorem wp_value_conversion_child (receiver : Loc) (s : QuerySnapshot) (ref child : w32)
    (state : Loc) (row : GoArray w32 2) (symbolFuel : Nat) (source : Value) (extra : IProp GF)
    (href : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hvalid : validSymbolRef s ref)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hchildref : row.arr[0]? = some child)
    (hop : decodedFact s (snapshotRecord s ref).Operator' = go!"conversion")
    (hinteger : (snapshotConstant s ref).Integer' = false)
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46)
    (hparse : parseInt64Contract (GF := GF))
    (wf : SymbolSnapshotValid s)
    (hresolve : symbolResolve s symbolFuel ref = some (W32 0))
    (hchild :
      {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ extra) }}
        (App (App (Val #(methods (go.PointerType graphView.ty) go!"value" #receiver)) (Val #child)) (Val #state))
      {{ RET #source; querySnapshotOwn receiver s ∗ extra }}) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ extra) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"value" #receiver)) (Val #ref)) (Val #state))
    {{ RET #(conversionResult (snapshotFact s ref (go!"type")) source); querySnapshotOwn receiver s ∗ extra }} := by
  wp_start as ⟨Hsnapshot, Hextra⟩
  try wp_auto
  wp_if_destruct
  · exfalso; obtain ⟨_, _⟩ := href; word
  · wp_apply wp_attrs_snapshot receiver s ref hvalid halloc $$ [Hsnapshot] with %attrs ⟨Hsnapshot, %hmetadata⟩
    · iexact Hsnapshot
    try wp_auto
    wp_apply wp_constantValue_snapshot receiver s ref hvalid halloc hparse $$ [Hsnapshot] with Hsnapshot
    · iexact Hsnapshot
    try wp_auto
    wp_apply wp_symbol_resolve receiver s symbolFuel ref (W32 0) wf hvalid hresolve $$ [Hsnapshot] with #Hsnapshot
    · iexact Hsnapshot
    ihave #Hctx := Hsnapshot
    iunfold querySnapshotOwn at Hctx
    icases Hctx with ⟨Hview, Hgraph, _, Hrows, _, _, _, _, _, _, _, _, _, _, _, _, _⟩
    try wp_auto
    wp_apply wp_operatorText_snapshot receiver s attrs.Operator' $$ [Hsnapshot] with Hsnapshot
    · iframe Hsnapshot
    rw [hmetadata.1, hop]
    try wp_auto
    wp_apply wp_conversionValue_child receiver s ref child state (snapshotConstant s ref) source row extra hinteger hvalid halloc href hrow hchildref hchild
      $$ [Hsnapshot Hextra] with ⟨Hsnapshot, Hextra⟩
    · iframe Hsnapshot Hextra
    wp_end

theorem wp_value_other_dispatch (receiver : Loc) (s : QuerySnapshot) (ref : w32)
    (state : Loc) (symbolFuel : Nat) (result : Value) (extra : IProp GF)
    (href : ref ≠ W32 0)
    (hvalid : validSymbolRef s ref)
    (hparent : decodedFact s (snapshotRecord s ref).Operator' ≠ go!"parentheses")
    (hconversion : decodedFact s (snapshotRecord s ref).Operator' ≠ go!"conversion")
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46)
    (hparse : parseInt64Contract (GF := GF))
    (wf : SymbolSnapshotValid s)
    (hresolve : symbolResolve s symbolFuel ref = some (W32 0))

    (hlength :
      {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ extra) }}
        (App (App (App (Val #(methods (go.PointerType graphView.ty) go!"lengthValue" #receiver)) (Val #ref)) (Val #state)) (Val #(lengthCandidate s ref)))
      {{ RET #result; querySnapshotOwn receiver s ∗ extra }}) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ extra) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"value" #receiver)) (Val #ref)) (Val #state))
    {{ RET #result; querySnapshotOwn receiver s ∗ extra }} := by
  wp_start as ⟨Hsnapshot, Hextra⟩
  try wp_auto
  wp_if_destruct
  · exfalso; simp_all
  · wp_apply wp_attrs_snapshot receiver s ref hvalid halloc $$ [Hsnapshot] with %attrs ⟨Hsnapshot, %hmetadata⟩
    · iexact Hsnapshot
    try wp_auto
    wp_apply wp_constantValue_snapshot receiver s ref hvalid halloc hparse $$ [Hsnapshot] with Hsnapshot
    · iexact Hsnapshot
    try wp_auto
    wp_apply wp_symbol_resolve receiver s symbolFuel ref (W32 0) wf hvalid hresolve $$ [Hsnapshot] with #Hsnapshot
    · iexact Hsnapshot
    ihave #Hctx := Hsnapshot
    iunfold querySnapshotOwn at Hctx
    icases Hctx with ⟨Hview, Hgraph, _, _, _, _, _, _, _, _, _, _, _, _, _, Htable, Hstrings⟩
    try wp_auto
    wp_apply wp_operatorText_snapshot receiver s attrs.Operator' $$ [Hsnapshot] with Hsnapshot
    · iframe Hsnapshot
    rw [hmetadata.1]
    try wp_auto
    simp only [hparent, _root_.decide_false]
    try wp_auto
    simp only [hconversion, _root_.decide_false]
    try wp_auto
    icases (pdg.Attributes.Attributes_access_load_Callee attrs_ptr attrs (DFrac.own 1)).access_strict $$ attrs with ⟨Hcallee, HattrsClose⟩
    try wp_auto
    rw [hmetadata.2]
    by_cases hs : (snapshotRecord s ref).Callee'.Status' = 1#8
    · try simp only [hs, _root_.decide_true]
      try wp_auto
      wp_apply pdg.wp_Text s.graph.Tables' s.table s.strings DFrac.discard (snapshotRecord s ref).Callee'.Value'
        $$ [Htable Hstrings] with ⟨Htable, Hstrings⟩
      · iframe Htable Hstrings
      try wp_auto
      simp only [lengthCandidate, hs, _root_.decide_true, Bool.true_and] at hlength
      wp_apply hlength $$ [Hsnapshot Hextra] with ⟨Hsnapshot, Hextra⟩
      · iframe Hsnapshot Hextra
      wp_end
    · have hs' : (snapshotRecord s ref).Callee'.Status' ≠ W8 1 := hs
      try simp only [hs', _root_.decide_false]
      try wp_auto
      simp only [lengthCandidate, hs, _root_.decide_false, Bool.false_and] at hlength
      wp_apply hlength $$ [Hsnapshot Hextra] with ⟨Hsnapshot, Hextra⟩
      · iframe Hsnapshot Hextra
      wp_end

theorem wp_value_other_no_length (receiver : Loc) (s : QuerySnapshot) (ref : w32)
    (state : Loc) (symbolFuel : Nat) (extra : IProp GF)
    (href : ref ≠ W32 0)
    (hvalid : validSymbolRef s ref)
    (hparent : decodedFact s (snapshotRecord s ref).Operator' ≠ go!"parentheses")
    (hconversion : decodedFact s (snapshotRecord s ref).Operator' ≠ go!"conversion")
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46)
    (hparse : parseInt64Contract (GF := GF))
    (wf : SymbolSnapshotValid s)
    (hresolve : symbolResolve s symbolFuel ref = some (W32 0))

    (hnotlength : (lengthCandidate s ref).Length' = false) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ extra) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"value" #receiver)) (Val #ref)) (Val #state))
    {{ RET #(lengthCandidate s ref); querySnapshotOwn receiver s ∗ extra }} := by
  apply wp_value_other_dispatch receiver s ref state symbolFuel (lengthCandidate s ref) extra href hvalid hparent hconversion halloc hparse wf hresolve
  wp_start_folded as Hresources
  wp_apply wp_lengthValue_not_length receiver ref state (lengthCandidate s ref) hnotlength
  wp_end

theorem wp_value_other_length_child (receiver : Loc) (s : QuerySnapshot) (ref child : w32)
    (state : Loc) (symbolFuel : Nat) (source : Value) (row : GoArray w32 2) (extra : IProp GF)
    (href : ref ≠ W32 0)
    (hvalid : validSymbolRef s ref)
    (hparent : decodedFact s (snapshotRecord s ref).Operator' ≠ go!"parentheses")
    (hconversion : decodedFact s (snapshotRecord s ref).Operator' ≠ go!"conversion")
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46)
    (hparse : parseInt64Contract (GF := GF))
    (wf : SymbolSnapshotValid s)
    (hresolve : symbolResolve s symbolFuel ref = some (W32 0))

    (hlength : (lengthCandidate s ref).Length' = true)
    (hoperand : 0 < uint.Z ref ∧ uint.Z ref ≤ s.operands.length)
    (hrow : s.operands[(uint.Z ref - 1).toNat]? = some row)
    (hchildref : row.arr[0]? = some child)
    (hchild :
      {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ extra) }}
        (App (App (Val #(methods (go.PointerType graphView.ty) go!"value" #receiver)) (Val #child)) (Val #state))
      {{ RET #source; querySnapshotOwn receiver s ∗ extra }}) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ extra) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"value" #receiver)) (Val #ref)) (Val #state))
    {{ RET #(lengthResult (lengthCandidate s ref) source); querySnapshotOwn receiver s ∗ extra }} := by
  apply wp_value_other_dispatch receiver s ref state symbolFuel (lengthResult (lengthCandidate s ref) source) extra href hvalid hparent hconversion halloc hparse wf hresolve
  exact wp_lengthValue_child receiver s ref child state (lengthCandidate s ref) source row extra hlength hoperand hrow hchildref hchild

theorem wp_value_bound (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (ref key : w32) (symbolFuel : Nat) (kind : GoString) (changing : Bool)
    (hvalid : validSymbolRef s ref)
    (href : ref ≠ W32 0)
    (wf : SymbolSnapshotValid s)
    (hresolve : symbolResolve s symbolFuel ref = some key)
    (hk : 0 < uint.Z key ∧ uint.Z key ≤ s.kinds.length)
    (hkind : s.kinds[(uint.Z key - 1).toNat]? = some kind)
    (hlen : s.unstable.length = s.kinds.length)
    (hunstable : s.unstable[(uint.Z key - 1).toNat]? = some changing)
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46)
    (hparse : parseInt64Contract (GF := GF)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"value" #receiver)) (Val #ref)) (Val #state))
    {{ RET #(bindingValue kind changing key m); querySnapshotOwn receiver s ∗ state ↦$ m }} := by
  have hkey : key ≠ W32 0 := by obtain ⟨_, _⟩ := hk; word
  apply wp_value_binding_dispatch receiver ref key state (snapshotConstant s ref)
    (bindingValue kind changing key m) iprop(querySnapshotOwn receiver s ∗ state ↦$ m) href hkey
  · wp_start_folded as ⟨Hsnapshot, Hstate⟩
    wp_apply wp_attrs_snapshot receiver s ref hvalid halloc $$ [Hsnapshot] with %attrs ⟨Hsnapshot, %hmetadata⟩
    · iexact Hsnapshot
    wp_end
  · wp_start_folded as ⟨Hsnapshot, Hstate⟩
    wp_apply wp_constantValue_snapshot receiver s ref hvalid halloc hparse $$ [Hsnapshot] with Hsnapshot
    · iexact Hsnapshot
    wp_end
  · wp_start_folded as ⟨Hsnapshot, Hstate⟩
    wp_apply wp_symbol_resolve receiver s symbolFuel ref key wf hvalid hresolve $$ [Hsnapshot] with Hsnapshot
    · iexact Hsnapshot
    wp_end
  · exact wp_binding_snapshot receiver s state m key kind changing hk hkind hlen hunstable

theorem wp_binding_snapshot_out_of_range (receiver : Loc) (s : QuerySnapshot) (state : Loc)
    (m : StateData) (key : w32) (hkey : s.kinds.length < uint.Z key) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"binding" #receiver)) (Val #key)) (Val #state))
    {{ RET #(zero_val Value); querySnapshotOwn receiver s ∗ state ↦$ m }} := by
  wp_start as ⟨#Hsnapshot, Hstate⟩
  ihave #Hctx := Hsnapshot
  iunfold querySnapshotOwn at Hctx
  icases Hctx with ⟨Hview, _, _, _, Hkinds, _, _, _, _, _, _, _, _, _, _, _, _⟩
  ihave %Hlen := ownSlice_len $$ Hkinds
  try wp_auto
  repeat (wp_if_destruct <;> try (exfalso; word))
  all_goals wp_end
  all_goals iframe Hsnapshot Hstate

theorem wp_value_bound_out_of_range (receiver : Loc) (s : QuerySnapshot) (state : Loc) (m : StateData)
    (ref key : w32) (symbolFuel : Nat)
    (hvalid : validSymbolRef s ref)
    (href : ref ≠ W32 0)
    (wf : SymbolSnapshotValid s)
    (hresolve : symbolResolve s symbolFuel ref = some key)
    (hout : s.kinds.length < uint.Z key)
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46)
    (hparse : parseInt64Contract (GF := GF)) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ (querySnapshotOwn receiver s ∗ state ↦$ m) }}
      (App (App (Val #(methods (go.PointerType graphView.ty) go!"value" #receiver)) (Val #ref)) (Val #state))
    {{ RET #(zero_val Value); querySnapshotOwn receiver s ∗ state ↦$ m }} := by
  have hkey : key ≠ W32 0 := by word
  apply wp_value_binding_dispatch receiver ref key state (snapshotConstant s ref)
    (zero_val Value) iprop(querySnapshotOwn receiver s ∗ state ↦$ m) href hkey
  · wp_start_folded as ⟨Hsnapshot, Hstate⟩
    wp_apply wp_attrs_snapshot receiver s ref hvalid halloc $$ [Hsnapshot] with %attrs ⟨Hsnapshot, %hmetadata⟩
    · iexact Hsnapshot
    wp_end
  · wp_start_folded as ⟨Hsnapshot, Hstate⟩
    wp_apply wp_constantValue_snapshot receiver s ref hvalid halloc hparse $$ [Hsnapshot] with Hsnapshot
    · iexact Hsnapshot
    wp_end
  · wp_start_folded as ⟨Hsnapshot, Hstate⟩
    wp_apply wp_symbol_resolve receiver s symbolFuel ref key wf hvalid hresolve $$ [Hsnapshot] with Hsnapshot
    · iexact Hsnapshot
    wp_end
  · exact wp_binding_snapshot_out_of_range receiver s state m key hout

end proof
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
