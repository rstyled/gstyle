#!/usr/bin/env python3
"""Check source binding, Lean proofs, axioms, and deliberate Go mutations."""

from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass, field, replace
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile

PROJECT = Path(__file__).resolve().parent
ROOT = PROJECT.parent.parent
REPORT = ROOT / "dist/verification/function-length"
PACKAGE = "github_com/rstyled/gstyle/domain/quality/linecount.lean"
GO_PACKAGE = "./domain/quality/linecount"
SOURCE = ROOT / "domain/quality/linecount/counter.go"
PREFIX = "Perennial.github_com.rstyled.gstyle.domain.quality.linecount."
THEOREMS = {PREFIX + name for name in (
    "freshLines_mem", "freshLines_nodup", "freshLines_length",
    "freshLines_length_int", "freshLines_extend", "wp_accepts", "wp_Add", "wp_Reject", "wp_Exceeds",
)}
ALLOWED_AXIOMS = {"propext", "Classical.choice", "Quot.sound"}
PROOF_FAILURE_PREFIXES = (
    "iapply: cannot apply ", "wp_apply: cannot apply ", "wp_apply: the goal ",
    "iexists: cannot turn ",
    "Tactic `rewrite` failed: Did not find an occurrence of the pattern",
    "Tactic `rfl` failed: The left-hand side",
)
MUTATIONS = {
    "omit-last-line": ("last - first + 1", "last - first"),
    "count-overlap-twice": ("first++", "first += 0"),
    "accept-zero-line": ("first >= 1", "first >= 0"),
    "inclusive-limit": ("actual > limit", "actual >= limit"),
}


def proof_rejected(output, proof_file):
    diagnostic = re.compile(r"error: " + re.escape(proof_file) + r":\d+:\d+: (.*)")
    found = False
    for line in output.splitlines():
        if not line.startswith("error:") or line == "error: build failed":
            continue
        match = diagnostic.fullmatch(line)
        if match is None:
            return False
        message = match.group(1)
        if not (message == "unsolved goals" or message.startswith(PROOF_FAILURE_PREFIXES) or
                message == "wp_if_destruct: no `decide` or Boolean variable in the expression"):
            return False
        found = True
    return found


def run(command, cwd, log, *, expect_failure=False, report=None, proof_file=None):
    report = report or REPORT
    result = subprocess.run(command, cwd=cwd, text=True, capture_output=True, timeout=1800)
    output = result.stdout + result.stderr
    (report / log).write_text(output)
    if expect_failure:
        if result.returncode == 0 or proof_file is None or not proof_rejected(output, proof_file):
            raise RuntimeError(f"mutation does not fail its proof: {log}")
    elif result.returncode != 0:
        raise RuntimeError(f"command failed; see {report / log}")
    return output


def check_axioms(output, theorems=None):
    theorems = theorems or THEOREMS
    entries = re.findall(r"'([^']+)' depends on axioms:\s*\[([^]]*)\]", output)
    empty = re.findall(r"'([^']+)' does not depend on any axioms", output)
    names = {name for name, _ in entries} | set(empty)
    if names != theorems:
        raise RuntimeError("axiom audit is missing or adds theorem entries")
    for name, raw in entries:
        axioms = {item.strip() for item in raw.split(",") if item.strip()}
        if not axioms <= ALLOWED_AXIOMS:
            raise RuntimeError(f"unexpected axioms for {name}: {axioms - ALLOWED_AXIOMS}")


@dataclass(frozen=True)
class Profile:
    report: Path
    package: str
    go_package: str
    source: Path
    sources: tuple
    theorems: set
    mutations: dict
    audit: str
    proof_file: str
    verified: str
    unverified: tuple
    generated: tuple = ("Code", "GeneratedProof")
    bindings: tuple = ()
    mutation_modules: dict = field(default_factory=dict)


def length_profile():
    return Profile(REPORT, PACKAGE, GO_PACKAGE, SOURCE, (SOURCE,), THEOREMS,
                   MUTATIONS, "Audit.lean", "Gstyle/Proof/Length.lean", "ordered token-span accumulator and strict threshold",
                   ("Go scanner", "token-to-span extraction", "configuration and diagnostics"))


def pdg_profile():
    prefix = "Perennial.github_com.rstyled.gstyle.domain.pdg.query."
    source = ROOT / "domain/pdg/query/query.go"
    names = ("ReferenceIndex", "ReferenceReject", "EvidenceAt", "EvidenceAbsent",
             "EvidenceStatus", "EvidenceUnknown", "CoverageSupported", "DependencySupported",
             "DependencyAdmitted", "DependencyUnavailable", "BooleanValue", "KnownFalse", "At", "AtAbsent")
    return Profile(
        ROOT / "dist/verification/pdg-queries", "github_com/rstyled/gstyle/domain/pdg/query.lean",
        "./domain/pdg/query", source, (source, source.with_name("records.go")),
        {prefix + "wp_" + name for name in names},
        {
            "wrong-generic-slot": ("return table[index], true", "return table[index+1], true"),
            "forge-generic-presence": ("return zero, false", "return zero, true"),
            "drop-generic-presence": ("return table[index], true", "return table[index], false"),
            "forge-unknown-boolean": ("return false, false", "return false, true"),
            "accept-zero-reference": ("ref == 0", "false"),
            "wrong-slot": ("return int(ref) - 1, true", "return int(ref), true"),
            "reject-last-slot": ("uint64(ref) > uint64(count)", "uint64(ref) >= uint64(count)"),
            "forge-missing-status": ("return Unknown", "return Supported"),
            "admit-inapplicable-edge": ("return status == Supported || status == Known\n", "return status == Supported || status == Known || status == NotApplicable\n"),
        },
        "AuditQueries.lean", "Gstyle/Proof/Queries.lean", "PDG reference/evidence/Boolean queries and composed admission gates",
        ("Go-to-PDG extraction", "graph snapshot provenance", "name-to-capability lookup",
         "whole-graph traversal", "pattern findings and rewrite semantics"),
    )


def output_mutation_modules():
    groups = {
        "Classify": "cross-classifier-callee cross-classifier-writer invert-blank-target ignore-consumption-form swap-consumption-counts forge-blank-target wrong-function-package miss-fprintln wrong-writer-package cross-writer-identity unrecognized-function memory-writer wrong-result-count handled-error wrong-value-count missed-standalone",
        "Describe": "wrong-finding-path wrong-finding-owner wrong-finding-name wrong-finding-line wrong-finding-end-line wrong-finding-kind wrong-finding-score drop-callee-name",
        "Report": "skip-last-report duplicate-report",
        "Suppression": "wrong-suppression-name inverted-suppression-match skip-last-suppression",
        "Find": "bypass-suppression-lookup",
        "Select": "ignore-suppression invert-selection skip-last-site duplicate-site",
    }
    return {name: f"Gstyle/Proof/Output/{module}.lean"
            for module, names in groups.items() for name in names.split()}


def output_profile():
    source = ROOT / "domain/patterns/outputcheck/output.go"
    prefix = "Perennial.github_com.rstyled.gstyle.domain.patterns.outputcheck."
    return Profile(
        ROOT / "dist/verification/unchecked-output", "github_com/rstyled/gstyle/domain/patterns/outputcheck.lean",
        "./domain/patterns/outputcheck", source, (source,),
        {prefix + name for name in ("wp_Discards", "wp_Select", "wp_Formatted", "wp_Memory", "wp_hasSuppression", "wp_Find", "wp_describe", "wp_Report", "wp_Classify")},
        {
            "cross-classifier-callee": ("Formatted(callee.Package, callee.Name)", "Formatted(writer.Package, writer.Name)"),
            "cross-classifier-writer": ("Memory(writer.Package, writer.Name)", "Memory(callee.Package, callee.Name)"),
            "invert-blank-target": ('use.ErrorName == "_"', 'use.ErrorName != "_"'),
            "ignore-consumption-form": ("Standalone: use.Standalone", "Standalone: false"),
            "swap-consumption-counts": ("Values: use.Values, Results: use.Results", "Values: use.Results, Results: use.Values"),
            "forge-blank-target": ('ErrorBlank: use.ErrorName == "_"', "ErrorBlank: true"),
            "wrong-finding-path": ("Path: function.Path", "Path: function.Name"),
            "wrong-finding-owner": ("ID: function.ID", "ID: site.FuncID"),
            "wrong-finding-name": ("Name: function.Name", "Name: function.ID"),
            "wrong-finding-line": ("Path: function.Path, Line: site.Line", "Path: function.Path, Line: 0"),
            "wrong-finding-end-line": ("EndLine: site.Line", "EndLine: 0"),
            "wrong-finding-kind": ('Kind: "unchecked_output"', 'Kind: "nilerr"'),
            "wrong-finding-score": ("ScoreMilli: 800", "ScoreMilli: 801"),
            "drop-callee-name": (" + site.Callee", ""),
            "skip-last-report": ("i < len(selected)", "i < len(selected)-1"),
            "duplicate-report": ("append(out, describe(function, selected[i]))", "append(out, describe(function, selected[i]), describe(function, selected[i]))"),
            "wrong-suppression-name": ('kinds[i] == "unchecked_output"', 'kinds[i] == "other"'),
            "inverted-suppression-match": ('kinds[i] == "unchecked_output"', 'kinds[i] != "unchecked_output"'),
            "skip-last-suppression": ("i < len(kinds)", "i < len(kinds)-1"),
            "bypass-suppression-lookup": ("hasSuppression(suppressions)", "false"),
            "wrong-function-package": ('pkg == "fmt"', 'pkg != "fmt"'),
            "miss-fprintln": (' || name == "Fprintln"', ''),
            "wrong-writer-package": ('pkg == "bytes"', 'pkg != "bytes"'),
            "cross-writer-identity": ('name == "Buffer"', 'name == "Builder"'),
            "unrecognized-function": ("!input.Formatted || input.Memory", "input.Memory"),
            "memory-writer": ("!input.Formatted || input.Memory", "!input.Formatted"),
            "wrong-result-count": ("input.Results == 2", "input.Results == 1"),
            "handled-error": (" && input.ErrorBlank", ""),
            "wrong-value-count": ("input.Values == 1", "input.Values == 2"),
            "missed-standalone": ("if input.Standalone {\n\t\treturn true", "if input.Standalone {\n\t\treturn false"),
            "ignore-suppression": ("if suppressed {", "if false {"),
            "invert-selection": ("if site.DiscardedOutput {", "if !site.DiscardedOutput {"),
            "skip-last-site": ("i < len(sites)", "i < len(sites)-1"),
            "duplicate-site": ("append(out, site)", "append(out, site, site)"),
        },
        "AuditOutput.lean", "Gstyle/Proof/Output.lean", "unchecked-output composed classification, suppression, selection and advisory payloads",
        ("AST/type fact extraction", "corpus traversal, input/Candidate projection and presentation", "Go compiler and stdlib behavior"),
        mutation_modules=output_mutation_modules(),
    )


def interval_mutation_modules():
    groups = {
        "Constrain": "dispatch-lower-as-upper dispatch-strict-lower-as-upper dispatch-strict-upper-as-lower dispatch-upper-as-lower dispatch-equality-misses-upper dispatch-default-narrows",
        "Strict": "strict-lower-overflow strict-upper-overflow strict-lower-includes-boundary strict-upper-includes-boundary strict-lower-wrong-transfer strict-upper-wrong-transfer",
        "Bounds": "lower-forgets-knowledge lower-trusts-unknown lower-wrong-direction lower-excludes-boundary upper-forgets-knowledge upper-trusts-unknown upper-wrong-direction upper-excludes-boundary",
        "Advisory": "wrong-advisory-path wrong-advisory-owner wrong-advisory-line wrong-advisory-kind wrong-advisory-score drop-index-name",
        "Select": "omit-deduplication deduplicate-by-node forget-seen-binding ignore-access-risk mark-safe-binding skip-last-access duplicate-access forge-source-node",
        "Join": "narrow-lower-bound narrow-upper-bound forge-known-lower forge-known-upper stale-unknown-lower stale-unknown-upper",
        "Risk": "warn-without-index warn-without-upper-check treat-zero-as-negative trust-unknown-lower",
    }
    return {name: f"Gstyle/Proof/Interval/{module}.lean"
            for module, names in groups.items() for name in names.split()}


def interval_profile():
    source = ROOT / "domain/runtimeguard/interval/interval.go"
    prefix = "Perennial.github_com.rstyled.gstyle.domain.runtimeguard.interval."
    return Profile(
        ROOT / "dist/verification/negative-index", "github_com/rstyled/gstyle/domain/runtimeguard/interval.lean",
        "./domain/runtimeguard/interval", source, (source,),
        {prefix + name for name in ("join_sound", "negative_detected", "wp_Join", "wp_NegativeIndex", "wp_JoinSound", "wp_NegativeDetected", "selected_avoids", "selected_risky", "selected_unique", "selected_sublist", "selected_keys", "wp_Select", "wp_SelectProperties", "advisory_line", "wp_Describe", "selected_negative", "wp_SelectNegative", "lower_intersection", "upper_intersection", "wp_Lower", "wp_Upper", "wp_LowerIntersection", "wp_UpperIntersection", "strict_lower_sound", "strict_upper_sound", "wp_StrictLower", "wp_StrictUpper", "wp_StrictLowerSound", "wp_StrictUpperSound", "constrain_sound", "wp_Constrain", "wp_ConstrainSound")},
        {
            "dispatch-lower-as-upper": ('case ">=":\n\t\treturn Lower(r, bound)', 'case ">=":\n\t\treturn Upper(r, bound)'),
            "dispatch-strict-lower-as-upper": ('case ">":\n\t\treturn StrictLower(r, bound)', 'case ">":\n\t\treturn StrictUpper(r, bound)'),
            "dispatch-strict-upper-as-lower": ('case "<":\n\t\treturn StrictUpper(r, bound)', 'case "<":\n\t\treturn StrictLower(r, bound)'),
            "dispatch-upper-as-lower": ('case "<=":\n\t\treturn Upper(r, bound)', 'case "<=":\n\t\treturn Lower(r, bound)'),
            "dispatch-equality-misses-upper": ("return Upper(Lower(r, bound), bound)", "return Lower(r, bound)"),
            "dispatch-default-narrows": ("default:\n\t\treturn r", "default:\n\t\treturn Lower(r, bound)"),
            "strict-lower-overflow": ("if bound == int64(^uint64(0)>>1)", "if false"),
            "strict-upper-overflow": ("if bound == -int64(^uint64(0)>>1)-1", "if false"),
            "strict-lower-includes-boundary": ("return Lower(r, bound+1)", "return Lower(r, bound)"),
            "strict-upper-includes-boundary": ("return Upper(r, bound-1)", "return Upper(r, bound)"),
            "strict-lower-wrong-transfer": ("return Lower(r, bound+1)", "return Upper(r, bound+1)"),
            "strict-upper-wrong-transfer": ("return Upper(r, bound-1)", "return Lower(r, bound-1)"),
            "lower-forgets-knowledge": ("r.HasLower = true", "r.HasLower = false"),
            "lower-trusts-unknown": ("!r.HasLower || r.Lower < bound", "r.Lower < bound"),
            "lower-wrong-direction": ("r.Lower < bound", "r.Lower > bound"),
            "lower-excludes-boundary": ("r.Lower = bound", "r.Lower = bound+1"),
            "upper-forgets-knowledge": ("r.HasUpper = true", "r.HasUpper = false"),
            "upper-trusts-unknown": ("!r.HasUpper || r.Upper > bound", "r.Upper > bound"),
            "upper-wrong-direction": ("r.Upper > bound", "r.Upper < bound"),
            "upper-excludes-boundary": ("r.Upper = bound", "r.Upper = bound-1"),
            "wrong-advisory-path": ("Path: path", "Path: function"),
            "wrong-advisory-owner": ("ID: function", "ID: name"),
            "wrong-advisory-line": ("Path: path, Line: int(line),", "Path: path, Line: int(line)+1,"),
            "wrong-advisory-kind": ('Kind: "negative_index"', 'Kind: "nilerr"'),
            "wrong-advisory-score": ("ScoreMilli: 800", "ScoreMilli: 801"),
            "drop-index-name": ('"upper-bound check on " + name', '"upper-bound check on "'),
            "omit-deduplication": ("if seen[site.Symbol] {", "if false {"),
            "deduplicate-by-node": ("if seen[site.Symbol]", "if seen[site.Node]"),
            "forget-seen-binding": ("seen[site.Symbol] = true", "seen[site.Symbol] = false"),
            "ignore-access-risk": ("if !NegativeIndex(site.Index, site.UpperChecked, site.Bounds) {", "if false {"),
            "mark-safe-binding": ("site := accesses[i]", "site := accesses[i]\n\t\tseen[site.Symbol] = true"),
            "skip-last-access": ("i < len(accesses)", "i < len(accesses)-1"),
            "duplicate-access": ("append(out, site)", "append(out, site, site)"),
            "forge-source-node": ("out = append(out, site)", "site.Node = 0\n\t\tout = append(out, site)"),
            "narrow-lower-bound": ("min(a.Lower, b.Lower)", "max(a.Lower, b.Lower)"),
            "narrow-upper-bound": ("max(a.Upper, b.Upper)", "min(a.Upper, b.Upper)"),
            "forge-known-lower": ("a.HasLower && b.HasLower", "a.HasLower || b.HasLower"),
            "forge-known-upper": ("a.HasUpper && b.HasUpper", "a.HasUpper || b.HasUpper"),
            "stale-unknown-lower": ("r.Lower = 0", "r.Lower = r.Lower"),
            "stale-unknown-upper": ("r.Upper = 0", "r.Upper = r.Upper"),
            "warn-without-index": ("!index || !upperChecked", "!upperChecked"),
            "warn-without-upper-check": ("!index || !upperChecked", "!index"),
            "treat-zero-as-negative": ("r.Lower < 0", "r.Lower <= 0"),
            "trust-unknown-lower": ("!r.HasLower || r.Lower < 0", "r.Lower < 0"),
        },
        "AuditInterval.lean", "Gstyle/Proof/Interval.lean", "sound interval join, negative-index decision and stable binding selection",
        ("PDG extraction", "state transfer and fixpoint traversal", "Value-to-Range projection", "Access projection, source-node resolution and candidate rendering"),
        mutation_modules=interval_mutation_modules(),
    )



def runtime_value_profile():
    source = ROOT / "domain/runtimeguard/condition.go"
    value = "domain/runtimeguard/value.go"
    sources = tuple(sorted(path for directory in ("domain/runtimeguard", "domain/runtimeguard/interval", "domain/pdg", "domain/pdg/query")
                           for path in (ROOT / directory).glob("*.go") if not path.name.endswith("_test.go")))
    prefix = "Perennial.github_com.rstyled.gstyle.domain.runtimeguard."
    return Profile(
        ROOT / "dist/verification/runtime-value", "github_com/rstyled/gstyle/domain/runtimeguard.lean",
        "./domain/runtimeguard", source, sources,
        {prefix + name for name in ("wp_Bounds", "wp_integerConstraint", "wp_integerConstraintSound", "wp_mergeBounds", "wp_mergePointers", "wp_merge", "wp_mergeSound")},
        {
            "merge-drops-other-range": ("interval.Join(v.Bounds(), other.Bounds())", "interval.Join(v.Bounds(), v.Bounds())", value),
            "merge-swaps-lower": ("return Value{Lower: r.Lower, Upper: r.Upper", "return Value{Lower: r.Upper, Upper: r.Upper", value),
            "merge-loses-trigger": ("v.UpperChecked || other.UpperChecked", "v.UpperChecked && other.UpperChecked", value),
            "merge-forges-length": ("v.Length && other.Length", "v.Length || other.Length", value),
            "merge-forges-below-length": ("v.BelowLength && other.BelowLength", "v.BelowLength || other.BelowLength", value),
            "merge-forges-non-nil": ("v.NonNil && other.NonNil", "v.NonNil || other.NonNil", value),
            "merge-forges-nil": ("v.Nil && other.Nil", "v.Nil || other.Nil", value),
            "merge-forges-guard": ("v.Guarded && other.Guarded", "v.Guarded || other.Guarded", value),
            "merge-confuses-identity": ("v.Identity == other.Identity", "v.Identity != other.Identity", value),
            "merge-drops-common-identity": ("out.Identity = v.Identity", "out.Identity = 0", value),
            "project-wrong-lower": ("Lower: v.Lower", "Lower: v.Upper", value),
            "project-wrong-upper": ("Upper: v.Upper", "Upper: v.Lower", value),
            "project-forged-lower": ("HasLower: v.HasLower", "HasLower: true", value),
            "project-forged-upper": ("HasUpper: v.HasUpper", "HasUpper: true", value),
            "copy-swapped-bounds": ("value.Lower, value.Upper = r.Lower, r.Upper", "value.Lower, value.Upper = r.Upper, r.Lower"),
            "copy-lost-lower": ("value.HasLower, value.HasUpper = r.HasLower, r.HasUpper", "value.HasLower, value.HasUpper = false, r.HasUpper"),
            "copy-lost-upper": ("value.HasLower, value.HasUpper = r.HasLower, r.HasUpper", "value.HasLower, value.HasUpper = r.HasLower, false"),
            "copy-clobbers-pointer": ("value.Lower, value.Upper = r.Lower, r.Upper", "value.Lower, value.Upper = r.Lower, r.Upper; value.NonNil = false"),
            "copy-clobbers-identity": ("value.Lower, value.Upper = r.Lower, r.Upper", "value.Lower, value.Upper = r.Lower, r.Upper; value.Identity = 0"),
            "wrong-operator-input": ("interval.Constrain(value.Bounds(), operator, bound)", 'interval.Constrain(value.Bounds(), ">=", bound)'),
            "wrong-bound-input": ("interval.Constrain(value.Bounds(), operator, bound)", "interval.Constrain(value.Bounds(), operator, bound+1)"),
        },
        "AuditRuntimeValue.lean", "Gstyle/Proof/RuntimeValue.lean", "actual Value interval projection, comparison adapter and complete Value join",
        ("PDG extraction", "other state transfers, state-map joins and fixpoint traversal", "Access and Candidate projection"), ("Code",),
        (("github_com/rstyled/gstyle/domain/runtimeguard/interval.lean", "./domain/runtimeguard/interval"),
         ("github_com/rstyled/gstyle/domain/pdg.lean", "./domain/pdg"),
         ("github_com/rstyled/gstyle/domain/pdg/query.lean", "./domain/pdg/query")),
    )



def state_map_profile():
    value = "domain/runtimeguard/value.go"
    graph = "domain/runtimeguard/graph.go"
    mutations = {
        "join-left-wrong-state": ("value.merge(v.binding(symbol, next))", "value.merge(v.binding(symbol, previous))"),
        "join-right-wrong-state": ("value.merge(v.binding(symbol, previous))", "value.merge(v.binding(symbol, next))"),
        "join-loses-right-only": ("if _, present := previous[symbol]; !present {", "if _, present := previous[symbol]; present {"),
        "join-overwrites-common": ("if _, present := previous[symbol]; !present {", "if _, present := previous[symbol]; present || !present {"),
        "join-wrong-key": ("out[symbol] = value.merge(v.binding(symbol, next))", "out[symbol+1] = value.merge(v.binding(symbol, next))"),
        "join-skips-left": ("for symbol, value := range previous {", "for symbol, value := range next {"),
        "join-nil-left-alias": ("return copyState(next)", "return next"),
        "join-nil-right-wrong-state": ("return copyState(previous)", "return copyState(next)"),
        "copy-loses-values": ("out[symbol] = value", "_ = value; out[symbol] = Value{}", value),
        "copy-loses-keys": ("out[symbol] = value", "_, _ = symbol, value", value),
        "copy-aliases-input": ("out := make(State, len(state))", "out := state", value),
        "copy-wrong-key": ("out[symbol] = value", "out[symbol+1] = value", value),
        "binding-inverts-presence": ("if value, ok := state[symbol]; ok {", "if value, ok := state[symbol]; !ok {", graph),
        "binding-ignores-unstable": ("if v.unstable[symbol-1] {", "if false {", graph),
        "binding-wrong-key": ("if value, ok := state[symbol]; ok {", "if value, ok := state[symbol+1]; ok {", graph),
        "binding-forges-unsigned-bound": ("return Value{Lower: 0, HasLower: true}", "return Value{Lower: 1, HasLower: true}", graph),
        "binding-wrong-identity": ("uint64(symbol) * 2", "uint64(symbol) * 3", graph),
    }
    equality = 'if other, ok := b[symbol]; !ok || other != value {'
    mutations.update({
        "equality-drops-presence": (equality, 'if other := b[symbol]; other != value {', value),
        "equality-inverts-presence": (equality, 'if other, ok := b[symbol]; ok || other != value {', value),
        "equality-wrong-key": (equality, 'if other, ok := b[symbol+1]; !ok || other != value {', value),
        "equality-ignores-values": (equality, '_ = value; if _, ok := b[symbol]; !ok {', value),
        "equality-identity-only": (equality, 'if other, ok := b[symbol]; !ok || other.Identity != value.Identity {', value),
        "equality-needs-both": (equality, 'if other, ok := b[symbol]; !ok && other != value {', value),
        "equality-inverts-values": (equality, 'if other, ok := b[symbol]; !ok || other == value {', value),
        "equality-skips-size": ('if len(a) != len(b) {', 'if false {', value),
        "equality-mutates-left": (equality, 'delete(a, symbol); '+equality, value),
        "equality-mutates-right": (equality, 'delete(b, symbol); '+equality, value),
    })
    modules = {name: "Gstyle/Proof/StateMap/" +
               ("Equality" if name.startswith("equality-") else "Copy" if name.startswith("copy-") else "Binding" if name.startswith("binding-") else "Join") + ".lean"
               for name in mutations}
    prefix = "Perennial.github_com.rstyled.gstyle.domain.runtimeguard."
    names = ("wp_base", "wp_binding", "wp_copyState", "wp_copyState_nil", "wp_join", "wp_join_sound",
             "wp_join_left_nil", "wp_join_right_nil", "wp_join_both_nil", "joinLookup_domain", "state_join_sound", "state_equal_of_size_and_entries", "wp_equalState",
             "wp_equalState_left_nil", "wp_equalState_right_nil", "wp_equalState_both_nil")
    return replace(runtime_value_profile(), report=ROOT / "dist/verification/state-map", source=ROOT / "domain/runtimeguard/flow.go",
                   theorems={prefix + name for name in names}, mutations=mutations,
                   audit="AuditStateMap.lean", proof_file="Gstyle/Proof/StateMap/Join.lean",
                   verified="production state-map join and exact content equality, input preservation and conditional interval soundness",
                   unverified=("aliased non-nil join input maps", "invalid cached binding references", "concurrent mutation", "Go-to-PDG facts and concrete execution semantics", "fixpoint convergence and other transfers", "package initialization"),
                   mutation_modules=modules)


def numeric_transfer_profile():
    track = "domain/runtimeguard/track.go"
    mutations = {
        "exact-ignores-lower": ("value.HasLower && value.HasUpper && value.Lower == value.Upper", "value.HasUpper && value.Lower == value.Upper"),
        "exact-ignores-upper": ("value.HasLower && value.HasUpper && value.Lower == value.Upper", "value.HasLower && value.Lower == value.Upper"),
        "exact-inverts-equality": ("value.Lower == value.Upper", "value.Lower != value.Upper"),
        "empty-rejects-singleton": ("value.Lower > value.Upper", "value.Lower >= value.Upper"),
        "empty-ignores-lower": ("value.HasLower && value.HasUpper && value.Lower > value.Upper", "value.HasUpper && value.Lower > value.Upper"),
        "empty-ignores-upper": ("value.HasLower && value.HasUpper && value.Lower > value.Upper", "value.HasLower && value.Lower > value.Upper"),
        "length-inverts-kind": ("if !bound.Length {", "if bound.Length {"),
        "length-wrong-operator": ('operator != "<" && operator != "<="', 'operator != "<" && operator != ">="'),
        "length-loses-check": ("value.UpperChecked = true", "value.UpperChecked = false"),
        "length-loses-strict": ('value.BelowLength || operator == "<"', 'value.BelowLength || operator == "<="'),
        "length-clears-existing": ('value.BelowLength || operator == "<"', 'operator == "<"'),
        "impossible-wrong-kind": ('v.kinds[symbol-1] != `"signed"`', 'v.kinds[symbol-1] == `"signed"`'),
        "impossible-wrong-min": ("bound.Lower == -int64(^uint64(0)>>1)-1", "bound.Lower == -int64(^uint64(0)>>1)"),
        "impossible-wrong-max": ("bound.Lower == int64(^uint64(0)>>1)", "bound.Lower == int64(^uint64(0)>>1)-1"),
        "infeasible-needs-both": ("emptyRange(value) || v.impossibleSignedComparison", "emptyRange(value) && v.impossibleSignedComparison"),
        "tracked-inverts-flag": ("&& v.tracked[symbol-1]", "&& !v.tracked[symbol-1]", track),
        "constrain-inverts-tracked": ("if !v.trackedBinding(symbol) {", "if v.trackedBinding(symbol) {"),
        "constrain-skips-integer": ("if exactInteger(bound) {", "if false && exactInteger(bound) {"),
        "constrain-wrong-bound": ("integerConstraint(value, operator, bound.Lower)", "integerConstraint(value, operator, bound.Lower+1)"),
        "constrain-wrong-key": ("state[symbol] = value", "state[symbol+1] = value"),
        "constrain-skips-rejection": ("if v.infeasibleConstraint(symbol, operator, bound, value) {", "if false && v.infeasibleConstraint(symbol, operator, bound, value) {"),
    }
    prefix = "Perennial.github_com.rstyled.gstyle.domain.runtimeguard."
    names = ("wp_exactInteger", "wp_emptyRange", "wp_lengthConstraint", "wp_trackedBinding",
             "wp_impossibleSignedComparison", "wp_infeasibleConstraint", "wp_constrain_numeric",
             "length_bounds", "numeric_constraint_sound", "represented_not_empty", "represented_comparison_possible",
             "constraint_preserves_other_keys", "constraint_state_sound", "wp_constrain_numeric_sound")
    return replace(runtime_value_profile(), report=ROOT / "dist/verification/numeric-transfer",
                   theorems={prefix + name for name in names}, mutations=mutations,
                   audit="AuditNumericTransfer.lean", proof_file="Gstyle/Proof/Transfer/Numeric.lean",
                   verified="production numeric constraint transfer, exact state update, other-key preservation and conditional interval soundness",
                   unverified=("nil/pointer constraints and alias semantics", "invalid references and mismatched metadata slices", "PDG extraction and comparison fact soundness", "assignments, increments and branch composition", "fixpoint convergence", "package initialization"),
                   mutation_modules={})


def signed_step_profile():
    mutations = {
        "step-reverses-increment": ("return incrementInteger(value)", "return decrementInteger(value)"),
        "step-reverses-decrement": ("return decrementInteger(value)\n}", "return incrementInteger(value)\n}"),
        "increment-loses-length-guard": ("!value.BelowLength &&", "true &&"),
        "increment-trusts-unknown-upper": ("!value.HasUpper ||", "false ||"),
        "increment-misses-overflow": ("value.Upper == int64(^uint64(0)>>1)", "value.Upper == int64(^uint64(0)>>1)-1"),
        "increment-ignores-overflow": ("value.Upper == int64(^uint64(0)>>1)", "false"),
        "increment-wrong-lower": ("Lower: value.Lower}", "Lower: value.Lower+1}"),
        "increment-forges-lower": ("HasLower: value.HasLower", "HasLower: true"),
        "increment-loses-lower": ("HasLower: value.HasLower", "HasLower: false"),
        "increment-retains-length": ("Lower: value.Lower}", "Lower: value.Lower, BelowLength: value.BelowLength}"),
        "increment-retains-upper-check": ("Lower: value.Lower}", "Lower: value.Lower, UpperChecked: value.UpperChecked}"),
        "decrement-trusts-unknown-lower": ("!value.HasLower ||", "false ||"),
        "decrement-misses-overflow": ("value.Lower == -int64(^uint64(0)>>1)-1", "value.Lower == -int64(^uint64(0)>>1)"),
        "decrement-ignores-overflow": ("value.Lower == -int64(^uint64(0)>>1)-1", "false"),
        "decrement-wrong-upper": ("Upper: value.Upper}", "Upper: value.Upper-1}"),
        "decrement-forges-upper": ("HasUpper: value.HasUpper", "HasUpper: true"),
        "decrement-retains-identity": ("Upper: value.Upper}", "Upper: value.Upper, Identity: value.Identity}"),
    }
    prefix = "Perennial.github_com.rstyled.gstyle.domain.runtimeguard."
    names = ("increment_integer_sound", "decrement_integer_sound", "integer_step_sound", "integer_step_metadata",
             "integer_step_interval_sound", "wp_incrementInteger", "wp_decrementInteger", "wp_integerStep", "wp_integerStep_sound")
    return replace(runtime_value_profile(), report=ROOT / "dist/verification/signed-step",
                   source=ROOT / "domain/runtimeguard/increment.go",
                   theorems={prefix + name for name in names}, mutations=mutations,
                   audit="AuditSignedStep.lean", proof_file="Gstyle/Proof/Transfer/Step.lean",
                   verified="production signed 64-bit increment/decrement widening, wrapped-result interval soundness and metadata invalidation",
                   unverified=("type-width and operator fact extraction/dispatch", "semantic provenance of BelowLength", "other arithmetic and narrower signed transfers", "state-map assignment integration", "fixpoint convergence", "pointer/alias semantics", "package initialization"),
                   mutation_modules={})


def assignment_profile():
    pointer = "domain/runtimeguard/pointer.go"
    write = "state[symbol] = value\n}\n\nfunc (v *graphView) forgetIdentity"
    mutations = {
        "selection-accepts-multiple": ("if writes != 1 {", "if writes < 1 {"),
        "selection-rejects-single": ("if writes != 1 {", "if writes == 1 {"),
        "selection-ignores-category": ('(form == "" && category == pdg.Declaration)', '(form == "")'),
        "selection-copies-add-assignment": ('form == "=" || form == ":="', 'form == "=" || form == ":=" || form == "+="'),
        "selection-copies-multiply-assignment": ('form == "=" || form == ":="', 'form == "=" || form == ":=" || form == "*="'),
        "selection-copies-shift-assignment": ('form == "=" || form == ":="', 'form == "=" || form == ":=" || form == "<<="'),
        "selection-wrong-declaration-tag": ("Declaration Category = iota", "Declaration Category = iota+1", "domain/pdg/model.go"),
        "assignment-overwrites-known-lower": ("base.HasLower && !value.HasLower", "base.HasLower && value.HasLower"),
        "assignment-forges-lower": ("value.Lower, value.HasLower = base.Lower, true", "value.Lower, value.HasLower = base.Lower+1, true"),
        "assignment-loses-lower-flag": ("value.Lower, value.HasLower = base.Lower, true", "value.Lower, value.HasLower = base.Lower, false"),
        "assignment-clobbers-upper": ("value.Lower, value.HasLower = base.Lower, true", "value.Lower, value.HasLower = base.Lower, true; value.Upper = 0"),
        "store-wrong-key": (write, write.replace("state[symbol]", "state[symbol+1]"), pointer),
        "store-loses-value": (write, write.replace("= value", "= Value{}"), pointer),
        "store-drops-write": (write, write.replace("state[symbol] = value", "_, _ = state, value"), pointer),
        "store-skips-normalization": ("value = v.assignmentValue(symbol, value)", "_ = v.assignmentValue(symbol, value)", pointer),
    }
    prefix = "Perennial.github_com.rstyled.gstyle.domain.runtimeguard."
    names = ("assignment_interval_sound", "compound_discards_rhs", "wp_singleValueWrite", "wp_assignmentValue",
             "wp_assign_nonpointer", "wp_singleValueWrite_compound", "wp_assign_interval_sound", "wp_assign_unknown_signed")
    return replace(runtime_value_profile(), report=ROOT / "dist/verification/assignment",
                   source=ROOT / "domain/runtimeguard/flow.go",
                   theorems={prefix + name for name in names}, mutations=mutations,
                   audit="AuditAssignment.lean", proof_file="Gstyle/Proof/Transfer/Assignment.lean",
                   verified="production assignment eligibility and non-pointer state-map writes, conservative compound selection and conditional interval preservation",
                   unverified=("RHS expression evaluation and PDG fact extraction", "full analysis.transfer composition, tracking, range writes and guard clearing", "pointer assignment and alias invalidation", "invalid references, nil maps and concurrent writes", "precise compound arithmetic", "branch composition and convergence", "package initialization"),
                   mutation_modules={})


def branch_profile():
    graph = "domain/runtimeguard/graph.go"
    left = "left := v.assume(v.operand(ref, 0), truth, copyState(state))"
    right = "right := v.assume(v.operand(ref, 1), truth, copyState(state))"
    mutations = {
        "branch-inverts-composition": ('(operator == "&&") == truth', '(operator == "&&") != truth'),
        "branch-skips-unreachable-left": ("if left == nil {", "if false && left == nil {"),
        "branch-loses-sequential-refinement": ("return v.assume(v.operand(ref, 1), truth, left)", "return v.assume(v.operand(ref, 1), truth, state)"),
        "branch-left-aliases-input": (left, left.replace("copyState(state)", "state")),
        "branch-right-aliases-input": (right, right.replace("copyState(state)", "state")),
        "branch-wrong-right-child": (right, right.replace("operand(ref, 1)", "operand(ref, 0)")),
        "branch-inverts-right-truth": (right, right.replace(", truth,", ", !truth,")),
        "branch-right-uses-left-state": (right, right.replace("copyState(state)", "copyState(left)")),
        "branch-resurrects-both-unreachable": ("if left == nil && right == nil {", "if false && left == nil && right == nil {"),
        "branch-drops-live-alternative": ("if left == nil && right == nil {", "if left == nil || right == nil {"),
        "branch-skips-join": ("return v.join(left, right)", "return left"),
        "operand-wrong-row": ("return v.operands[ref-1][position]", "return v.operands[ref][position]", graph),
        "operand-wrong-position": ("return v.operands[ref-1][position]", "return v.operands[ref-1][1-position]", graph),
        "operand-rejects-last-position": ("position >= len(v.operands[ref-1])", "position >= len(v.operands[ref-1])-1", graph),
        "operand-ignores-zero-reference": ("if ref == 0 || position >= len(v.operands[ref-1]) {", "if false || position >= len(v.operands[ref-1]) {", graph),
    }
    prefix = "Perennial.github_com.rstyled.gstyle.domain.runtimeguard."
    names = ("branch_join_sound", "logical_truth_cases", "branch_result_sound", "joined_branch_valid",
             "wp_operand", "wp_operand_zero", "wp_operand_outside", "wp_logical", "wp_logical_sound")
    return replace(runtime_value_profile(), report=ROOT / "dist/verification/branch-composition",
                   theorems={prefix + name for name in names}, mutations=mutations,
                   audit="AuditBranch.lean", proof_file="Gstyle/Proof/Transfer/Branch.lean",
                   verified="production logical composition conditional on explicit child contracts, operand lookup and conditional interval-state composition",
                   unverified=("child assume execution and semantic contracts", "PDG condition/operator facts and metadata validity", "condition evaluation effects", "full analysis.transfer and RHS evaluation", "fixpoint convergence", "pointer/alias semantics", "package initialization"),
                   mutation_modules={})


def pdg_text_profile():
    source = ROOT / "domain/pdg/tables.go"
    sources = tuple(sorted(path for directory in ("domain/pdg", "domain/pdg/query")
                           for path in (ROOT / directory).glob("*.go") if not path.name.endswith("_test.go")))
    prefix = "Perennial.github_com.rstyled.gstyle.domain.pdg."
    return Profile(
        ROOT / "dist/verification/pdg-text", "github_com/rstyled/gstyle/domain/pdg.lean", "./domain/pdg", source, sources,
        {prefix + name for name in ("wp_Text_present", "wp_Text_absent", "wp_Text")},
        {
            "text-wrong-slot": ("return t.Strings[index]", "return t.Strings[index+1]"),
            "text-first-slot": ("return t.Strings[index]", "_ = index; return t.Strings[0]"),
            "text-wrong-count": ("Ref(id), len(t.Strings)", "Ref(id), len(t.Strings)-1"),
            "text-uses-capacity": ("Ref(id), len(t.Strings)", "Ref(id), cap(t.Strings)"),
            "text-wrong-reference": ("Ref(id), len(t.Strings)", "Ref(id+1), len(t.Strings)"),
            "text-inverts-presence": ("if !ok {", "if ok {"),
            "text-ignores-absence": ("if !ok {", "if false && !ok {"),
            "text-forges-absence-value": ('return ""', 'return "unknown"'),
            "text-mutates-strings": ("return t.Strings[index]", 't.Strings[index] = ""; return t.Strings[index]'),
            "text-mutates-metadata": ("index, ok := query.ReferenceIndex", "t.Sources = nil; index, ok := query.ReferenceIndex"),
        },
        "AuditPdgText.lean", "Gstyle/Proof/Pdg/Text.lean", "production PDG string-table lookup and preservation under read-only ownership",
        ("string interning and PDG extraction", "graph/attribute and extension lookup", "condition and transfer integration", "package initialization"),
        ("Code",), (("github_com/rstyled/gstyle/domain/pdg/query.lean", "./domain/pdg/query"),),
    )


def operator_profile():
    source = ROOT / "domain/runtimeguard/graph.go"
    sources = tuple(sorted(path for directory in ("domain/runtimeguard", "domain/runtimeguard/interval", "domain/pdg", "domain/pdg/query")
                           for path in (ROOT / directory).glob("*.go") if not path.name.endswith("_test.go")))
    prefix = "Perennial.github_com.rstyled.gstyle.domain.runtimeguard."
    return Profile(
        ROOT / "dist/verification/operator", "github_com/rstyled/gstyle/domain/runtimeguard.lean", "./domain/runtimeguard", source, sources,
        {prefix + name for name in ("wp_operatorText_known", "wp_operatorText_unavailable")},
        {
            "operator-inverted-status": ("if fact.Status != pdg.Known {", "if fact.Status == pdg.Known {"),
            "operator-accepts-unknown": ("if fact.Status != pdg.Known {", "if fact.Status > pdg.Known {"),
            "operator-accepts-unsupported": ("if fact.Status != pdg.Known {", "if fact.Status != pdg.Known && fact.Status != pdg.Unsupported {"),
            "operator-rejects-known": ("if fact.Status != pdg.Known {", "if fact.Status <= pdg.Known {"),
            "operator-forges-unavailable": ('if fact.Status != pdg.Known {\n\t\treturn ""', 'if fact.Status != pdg.Known {\n\t\treturn "unknown"'),
            "operator-wrong-reference": ("return tables.Text(fact.Value)", "return tables.Text(fact.Value+1)"),
            "operator-discards-known": ("return tables.Text(fact.Value)", 'return ""'),
            "operator-mutates-metadata": ("return tables.Text(fact.Value)", "tables.Sources = nil; return tables.Text(fact.Value)"),
        },
        "AuditOperator.lean", "Gstyle/Proof/Transfer/Operator.lean", "production operator reader rejects every unavailable status and preserves known text/table resources",
        ("graph attribute extraction and reader call-site composition", "fact provenance and producer correctness", "condition and transfer semantics", "package initialization"),
        ("Code",), (("github_com/rstyled/gstyle/domain/pdg.lean", "./domain/pdg"),
                    ("github_com/rstyled/gstyle/domain/pdg/query.lean", "./domain/pdg/query"),
                    ("github_com/rstyled/gstyle/domain/runtimeguard/interval.lean", "./domain/runtimeguard/interval")),
    )


def pdg_list_profile():
    source = ROOT / "domain/pdg/attributes.go"
    sources = tuple(sorted(path for directory in ("domain/pdg", "domain/pdg/query")
                           for path in (ROOT / directory).glob("*.go") if not path.name.endswith("_test.go")))
    prefix = "Perennial.github_com.rstyled.gstyle.domain.pdg."
    return Profile(
        ROOT / "dist/verification/pdg-list", "github_com/rstyled/gstyle/domain/pdg.lean", "./domain/pdg", source, sources,
        {prefix + name for name in ("wp_listAt_present", "wp_listAt_absent", "wp_listAt")},
        {
            "list-wrong-slot": ("return lists[ref-1]", "return lists[ref]"),
            "list-first-row": ("return lists[ref-1]", "return lists[0]"),
            "list-uses-capacity": ("uint64(len(lists))", "uint64(cap(lists))"),
            "list-rejects-last": ("uint64(ref) > uint64(len(lists))", "uint64(ref) >= uint64(len(lists))"),
            "list-ignores-zero": ("if ref == 0 || uint64(ref) > uint64(len(lists)) {", "if uint64(ref) > uint64(len(lists)) {"),
            "list-discards-present": ("return lists[ref-1]", "return nil"),
            "list-collapses-empty": ("return lists[ref-1]", "if len(lists[ref-1]) == 0 { return nil }; return lists[ref-1]"),
            "list-forges-absence": ("return nil\n\t}\n\treturn lists[ref-1]", "return lists[0]\n\t}\n\treturn lists[ref-1]"),
            "list-mutates-table": ("return lists[ref-1]", "lists[ref-1] = nil; return lists[ref-1]"),
        },
        "AuditPdgList.lean", "Gstyle/Proof/Pdg/List.lean", "production generic PDG list lookup returns exact descriptors and preserves borrowed tables",
        ("graph attributes and expansion", "inner row storage and graph construction", "fact provenance and condition/transfer integration", "package initialization"),
        ("Code",), (("github_com/rstyled/gstyle/domain/pdg/query.lean", "./domain/pdg/query"),),
    )


def pdg_attributes_profile():
    source = ROOT / "domain/pdg/attributes.go"
    sources = tuple(sorted(path for directory in ("domain/pdg", "domain/pdg/query")
                           for path in (ROOT / directory).glob("*.go") if not path.name.endswith("_test.go")))
    prefix = "Perennial.github_com.rstyled.gstyle.domain.pdg."
    mutations = {
        "attributes-wrong-slot": ("g.Attributes[node.Attributes-1]", "g.Attributes[node.Attributes]"),
        "attributes-first-record": ("g.Attributes[node.Attributes-1]", "g.Attributes[0]"),
        "attributes-uses-capacity": ("uint64(len(g.Attributes))", "uint64(cap(g.Attributes))"),
        "attributes-rejects-last": ("uint64(node.Attributes) > uint64(len(g.Attributes))", "uint64(node.Attributes) >= uint64(len(g.Attributes))"),
        "attributes-ignores-zero": ("node.Attributes == 0 || uint64(node.Attributes)", "uint64(node.Attributes)"),
        "attributes-discards-present": ("return g.expand(g.Attributes[node.Attributes-1])", "return Attributes{}"),
        "expand-swaps-facts": ("Writes: listAt(g.AttributeTables.Accesses, a.Writes),\n\t\tOperator: a.Operator, Literal: a.Literal", "Writes: listAt(g.AttributeTables.Accesses, a.Writes),\n\t\tOperator: a.Literal, Literal: a.Operator"),
        "expand-wrong-symbols": ("g.AttributeTables.References, a.Symbols", "g.AttributeTables.References, a.Defines"),
        "expand-wrong-reads": ("g.AttributeTables.Accesses, a.Reads", "g.AttributeTables.Accesses, a.Writes"),
        "expand-wrong-extensions": ("g.AttributeTables.Extensions, a.Extensions", "g.AttributeTables.Extensions, a.Reads"),
        "expand-wrong-outcomes": ("g.outcomes(a.Outcomes)", "g.outcomes(a.Defines)"),
        "outcomes-zeroes-values": ("texts[i] = Text(value)", "_ = value; texts[i] = 0"),
        "outcomes-reverses-values": ("texts[i] = Text(value)", "texts[len(texts)-1-i] = Text(value)"),
        "outcomes-shifts-values": ("texts[i] = Text(value)", "texts[i] = Text(value+1)"),
        "outcomes-skips-last": ("texts[i] = Text(value)", "if i == len(refs)-1 { continue }; texts[i] = Text(value)"),
        "outcomes-discards-result": ("return texts", "return nil"),
        "outcomes-first-value": ("texts[i] = Text(value)", "_ = value; texts[i] = Text(refs[0])"),
        "outcomes-allocates-empty": ("if len(refs) == 0 {\n\t\treturn nil", "if len(refs) == 0 {\n\t\treturn []Text{}"),
        "attributes-mutates-graph": ("if node.Attributes == 0 ||", "g.Profile = 0; if node.Attributes == 0 ||"),
        "attributes-mutates-record": ("return g.expand(g.Attributes[node.Attributes-1])", "g.Attributes[node.Attributes-1] = AttributeSet{}; return g.expand(g.Attributes[node.Attributes-1])"),
        "outcomes-mutates-source": ("texts[i] = Text(value)", "refs[i] = 0; texts[i] = Text(value)"),
    }
    return Profile(
        ROOT / "dist/verification/pdg-attributes", "github_com/rstyled/gstyle/domain/pdg.lean", "./domain/pdg", source, sources,
        {prefix + name for name in ("filledPrefix_step", "wp_outcomes", "wp_expand", "wp_NodeAttributes_zero", "wp_NodeAttributes_absent", "wp_NodeAttributes_present")},
        mutations, "AuditPdgAttributes.lean", "Gstyle/Proof/Pdg/Attributes.lean",
        "production graph attribute selection, expansion and complete outcome copying with input preservation",
        ("PDG construction and extension lookup", "producer and fact provenance", "condition and full transfer integration", "package initialization"),
        ("Code",), (("github_com/rstyled/gstyle/domain/pdg/query.lean", "./domain/pdg/query"),),
        mutation_modules={name: "Gstyle/Proof/Pdg/Outcomes.lean" for name in mutations if name.startswith("outcomes-")},
    )


def runtime_facts_profile():
    source = ROOT / "domain/runtimeguard/graph.go"
    sources = tuple(sorted(path for directory in ("domain/runtimeguard", "domain/runtimeguard/interval", "domain/pdg", "domain/pdg/query")
                           for path in (ROOT / directory).glob("*.go") if not path.name.endswith("_test.go")))
    prefix = "Perennial.github_com.rstyled.gstyle.domain.runtimeguard."
    header = 'if g.Tables.Text(value.Name) == "gstyle.runtime:"+name {'
    result = 'return g.Tables.Text(value.Value)'
    attrs = 'return v.graph.NodeAttributes(v.graph.Nodes[ref-1])'
    fact = 'return extension(v.graph, v.attrs(ref).Extensions, name)'
    mutations = {
        "extension-omits-namespace": ('"gstyle.runtime:"+name', 'name'),
        "extension-wrong-namespace": ('"gstyle.runtime:"+name', '"gstyle.Runtime:"+name'),
        "extension-key-from-value": ('g.Tables.Text(value.Name)', 'g.Tables.Text(value.Value)'),
        "extension-shifts-key": ('g.Tables.Text(value.Name)', 'g.Tables.Text(value.Name+1)'),
        "extension-inverts-match": (header, header.replace(' == ', ' != ')),
        "extension-ignores-key": (header, 'if true {'),
        "extension-returns-name": (result, 'return g.Tables.Text(value.Name)'),
        "extension-discards-match": (result, 'return ""'),
        "extension-shifts-value": (result, 'return g.Tables.Text(value.Value+1)'),
        "extension-skips-first-name": (header, 'if value.Name == values[0].Name { continue }; '+header),
        "extension-skips-tail-value": (header, 'if value.Value == values[len(values)-1].Value { continue }; '+header),
        "extension-skips-absent-value": (header, header.removesuffix(' {')+' && value.Value != 0 {'),
        "extension-stops-on-nonmatch": (result+'\n\t\t}\n\t}\n\treturn ""', result+'\n\t\t}\n\t\treturn ""\n\t}\n\treturn ""'),
        "extension-forges-absence": ('return ""\n}\n\nfunc validateFlow', 'return "unknown"\n}\n\nfunc validateFlow'),
        "extension-mutates-graph": (header, 'g.Profile = 0; '+header),
        "extension-mutates-table": (header, 'g.Tables.Sources = nil; '+header),
        "extension-mutates-rows": (header, 'values[0].Value = 0; '+header),
        "extension-mutates-strings": (result, 'g.Tables.Strings[value.Value-1] = ""; '+result),
        "attrs-ignores-zero": ('func (v *graphView) attrs(ref pdg.Ref) pdg.Attributes {\n\tif ref == 0 {', 'func (v *graphView) attrs(ref pdg.Ref) pdg.Attributes {\n\tif false && ref == 0 {'),
        "attrs-wrong-node": (attrs, attrs.replace('[ref-1]', '[ref]')),
        "attrs-first-node": (attrs, attrs.replace('[ref-1]', '[0]')),
        "attrs-discards-record": (attrs, 'return pdg.Attributes{}'),
        "attrs-mutates-view": (attrs, 'v.graph = nil; '+attrs),
        "fact-wrong-name": (fact, 'return extension(v.graph, v.attrs(ref).Extensions, "other")'),
        "fact-wrong-node": (fact, 'return extension(v.graph, v.attrs(0).Extensions, name)'),
        "fact-discards-extensions": (fact, 'return extension(v.graph, nil, name)'),
        "fact-mutates-view": (fact, 'v.kinds = nil; '+fact),
    }
    boolean_source = "domain/runtimeguard/condition.go"
    reject = 'return value == "false" && truth'
    mutations.update({
        "boolean-inverts-true": ('return !truth', 'return truth', boolean_source),
        "boolean-inverts-false": (reject, 'return value == "false" && !truth', boolean_source),
        "boolean-swaps-true-literal": ('if value == "true" {', 'if value == "false" {', boolean_source),
        "boolean-swaps-false-literal": (reject, 'return value == "true" && truth', boolean_source),
        "boolean-unknown-as-false": (reject, 'return truth', boolean_source),
        "boolean-empty-as-false": (reject, 'return (value == "false" || value == "") && truth', boolean_source),
        "boolean-string-as-false": (reject, 'return (value == "false" || value == `"false"`) && truth', boolean_source),
        "boolean-always-conflicts": (reject, 'return true', boolean_source),
        "boolean-drops-true-conflict": ('return !truth', 'return false', boolean_source),
        "boolean-drops-false-conflict": (reject, 'return false', boolean_source),
        "boolean-bypasses-check": ('booleanContradiction(value, truth) {', 'false && booleanContradiction(value, truth) {', boolean_source),
        "boolean-keeps-rejected-state": ('if value := v.fact(ref, "boolean"); booleanContradiction(value, truth) {\n\t\treturn nil', 'if value := v.fact(ref, "boolean"); booleanContradiction(value, truth) {\n\t\treturn state', boolean_source),
    })
    direct = 'return node.Category == pdg.Read && len(attrs.Reads) == 1 && attrs.Reads[0].Location == 0'
    parens = 'if operator == "parentheses" {\n\t\treturn v.symbol(v.operand(ref, 0))'
    operator = 'operator := operatorText(v.graph.Tables, attrs.Operator)\n\t'+parens
    mutations.update({
        "read-wrong-category": (direct, direct.replace('== pdg.Read', '!= pdg.Read')),
        "read-broad-arity": (direct, direct.replace('== 1', '> 0')),
        "read-empty-arity": (direct, direct.replace('== 1', '== 0')),
        "read-inverts-location": (direct, direct.replace('Location == 0', 'Location != 0')),
        "read-uses-symbol": (direct, direct.replace('.Location', '.Symbol')),
        "read-uses-writes": (direct, direct.replace('attrs.Reads', 'attrs.Writes')),
        "symbol-zero-wrong": ('func (v *graphView) symbol(ref pdg.Ref) pdg.Ref {\n\tif ref == 0 {\n\t\treturn 0', 'func (v *graphView) symbol(ref pdg.Ref) pdg.Ref {\n\tif ref == 0 {\n\t\treturn 1'),
        "symbol-cache-dropped": ('return v.fields[ref-1]', 'return 0'),
        "symbol-cache-index": ('if v.fields[ref-1] != 0 {', 'if v.fields[ref] != 0 {'),
        "symbol-cache-inverted": ('if v.fields[ref-1] != 0 {', 'if v.fields[ref-1] == 0 {'),
        "symbol-limit-inclusive": ('if int(symbol) > len(v.kinds) {', 'if int(symbol) >= len(v.kinds) {'),
        "symbol-limit-skipped": ('if int(symbol) > len(v.kinds) {', 'if false {'),
        "symbol-read-location": ('symbol := attrs.Reads[0].Symbol', 'symbol := attrs.Reads[0].Location'),
        "symbol-ignores-status": (operator, operator.replace('operatorText(v.graph.Tables, attrs.Operator)', 'v.graph.Tables.Text(attrs.Operator.Value)')),
        "symbol-wrong-operator": (parens, parens.replace('"parentheses"', '"conversion"')),
        "symbol-wrong-child": (parens, parens.replace('v.operand(ref, 0)', 'v.operand(ref, 1)')),
        "symbol-skips-recursion": (parens, parens.replace('v.symbol(v.operand(ref, 0))', 'v.operand(ref, 0)')),
        "symbol-mutates-reads": ('symbol := attrs.Reads[0].Symbol', 'attrs.Reads[0].Location = 1; symbol := attrs.Reads[0].Symbol'),
    })
    modules = {name: ("Gstyle/Proof/Transfer/Extension.lean" if name.startswith("extension-") else "Gstyle/Proof/Transfer/Boolean.lean")
               for name in mutations if name.startswith(("extension-", "boolean-"))}
    modules.update({name: ("Gstyle/Proof/Transfer/BindingRead.lean" if name.startswith(("read-", "symbol-zero-", "symbol-cache-")) else "Gstyle/Proof/Transfer/SymbolRead.lean")
                    for name in mutations if name.startswith(("read-", "symbol-"))})
    model_source = "verification/length/Gstyle/Proof/Transfer/SymbolModel.lean"
    rows_source = "verification/length/Gstyle/Proof/Pdg/ReadonlyRows.lean"
    model_mutations = {
        "model-zero-result": ('if ref = W32 0 then .inl (W32 0) else', 'if ref = W32 0 then .inl (W32 1) else'),
        "model-cache-result": ('if cached ≠ W32 0 then .inl cached else', 'if cached ≠ W32 0 then .inl (cached + W32 1) else'),
        "model-cache-guard": ('if cached ≠ W32 0 then .inl cached else', 'if cached = W32 0 then .inl cached else'),
        "model-direct-location": ("(reads.head?.getD (zero_val pdg.Access)).Symbol'", "(reads.head?.getD (zero_val pdg.Access)).Location'"),
        "model-direct-limit": ("boundedReadSymbol (sint.Z s.view.kinds'.len) (reads.head?", "boundedReadSymbol (sint.Z s.view.kinds'.len + 1) (reads.head?"),
        "model-direct-inverted": ('if directReadValue node reads then', 'if !directReadValue node reads then'),
        "model-operator-status": ("let op := if attrs.Operator'.Status' = 1#8 then", "let op := if attrs.Operator'.Status' = 0#8 then"),
        "model-operator-name": ('if op = go!"parentheses" then', 'if op = go!"conversion" then'),
        "model-child-position": ("(zero_val (GoArray w32 2))).arr[0]?", "(zero_val (GoArray w32 2))).arr[1]?"),
        "model-recurses-parent": ('| .inr child => symbolResolve s fuel child', '| .inr child => symbolResolve s fuel ref'),
        "model-empty-fuel": ('| 0, _ => none', '| 0, _ => some (W32 0)'),
    }
    mutations.update({name: (*mutation, model_source) for name, mutation in model_mutations.items()})
    mutations["rows-shift-content"] = ('contents[(uint.Z ref - 1).toNat]?.getD []', 'contents[(uint.Z ref).toNat]?.getD []', rows_source)
    mutations["rows-admit-zero"] = ('if 0 < uint.Z ref ∧ uint.Z ref ≤ contents.length then', 'if 0 ≤ uint.Z ref ∧ uint.Z ref ≤ contents.length then', rows_source)
    modules.update({name: "Gstyle/Proof/Transfer/SymbolModel.lean" for name in model_mutations})
    modules.update({name: "Gstyle/Proof/Pdg/ReadonlyRows.lean" for name in mutations if name.startswith("rows-")})
    sources += (ROOT / model_source, ROOT / rows_source)
    recursive_theorems = {
        'Perennial.github_com.rstyled.gstyle.domain.pdg.readonlyRows_length',
        'Perennial.github_com.rstyled.gstyle.domain.pdg.readonlyRows_at',
        'Perennial.github_com.rstyled.gstyle.domain.pdg.readonlyRows_at_ref',
        'Perennial.github_com.rstyled.gstyle.domain.runtimeguard.querySnapshotOwn_rows',
        'Perennial.github_com.rstyled.gstyle.domain.runtimeguard.directReadValue_true',
        'Perennial.github_com.rstyled.gstyle.domain.runtimeguard.symbolStep_zero',
        'Perennial.github_com.rstyled.gstyle.domain.runtimeguard.symbolResolve_zero',
        'Perennial.github_com.rstyled.gstyle.domain.runtimeguard.symbolResolve_terminal',
        'Perennial.github_com.rstyled.gstyle.domain.runtimeguard.symbolResolve_child',
        'Perennial.github_com.rstyled.gstyle.domain.runtimeguard.symbolResolve_add',
        'Perennial.github_com.rstyled.gstyle.domain.runtimeguard.symbolResolve_deterministic',
        'Perennial.github_com.rstyled.gstyle.domain.runtimeguard.symbolResolve_rank',
        'Perennial.github_com.rstyled.gstyle.domain.runtimeguard.symbolStep_cached',
        'Perennial.github_com.rstyled.gstyle.domain.runtimeguard.symbolStep_absent',
        'Perennial.github_com.rstyled.gstyle.domain.runtimeguard.symbolStep_present',
        'Perennial.github_com.rstyled.gstyle.domain.runtimeguard.wp_symbol_snapshot_zero',
        'Perennial.github_com.rstyled.gstyle.domain.runtimeguard.wp_symbol_snapshot_cached',
        'Perennial.github_com.rstyled.gstyle.domain.runtimeguard.wp_symbol_snapshot_absent',
        'Perennial.github_com.rstyled.gstyle.domain.runtimeguard.wp_symbol_snapshot_direct',
        'Perennial.github_com.rstyled.gstyle.domain.runtimeguard.wp_symbol_snapshot_unavailable_operator',
        'Perennial.github_com.rstyled.gstyle.domain.runtimeguard.wp_symbol_snapshot_known_other',
        'Perennial.github_com.rstyled.gstyle.domain.runtimeguard.wp_symbol_snapshot_parentheses',
        'Perennial.github_com.rstyled.gstyle.domain.runtimeguard.symbolRecordStep_direct',
        'Perennial.github_com.rstyled.gstyle.domain.runtimeguard.symbolRecordStep_unavailable',
        'Perennial.github_com.rstyled.gstyle.domain.runtimeguard.symbolRecordStep_other',
        'Perennial.github_com.rstyled.gstyle.domain.runtimeguard.symbolRecordStep_parentheses',
        'Perennial.github_com.rstyled.gstyle.domain.runtimeguard.wp_symbol_resolve',
        'Perennial.github_com.rstyled.gstyle.domain.runtimeguard.wp_symbol_ranked',
    }
    return Profile(
        ROOT / "dist/verification/runtime-facts", "github_com/rstyled/gstyle/domain/runtimeguard.lean", "./domain/runtimeguard", source, sources,
        {prefix + name for name in ("extensionValue_first", "extensionValue_missing", "wp_extension", "wp_extension_empty", "wp_attrs_zero", "wp_attrs_absent", "wp_attrs_present", "wp_fact_zero", "wp_fact_absent", "wp_fact_present", "booleanConflict_unknown", "booleanConflict_literals", "wp_booleanContradiction", "wp_assume_boolean_reject", "wp_directBindingRead_other", "wp_directBindingRead_arity", "wp_directBindingRead_single", "wp_directBindingRead", "wp_operand_readonly", "wp_symbol_zero", "wp_symbol_cached", "wp_symbol_direct", "wp_symbol_unavailable_operator", "wp_symbol_known_other", "wp_symbol_absent_attributes", "wp_symbol_parentheses", "wp_symbol_parentheses_cached_child", "wp_symbol_parentheses_zero_child")} | recursive_theorems,
        mutations, "AuditRuntimeFacts.lean", "Gstyle/Proof/Transfer/Facts.lean",
        "production runtime queries and finite/ranked recursive symbol resolution with preserved snapshots",
        ("cyclic or unranked symbol inputs", "snapshot construction", "producer and semantic fact provenance", "metadata cache agreement and RHS evaluation", "actual condition and whole state-transfer semantics", "package initialization"),
        ("Code",), (("github_com/rstyled/gstyle/domain/pdg.lean", "./domain/pdg"),
                    ("github_com/rstyled/gstyle/domain/pdg/query.lean", "./domain/pdg/query"),
                    ("github_com/rstyled/gstyle/domain/runtimeguard/interval.lean", "./domain/runtimeguard/interval")),
        mutation_modules=modules,
    )


def value_evaluation_profile():
    base = runtime_facts_profile()
    groups = {
        "ConstantExecution": {
            "constant-wrong-non-nil": ('v.fact(ref, "non-nil")', 'v.fact(ref, "nil")'),
            "constant-wrong-nil": ('Nil: v.fact(ref, "nil")', 'Nil: v.fact(ref, "non-nil")'),
            "constant-integer-key": ('strconv.ParseInt(v.fact(ref, "integer"), 10, 64)', 'strconv.ParseInt(v.fact(ref, "other"), 10, 64)'),
            "constant-integer-base": ('strconv.ParseInt(v.fact(ref, "integer"), 10, 64)', 'strconv.ParseInt(v.fact(ref, "integer"), 16, 64)'),
            "constant-integer-width": ('strconv.ParseInt(v.fact(ref, "integer"), 10, 64)', 'strconv.ParseInt(v.fact(ref, "integer"), 10, 32)'),
            "constant-integer-error": ('strconv.ParseInt(v.fact(ref, "integer"), 10, 64); err == nil', 'strconv.ParseInt(v.fact(ref, "integer"), 10, 64); err != nil'),
            "constant-lower-shift": ('value.Lower, value.Upper = integer, integer', 'value.Lower, value.Upper = integer+1, integer'),
            "constant-upper-shift": ('value.Lower, value.Upper = integer, integer', 'value.Lower, value.Upper = integer, integer-1'),
            "constant-missing-lower": ('value.HasLower, value.HasUpper, value.Integer = true, true, true', 'value.HasLower, value.HasUpper, value.Integer = false, true, true'),
            "constant-missing-upper": ('value.HasLower, value.HasUpper, value.Integer = true, true, true', 'value.HasLower, value.HasUpper, value.Integer = true, false, true'),
            "constant-missing-integer": ('value.HasLower, value.HasUpper, value.Integer = true, true, true', 'value.HasLower, value.HasUpper, value.Integer = true, true, false'),
            "constant-extent-key": ('strconv.ParseInt(v.fact(ref, "literal-extent"), 10, 64)', 'strconv.ParseInt(v.fact(ref, "integer"), 10, 64)'),
            "constant-extent-error": ('strconv.ParseInt(v.fact(ref, "literal-extent"), 10, 64); err == nil', 'strconv.ParseInt(v.fact(ref, "literal-extent"), 10, 64); err != nil'),
            "constant-extent-shift": ('value.Extent, value.HasExtent = extent, true', 'value.Extent, value.HasExtent = extent+1, true'),
            "constant-missing-extent": ('value.Extent, value.HasExtent = extent, true', 'value.Extent, value.HasExtent = extent, false'),
            "constant-mutates-view": ('func (v *graphView) constantValue(ref pdg.Ref) Value {', 'func (v *graphView) constantValue(ref pdg.Ref) Value { v.fields = nil;'),
        },
        "LengthExecution": {
            "value-zero-forges-bound": ('func (v *graphView) value(ref pdg.Ref, state State) Value {\n\tif ref == 0 {\n\t\treturn Value{}', 'func (v *graphView) value(ref pdg.Ref, state State) Value {\n\tif ref == 0 {\n\t\treturn Value{HasLower: true}'),
            "length-inverts-admission": ('if !value.Length {', 'if value.Length {'),
            "length-inverts-extent": ('if !source.HasExtent {', 'if source.HasExtent {'),
            "length-shifts-lower": ('Lower: source.Extent, Upper: source.Extent', 'Lower: source.Extent+1, Upper: source.Extent'),
            "length-shifts-upper": ('Lower: source.Extent, Upper: source.Extent', 'Lower: source.Extent, Upper: source.Extent-1'),
            "length-drops-integer": ('return Value{Length: true, Integer: true, HasLower:', 'return Value{Length: true, Integer: false, HasLower:'),
            "conversion-inverts-constant": ('if constant.Integer {', 'if !constant.Integer {'),
        },
        "ConversionExecution": {
            "conversion-type-key": ('switch v.fact(ref, "type") {', 'switch v.fact(ref, "other") {'),
            "conversion-swaps-nil": ('return Value{NonNil: source.NonNil, Nil: source.Nil}', 'return Value{NonNil: source.Nil, Nil: source.NonNil}'),
            "conversion-unsigned-shift": ('return Value{HasLower: true, Lower: 0, Length: source.Length}', 'return Value{HasLower: true, Lower: 1, Length: source.Length}'),
            "conversion-default-drops-length": ('return Value{Length: source.Length}', 'return Value{}'),
        },
        "ValueDispatch": {
            "value-ignores-binding": ('if symbol := v.symbol(ref); symbol != 0 {', 'if symbol := v.symbol(ref); false && symbol != 0 {'),
            "value-wrong-binding": ('return v.binding(symbol, state)', 'return v.binding(ref, state)'),
            "value-parent-wrong-child": ('return v.value(v.operand(ref, 0), state)', 'return v.value(v.operand(ref, 1), state)'),
            "value-ignores-callee-status": ('value.Length = attrs.Callee.Status == pdg.Known &&', 'value.Length = true &&'),
            "value-wrong-callee": ('v.graph.Tables.Text(attrs.Callee.Value) == "len"', 'v.graph.Tables.Text(attrs.Callee.Value) == "cap"'),
        },
    }
    mutations, modules = {}, {}
    for module, entries in groups.items():
        mutations.update(entries)
        modules.update({name: f"Gstyle/Proof/Transfer/{module}.lean" for name in entries})
    model_source = "verification/length/Gstyle/Proof/Transfer/ValueResolution.lean"
    model_mutations = {
        "value-model-empty-fuel": ('| 0, _ => none', '| 0, _ => some (zero_val Value)'),
        "value-model-child-index": (').arr[0]?).getD (W32 0)', ').arr[1]?).getD (W32 0)'),
        "value-model-wrong-parent": ('if op = go!"parentheses" then valueResolve', 'if op = go!"conversion" then valueResolve'),
        "value-model-wrong-conversion": ('else if op = go!"conversion" then', 'else if op = go!"parentheses" then'),
        "value-model-binding-limit": ('if uint.Z key ≤ s.kinds.length then', 'if uint.Z key < s.kinds.length then'),
    }
    mutations.update({name: (*mutation, model_source) for name, mutation in model_mutations.items()})
    modules.update({name: "Gstyle/Proof/Transfer/ValueResolution.lean" for name in model_mutations})
    proof_sources = (
        "Gstyle/Proof/Transfer/FactSnapshot.lean",
        "Gstyle/Proof/Transfer/ValueModel.lean",
        "Gstyle/Proof/Transfer/ParseIntContract.lean",
        "Gstyle/Proof/Transfer/ConstantExecution.lean",
        "Gstyle/Proof/Transfer/LengthExecution.lean",
        "Gstyle/Proof/Transfer/ConversionExecution.lean",
        "Gstyle/Proof/Transfer/ValueDispatch.lean",
        "Gstyle/Proof/Transfer/ValueEvaluation.lean",
        "Gstyle/Proof/Transfer/ValueResolution.lean",
        "Gstyle/Proof/StrconvPackage.lean", "Gstyle/Proof/RuntimeValue.lean",
        "Gstyle/Proof/StateMap/Binding.lean", "Gstyle/Proof/Pdg/GraphLayout.lean",
    )
    names = (
        "wp_fact_snapshot",
        "wp_attrs_snapshot_absent",
        "wp_attrs_snapshot",
        "wp_operatorText_snapshot",
        "signedDecimal_range",
        "decimalInt64_range",
        "decimalInt64_zero",
        "decimalInt64_negative_zero",
        "decimalInt64_positive",
        "decimalInt64_negative",
        "decimalInt64_leading_zero",
        "decimalInt64_minimum",
        "decimalInt64_maximum",
        "decimalInt64_positive_overflow",
        "decimalInt64_negative_overflow",
        "decimalInt64_empty",
        "decimalInt64_minus_only",
        "decimalInt64_plus_only",
        "decimalInt64_space",
        "decimalInt64_underscore",
        "decimalInt64_hex",
        "decimalInt64_unicode",
        "parseInt64Result_success",
        "parseInt64Result_error",
        "wp_constantValue_snapshot",
        "wp_lengthValue_not_length",
        "wp_value_zero",
        "wp_conversionValue_integer",
        "wp_lengthValue_child",
        "wp_lengthValue_zero_child",
        "wp_conversionValue_child",
        "wp_value_binding_dispatch",
        "wp_binding_snapshot",
        "wp_value_cached",
        "wp_value_parentheses_dispatch",
        "wp_value_conversion_constant",
        "wp_value_conversion_child",
        "wp_value_other_dispatch",
        "wp_value_other_no_length",
        "wp_value_other_length_child",
        "wp_value_bound",
        "wp_binding_snapshot_out_of_range",
        "wp_value_bound_out_of_range",
        "wp_value_evaluation",
        "valueChild_row",
        "valueResolve_evaluation",
        "wp_value_resolve",
        "wp_base",
        "wp_binding",
    )
    return replace(base, report=ROOT / "dist/verification/value-evaluation",
                   sources=base.sources + tuple(PROJECT / path for path in proof_sources),
                   theorems={"Perennial.github_com.rstyled.gstyle.domain.runtimeguard." + name for name in names},
                   mutations=mutations, mutation_modules=modules, audit="AuditValueEvaluation.lean",
                   proof_file="Gstyle/Proof/Transfer/ValueDispatch.lean",
                   verified="production value evaluation for finite model witnesses with preserved PDG snapshots and state maps",
                   unverified=("standard-library ParseInt implementation", "witness existence and termination", "snapshot construction and producer fact truth", "whole state-transfer and program interval soundness", "program pointer/alias semantics and package initialization"))


def comparison_profile():
    base = value_evaluation_profile()
    groups = {
        "ComparisonExecution": {
            "compare-inverts-truth": ('if !truth {\n\t\toperator = inverse[operator]', 'if truth {\n\t\toperator = inverse[operator]'),
            "compare-wrong-inverse-table": ('operator = inverse[operator]', 'operator = reverse[operator]'),
            "compare-wrong-inverse-key": ('operator = inverse[operator]', 'operator = inverse["=="]'),
            "compare-wrong-left-position": ('left, right := v.operand(ref, 0), v.operand(ref, 1)', 'left, right := v.operand(ref, 1), v.operand(ref, 1)'),
            "compare-wrong-right-position": ('left, right := v.operand(ref, 0), v.operand(ref, 1)', 'left, right := v.operand(ref, 0), v.operand(ref, 0)'),
            "compare-swaps-values": ('leftValue, rightValue := v.value(left, state), v.value(right, state)', 'leftValue, rightValue := v.value(right, state), v.value(left, state)'),
            "compare-drops-right-state": ('leftValue, rightValue := v.value(left, state), v.value(right, state)', 'leftValue, rightValue := v.value(left, state), v.value(right, nil)'),
            "compare-wrong-left-symbol": ('if symbol := v.symbol(left); symbol != 0 {', 'if symbol := v.symbol(right); symbol != 0 {'),
            "compare-wrong-right-symbol": ('if symbol := v.symbol(right); symbol != 0 {', 'if symbol := v.symbol(left); symbol != 0 {'),
            "compare-rejects-live-state": ('if state == nil {', 'if state != nil {'),
            "compare-continues-after-rejection": ('if state == nil {', 'if false && state == nil {'),
            "compare-discards-final-state": ('return state\n}\n\nfunc (v *graphView) constrain', 'return nil\n}\n\nfunc (v *graphView) constrain'),
            "compare-mutates-operands": ('left, right := v.operand(ref, 0), v.operand(ref, 1)', 'v.operands[ref-1][0] = 0; left, right := v.operand(ref, 0), v.operand(ref, 1)'),
            "compare-mutates-inverse": ('operator = inverse[operator]', 'inverse[operator] = ""; operator = inverse[operator]'),
        },
        "ComparisonTransfers": {
            "compare-ignores-left-binding": ('if symbol := v.symbol(left); symbol != 0 {', 'if symbol := v.symbol(left); false && symbol != 0 {'),
            "compare-ignores-right-binding": ('if symbol := v.symbol(right); symbol != 0 {', 'if symbol := v.symbol(right); false && symbol != 0 {'),
            "compare-wrong-left-bound": ('state = v.constrain(symbol, operator, rightValue, state)', '_ = rightValue; state = v.constrain(symbol, operator, leftValue, state)'),
            "compare-drops-left-rejection": ('state = v.constrain(symbol, operator, rightValue, state)', 'v.constrain(symbol, operator, rightValue, state)'),
            "compare-wrong-reverse-table": ('reverse[operator]', 'inverse[operator]'),
            "compare-wrong-reverse-key": ('reverse[operator]', 'reverse["=="]'),
            "compare-wrong-right-bound": ('state = v.constrain(symbol, reverse[operator], leftValue, state)', '_ = leftValue; state = v.constrain(symbol, reverse[operator], rightValue, state)'),
            "compare-drops-right-rejection": ('state = v.constrain(symbol, reverse[operator], leftValue, state)', 'v.constrain(symbol, reverse[operator], leftValue, state)'),
            "compare-mutates-reverse": ('state = v.constrain(symbol, reverse[operator], leftValue, state)', 'reverse[operator] = ""; state = v.constrain(symbol, reverse[operator], leftValue, state)'),
        },
        "Numeric": {
            "constraint-tracked-off-by-one": ('int(symbol) <= len(v.tracked)', 'int(symbol) < len(v.tracked)', 'domain/runtimeguard/track.go'),
            "constraint-tracks-zero": ('return symbol != 0 && int(symbol) <= len(v.tracked)', 'return int(symbol) <= len(v.tracked)', 'domain/runtimeguard/track.go'),
            "constraint-skips-limit": ('return symbol != 0 && int(symbol) <= len(v.tracked) &&', 'return symbol != 0 &&', 'domain/runtimeguard/track.go'),
            "constraint-wrong-tracked-index": ('return symbol != 0 && int(symbol) <= len(v.tracked) && v.tracked[symbol-1]', 'return symbol != 0 && int(symbol) <= len(v.tracked) && v.tracked[symbol]', 'domain/runtimeguard/track.go'),
            "constraint-ignores-tracking": ('if !v.trackedBinding(symbol) {', 'if false && !v.trackedBinding(symbol) {'),
            "constraint-drops-write": ('state[symbol] = value', 'state[symbol] = bound'),
            "constraint-keeps-infeasible": ('if v.infeasibleConstraint(symbol, operator, bound, value) {\n\t\treturn nil', 'if v.infeasibleConstraint(symbol, operator, bound, value) {\n\t\treturn state'),
        },
    }
    mutations, expected = {}, {}
    for module, entries in groups.items():
        mutations.update(entries)
        expected.update({name: f"Gstyle/Proof/Transfer/{module}.lean" for name in entries})
    model_source = "verification/length/Gstyle/Proof/Transfer/ComparisonModel.lean"
    model_mutations = {
        "compare-model-inverse-strict": ('if op = go!"<" then go!">=" else', 'if op = go!"<" then go!">" else'),
        "compare-model-reverse-equality": ('if op = go!"==" then go!"==" else', 'if op = go!"==" then go!"!=" else'),
        "compare-model-normalization": ('if truth then op else inverseOperator op', 'if truth then inverseOperator op else op'),
        "compare-model-unknown-relation": ('if op = go!"!=" then left ≠ right else True', 'if op = go!"!=" then left ≠ right else False'),
    }
    mutations.update({name: (*change, model_source) for name, change in model_mutations.items()})
    expected.update({name: "Gstyle/Proof/Transfer/ComparisonModel.lean" for name in model_mutations})
    metadata_mutations = {
        "compare-model-absent-tracked": ('(ha : active = false)', '(ha : active = true)'),
        "compare-model-absent-last-key": ('(hout : key = W32 0 ∨ s.kinds.length < uint.Z key)', '(hout : key = W32 0 ∨ s.kinds.length ≤ uint.Z key)'),
    }
    mutations.update({name: (*change, model_source) for name, change in metadata_mutations.items()})
    expected.update({name: "Gstyle/Proof/Transfer/ComparisonConstraint.lean" for name in metadata_mutations})
    state_mutations = {
        "compare-model-first-bound": ('constraintState c.leftActive c.leftKind c.leftChanging c.leftKey m c.operator c.rightValue', 'constraintState c.leftActive c.leftKind c.leftChanging c.leftKey m c.operator c.leftValue'),
        "compare-model-restarts-right": ('constraintState c.rightActive c.rightKind c.rightChanging c.rightKey (comparisonLeftMap c m) c.reversed c.leftValue', 'constraintState c.rightActive c.rightKind c.rightChanging c.rightKey m c.reversed c.leftValue'),
        "compare-model-skips-right-map": ('if comparisonLeftRejected c m then comparisonLeftMap c m else comparisonRightMap c m', 'if comparisonLeftRejected c m then comparisonLeftMap c m else comparisonLeftMap c m'),
        "compare-model-requires-both-rejections": ('comparisonLeftRejected c m || comparisonRightRejected c m', 'comparisonLeftRejected c m && comparisonRightRejected c m'),
    }
    mutations.update({name: (*change, model_source) for name, change in state_mutations.items()})
    expected.update({name: "Gstyle/Proof/Transfer/ComparisonBoth.lean" for name in state_mutations})
    proof_sources = (
        "Gstyle/Proof/Transfer/ComparisonModel.lean",
        "Gstyle/Proof/Transfer/ComparisonExecution.lean",
        "Gstyle/Proof/Transfer/ComparisonConstraint.lean",
        "Gstyle/Proof/Transfer/ComparisonTransfers.lean",
        "Gstyle/Proof/Transfer/ComparisonBoth.lean",
        "Gstyle/Proof/Transfer/ComparisonSound.lean",
        "Gstyle/Proof/Transfer/Numeric.lean", "Gstyle/Proof/Transfer/Sound.lean",
    )
    names = (
        "inverseOperator_relation",
        "reverseOperator_relation",
        "inverseOperator_involution",
        "reverseOperator_involution",
        "normalizedOperator_relation",
        "inverse_reverse_commute",
        "unknownOperator_relation",
        "reversedNormalizedOperator_relation",
        "normalizedBranchRelations",
        "wp_comparison_unbound_true",
        "wp_comparison_unbound_false",
        "wp_comparison_nil_zero_operands_true",
        "wp_comparison_nil_zero_operands_false",
        "wp_constrain_snapshot_numeric",
        "wp_constrain_snapshot_invalid",
        "wp_constrain_snapshot_numeric_any",
        "wp_comparison_left_numeric_true",
        "wp_comparison_left_numeric_false",
        "wp_comparison_right_numeric_true",
        "wp_comparison_right_numeric_false",
        "wp_comparison_both_numeric_true",
        "wp_comparison_both_numeric_false",
        "exact_value_contains",
        "integerRelation_satisfies",
        "numeric_state_step_sound",
        "comparison_numeric_state_sound",
        "comparison_branch_state_sound",
        "wp_comparison_both_numeric_true_sound",
        "wp_comparison_both_numeric_false_sound",
        "wp_trackedBinding",
        "wp_trackedBinding_invalid",
        "wp_impossibleSignedComparison",
        "wp_infeasibleConstraint",
        "wp_constrain_numeric",
    )
    return replace(base, report=ROOT / "dist/verification/comparison", source=ROOT / "domain/runtimeguard/condition.go",
                   sources=base.sources + tuple(PROJECT / path for path in proof_sources),
                   theorems={"Perennial.github_com.rstyled.gstyle.domain.runtimeguard." + name for name in names},
                   mutations=mutations, mutation_modules=expected, audit="AuditComparison.lean",
                   proof_file="Gstyle/Proof/Transfer/ComparisonExecution.lean",
                   verified="production numeric comparisons with exact state updates and all-key interval preservation under explicit input assumptions",
                   unverified=("operator table initialization", "ParseInt implementation and producer fact truth", "program pointer/nil and alias semantics", "32-bit Go execution", "query termination and worklist convergence", "general nil-state operand evaluation"))


def generate(profile, goose, proofgen, source_root, output, log_prefix):
    for tool, directory in ((goose, "Code"), (proofgen, "GeneratedProof")):
        if directory not in profile.generated:
            continue
        run([str(tool), "-out", str(output / directory),
             "-configdir", str(PROJECT / "Gstyle/Code"),
             "-lean-root", "github.com/rstyled/gstyle=Gstyle", "-dir", str(source_root), profile.go_package, *[package for _, package in profile.bindings]],
            ROOT, f"{log_prefix}-{directory}.log", report=profile.report)


def seed_build(lean):
    # Private copies preserve the baseline and let Lake invalidate changed inputs.
    shutil.copytree(PROJECT / ".lake/build", lean / ".lake/build")


def mutation_destination(source, work):
    kind = source.suffix
    source = source.resolve()
    if source.suffix != kind:
        raise RuntimeError(f"mutation source changes language through symlink: {source}")
    if kind == ".go":
        base, destination = ROOT.resolve(), work / "go"
    elif kind == ".lean":
        base, destination = (PROJECT / "Gstyle/Proof").resolve(), work / "lean/Gstyle/Proof"
    else:
        raise RuntimeError(f"unsupported mutation source: {source}")
    try:
        relative = source.relative_to(base)
    except ValueError as error:
        raise RuntimeError(f"mutation source escapes its source tree: {source}") from error
    return destination / relative


def check_mutation(profile, goose, proofgen, item):
    name, mutation = item
    scratch = profile.report / ".scratch"
    before, after = mutation[:2]
    source = ROOT / mutation[2] if len(mutation) == 3 else profile.source
    if source not in profile.sources:
        raise RuntimeError(f"mutation source is not bound: {name}")
    with tempfile.TemporaryDirectory(prefix=name + "-", dir=scratch) as directory:
        work = Path(directory)
        target = mutation_destination(source, work)
        for original_source in profile.sources:
            copied_source = work / "go" / original_source.relative_to(ROOT)
            copied_source.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(original_source, copied_source)
        original = source.read_text()
        if original.count(before) != 1:
            raise RuntimeError(f"mutation target is not unique: {name}")
        (work / "go/go.mod").write_text("module github.com/rstyled/gstyle\n\ngo 1.26.0\n")
        lean = work / "lean"
        lean.mkdir()
        for filename in ("lakefile.toml", "lake-manifest.json", "lean-toolchain", "Gstyle.lean"):
            shutil.copy2(PROJECT / filename, lean / filename)
        shutil.copytree(PROJECT / "Gstyle", lean / "Gstyle")
        (lean / ".lake").mkdir()
        (lean / ".lake/packages").symlink_to(PROJECT / ".lake/packages", target_is_directory=True)
        seed_build(lean)
        target.write_text(original.replace(before, after))
        if source.suffix == ".go":
            generate(profile, goose, proofgen, work / "go", lean / "Gstyle", name)
        proof_file = profile.mutation_modules.get(name, profile.proof_file)
        module = proof_file.removesuffix(".lean").replace("/", ".")
        run(["lake", "build", module], lean, f"{name}-proof.log", expect_failure=True, report=profile.report, proof_file=proof_file)
    print(f"REJECTED {name}", flush=True)
    return name


def mutation_shard():
    raw = os.environ.get("GSTYLE_MUTATION_SHARD", "0/1")
    parts = raw.split("/")
    if len(parts) != 2 or not all(part.isdecimal() for part in parts):
        raise RuntimeError("GSTYLE_MUTATION_SHARD must be INDEX/COUNT")
    index, count = map(int, parts)
    if count < 1 or index >= count:
        raise RuntimeError("mutation shard requires 0 <= INDEX < COUNT")
    return index, count


def shard_mutations(mutations, shard):
    index, count = shard
    if count > len(mutations):
        raise RuntimeError("mutation shard count exceeds mutation count")
    return {name: mutation for position, (name, mutation) in enumerate(mutations.items())
            if position % count == index}


def check_mutations(profile, goose, proofgen):
    (profile.report / ".scratch").mkdir(exist_ok=True)
    # Each worker owns its Go sources and Lean build. Dependencies are prebuilt.
    with ThreadPoolExecutor(max_workers=2) as pool:
        return list(pool.map(lambda item: check_mutation(profile, goose, proofgen, item),
                             profile.mutations.items()))


def check_generated(profile, generated):
    for package in (profile.package, *[package for package, _ in profile.bindings]):
        for directory in profile.generated:
            actual = PROJECT / "Gstyle" / directory / package
            if actual.read_bytes() != (generated / directory / package).read_bytes():
                raise RuntimeError(f"stale generated source: {actual}")


def source_hashes(paths):
    return {str(path.relative_to(ROOT)): hashlib.sha256(path.read_bytes()).hexdigest() for path in paths}


def main(profile=None):
    profile = profile or length_profile()
    report = profile.report
    report.mkdir(parents=True, exist_ok=True)
    success = report / "result.json"
    success.unlink(missing_ok=True)
    bound_sources = source_hashes(profile.sources)
    run(["lake", "build"], PROJECT, "build.log", report=report)
    perennial = PROJECT / ".lake/packages/perennial"
    manifest = json.loads((PROJECT / "lake-manifest.json").read_text())
    for package in manifest["packages"]:
        path = PROJECT / ".lake/packages" / package["name"]
        revision = run(["git", "rev-parse", "HEAD"], path, f"{package['name']}-revision.log", report=report).strip()
        if revision != package["rev"]:
            raise RuntimeError(f"dependency revision differs: {package['name']}")
        run(["git", "diff", "--exit-code", "HEAD"], path, f"{package['name']}-diff.log", report=report)
    goose, proofgen = report / "goose", report / "proofgen"
    for tool in (goose, proofgen):
        run(["go", "build", "-buildvcs=false", "-o", str(tool), f"./goose/cmd/{tool.name}"],
            perennial, f"build-{tool.name}.log", report=report)
    generated = report / "regenerated/Gstyle"
    generate(profile, goose, proofgen, ROOT, generated, "source")
    check_generated(profile, generated)
    check_axioms(run(["lake", "env", "lean", profile.audit], PROJECT, "axioms.log", report=report), profile.theorems)
    shard = mutation_shard()
    mutations = shard_mutations(profile.mutations, shard)
    mutants = check_mutations(replace(profile, mutations=mutations), goose, proofgen)
    if source_hashes(profile.sources) != bound_sources:
        raise RuntimeError("bound source changed during verification")
    success.write_text(json.dumps({
        "sources_sha256": bound_sources,
        "verified": profile.verified, "unverified": profile.unverified,
        "rejected_mutations": mutants,
        "mutation_shard": {"index": shard[0], "count": shard[1], "total_mutations": len(profile.mutations)},
    }, indent=2) + "\n")
    print(f"PASS {profile.verified}; source binding, axiom audit, and {len(mutants)} mutations (shard {shard[0]}/{shard[1]})", flush=True)


if __name__ == "__main__":
    try:
        import sys
        profiles = {(): length_profile, ("--pdg",): pdg_profile, ("--output",): output_profile, ("--interval",): interval_profile, ("--runtime-value",): runtime_value_profile, ("--state-map",): state_map_profile, ("--numeric-transfer",): numeric_transfer_profile, ("--signed-step",): signed_step_profile, ("--assignment",): assignment_profile, ("--branch",): branch_profile, ("--pdg-text",): pdg_text_profile, ("--operator",): operator_profile, ("--pdg-list",): pdg_list_profile, ("--pdg-attributes",): pdg_attributes_profile, ("--runtime-facts",): runtime_facts_profile, ("--value-evaluation",): value_evaluation_profile, ("--comparison",): comparison_profile}
        arguments = tuple(sys.argv[1:])
        if arguments not in profiles:
            raise SystemExit("usage: verify.py [--pdg | --output | --interval | --runtime-value | --state-map | --numeric-transfer | --signed-step | --assignment | --branch | --pdg-text | --operator | --pdg-list | --pdg-attributes | --runtime-facts | --value-evaluation | --comparison]")
        main(profiles[arguments]())
    except (OSError, RuntimeError, subprocess.SubprocessError) as error:
        raise SystemExit(str(error)) from error
