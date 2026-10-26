module
public import Gstyle.Proof.RuntimePackage
public import Gstyle.Proof.Worklist.ConstructorFact
@[expose] public section
noncomputable section
namespace Perennial
open Iris Iris.BI Iris.ProgramLogic Iris.Std
namespace github_com.rstyled.gstyle.domain.runtimeguard

/-- Scalar nodes can retain other metadata, including reads, writes, and operators. -/
def scalarFieldMetadata [FfiSyntax] (s : QuerySnapshot) : Prop :=
  ∀ a ∈ s.attributes,
    extensionValue s.strings (go!"field-path") (pdg.rowContents s.extensionRows a.Extensions') = go!"" ∧
    extensionValue s.strings (go!"dereference-paths") (pdg.rowContents s.extensionRows a.Extensions') = go!""

theorem snapshotFact_empty_of_extensions [FfiSyntax] (s : QuerySnapshot) (ref : w32) (name : GoString)
    (hmissing : ∀ a ∈ s.attributes,
      extensionValue s.strings name (pdg.rowContents s.extensionRows a.Extensions') = go!"") :
    snapshotFact s ref name = go!"" := by
  unfold snapshotFact
  split
  · rfl
  · split
    · rfl
    · split
      · rfl
      · split
        · rfl
        · rename_i a ha
          exact hmissing a (List.mem_of_getElem? ha)

/-- Only field-related components change during scalar field construction. -/
def constructorFieldsView [FfiSyntax] (view : graphView) (fields : GoSlice)
    (dereferences paths fieldRoots : GoMap) : graphView :=
  { view with
    fields' := fields
    dereferences' := dereferences
    paths' := paths
    fieldRoots' := fieldRoots }

section execution
variable [ext : FfiSyntax] [ffi : FfiModel] [FfiInterp ffi] [FfiSemantics ext ffi]
variable [go_gctx : GoGlobalContext] {hlc : HasLC} {GF : BundledGFunctors} [HeapGS hlc GF]
variable [go.Semantics]
variable [FuncUnfold decodeFact [] decodeFact.impl]
local notation "pkg" => pkg_id.github_com.rstyled.gstyle.domain.runtimeguard

/-- Empty metadata returns before the opaque JSON function and never reads its target. -/
theorem wp_decodeFact_empty (target : GoInterface) :
    {{ isPkgInit (PROP := IProp GF) pkg }}
      (App (App (Val (@! decodeFact)) (Val #(go!""))) (Val #target))
    {{ RET #false; True }} := by
  wp_start
  wp_auto
  wp_end

variable [graphView.TypeAssumptions] [pdg.query.Assumptions]
variable [pdg.Graph.TypeAssumptions] [pdg.Node.TypeAssumptions] [pdg.Attributes.TypeAssumptions]
variable [pdg.AttributeSet.TypeAssumptions] [pdg.AttributeTables.TypeAssumptions]
variable [pdg.Producer.TypeAssumptions] [pdg.Function.TypeAssumptions] [pdg.Span.TypeAssumptions]
variable [pdg.CountFact.TypeAssumptions] [pdg.Conformance.TypeAssumptions] [pdg.Fact.TypeAssumptions]
variable [pdg.Position.TypeAssumptions] [pdg.CallForm.TypeAssumptions] [pdg.ExitKind.TypeAssumptions]
variable [pdg.ConformanceStatus.TypeAssumptions] [pdg.Category.TypeAssumptions]
variable [pdg.Tables.TypeAssumptions] [pdg.Extension.TypeAssumptions]
variable [FuncUnfold extension [] extension.impl]
variable [FuncUnfold pdg.listAt [pdg.Ref.ty] (pdg.listAt.impl pdg.Ref.ty)]
variable [FuncUnfold pdg.listAt [pdg.Access.ty] (pdg.listAt.impl pdg.Access.ty)]
variable [FuncUnfold pdg.listAt [pdg.Extension.ty] (pdg.listAt.impl pdg.Extension.ty)]

/-- Execute field initialization for any bounded scalar graph, retaining all unrelated metadata. -/
theorem wp_fieldBindings_scalar (receiver : Loc) (view : graphView) (s : QuerySnapshot)
    (hgraph : view.graph' = s.view.graph') (hcount : s.nodes.length < 2^32)
    (halloc : ∀ a ∈ s.attributes, (pdg.rowContents s.referenceRows a.Outcomes').length ≤ 2^46)
    (hscalar : scalarFieldMetadata s) :
    {{ isPkgInit (PROP := IProp GF) pkg ∗ receiver ↦ view ∗ constructorGraphOwn s }}
      (App (Val #(methods (go.PointerType graphView.ty) go!"fieldBindings" #receiver)) (Val #()))
    {{ (fields : GoSlice) (dereferences paths roots : GoMap), RET #();
       receiver ↦ constructorFieldsView view fields dereferences paths roots ∗
       constructorGraphOwn s ∗ fields ↦* List.replicate s.nodes.length (W32 0) ∗
       dereferences ↦$ (∅ : GMap w32 GoSlice) ∗ paths ↦$ (∅ : GMap GoString w32) ∗
       roots ↦$ (∅ : GMap w32 w32) }} := by
  wp_start as ⟨Hview, #Hdata⟩
  ihave #Hctx := Hdata
  iunfold constructorGraphOwn at Hctx
  icases Hctx with ⟨Hgraph, Hnodes, _, _, _, _, _, _, _, _, _⟩
  ihave #HgraphLocal : (view.graph' ↦□ s.graph : IProp GF) $$ [Hgraph]
  · rw [hgraph]; iexact Hgraph
  ihave %Hlen := ownSlice_len $$ Hnodes
  wp_auto
  wp_apply wp_slice_make2 (V := w32) s.graph.Nodes'.len (by
    intro _
    simp only [go.typeSize_w32]
    word) $$ [] with %fields ⟨Hfields, Hcap⟩
  · ipureintro; exact Hlen.2
  rw [← Hlen.1, show zero_val w32 = W32 0 from rfl]
  try wp_auto
  wp_apply wp_map_make1 (K := w32) (V := GoSlice) pdg.Ref.ty (go.SliceType pdg.Ref.ty)
    with %dereferences Hdereferences
  try wp_auto
  wp_apply wp_map_make1 (K := GoString) (V := w32) go.string pdg.Ref.ty with %paths Hpaths
  try wp_auto
  wp_apply wp_map_make1 (K := w32) (V := w32) pdg.Ref.ty pdg.Ref.ty with %roots Hroots
  try wp_auto
  rename_i body_i_ptr
  irename : (body_i_ptr ↦ zero_val w64 : IProp GF) => body_i
  ihave HI : (∃ (iv bv : w64),
      "i" ∷ i_ptr ↦ iv ∗ "body_i" ∷ body_i_ptr ↦ bv ∗ "v" ∷ v_ptr ↦ receiver ∗
      "Hview" ∷ receiver ↦ constructorFieldsView view fields dereferences paths roots ∗
      "%Hbounds" ∷ ⌜0 ≤ sint.Z iv ∧ sint.Z iv ≤ s.nodes.length⌝ : IProp GF)
      $$ [i body_i v Hview]
  · iexists (W64 0), (W64 0)
    iframe
    ipureintro; constructor <;> word
  wp_for HI
  wp_if_destruct
  · have hi : sint.nat iv < s.nodes.length := by word
    list_elem s.nodes (sint.nat iv) as node
    rw [ite_eq_left (by constructor <;> word : 0 ≤ sint.Z iv ∧ sint.Z iv < sint.Z s.graph.Nodes'.len)]
    wp_apply wp_load_slice_index s.graph.Nodes' (sint.Z iv) s.nodes DFrac.discard node Hbounds.1
      $$ [Hnodes] with Hnodes
    · iframe Hnodes; ipureintro; exact Hnode_lookup
    try wp_auto
    have href : 0 < uint.Z (W32 (sint.Z (iv + W64 1))) ∧
        uint.Z (W32 (sint.Z (iv + W64 1))) ≤ s.nodes.length := by word
    wp_apply wp_graphView_fact_raw receiver (constructorFieldsView view fields dereferences paths roots)
      s (W32 (sint.Z (iv + W64 1))) (go!"field-path") (DFrac.own 1)
      hgraph (Or.inr href) halloc $$ [Hview Hdata] with ⟨Hview, _⟩
    · iframe Hview Hdata
    rw [snapshotFact_empty_of_extensions s _ (go!"field-path") (fun a ha => (hscalar a ha).1)]
    try wp_auto
    wp_apply wp_decodeFact_empty _
    try wp_auto
    wp_apply wp_graphView_fact_raw receiver (constructorFieldsView view fields dereferences paths roots)
      s (W32 (sint.Z (iv + W64 1))) (go!"dereference-paths") (DFrac.own 1)
      hgraph (Or.inr href) halloc $$ [Hview Hdata] with ⟨Hview, _⟩
    · iframe Hview Hdata
    rw [snapshotFact_empty_of_extensions s _ (go!"dereference-paths") (fun a ha => (hscalar a ha).2)]
    try wp_auto
    wp_apply wp_decodeFact_empty _
    try wp_auto
    wp_for_post
    iframe
    iexists (iv + W64 1), iv
    iframe
    ipureintro; constructor <;> word
  · try wp_auto
    iapply HΦ $$ %fields %dereferences %paths %roots
    iframe Hdata Hview Hfields Hdereferences Hpaths Hroots

end execution
end github_com.rstyled.gstyle.domain.runtimeguard
end Perennial
