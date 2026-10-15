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
    modules = {name: "Gstyle/Proof/StateMap/" +
               ("Copy" if name.startswith("copy-") else "Binding" if name.startswith("binding-") else "Join") + ".lean"
               for name in mutations}
    prefix = "Perennial.github_com.rstyled.gstyle.domain.runtimeguard."
    names = ("wp_base", "wp_binding", "wp_copyState", "wp_copyState_nil", "wp_join", "wp_join_sound",
             "wp_join_left_nil", "wp_join_right_nil", "wp_join_both_nil", "joinLookup_domain", "state_join_sound")
    return replace(runtime_value_profile(), report=ROOT / "dist/verification/state-map", source=ROOT / "domain/runtimeguard/flow.go",
                   theorems={prefix + name for name in names}, mutations=mutations,
                   audit="AuditStateMap.lean", proof_file="Gstyle/Proof/StateMap/Join.lean",
                   verified="production state-map join, exact key coverage, input preservation and conditional interval soundness",
                   unverified=("aliased non-nil input maps", "invalid cached binding references", "concurrent mutation", "Go-to-PDG facts and concrete execution semantics", "fixpoint convergence and other transfers", "package initialization"),
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


def check_mutation(profile, goose, proofgen, item):
    name, mutation = item
    scratch = profile.report / ".scratch"
    before, after = mutation[:2]
    source = ROOT / mutation[2] if len(mutation) == 3 else profile.source
    if source not in profile.sources:
        raise RuntimeError(f"mutation source is not bound: {name}")
    with tempfile.TemporaryDirectory(prefix=name + "-", dir=scratch) as directory:
        work = Path(directory)
        for original_source in profile.sources:
            target = work / "go" / original_source.relative_to(ROOT)
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(original_source, target)
        original = source.read_text()
        if original.count(before) != 1:
            raise RuntimeError(f"mutation target is not unique: {name}")
        (work / "go" / source.relative_to(ROOT)).write_text(original.replace(before, after))
        (work / "go/go.mod").write_text("module github.com/rstyled/gstyle\n\ngo 1.26.0\n")
        lean = work / "lean"
        lean.mkdir()
        for filename in ("lakefile.toml", "lake-manifest.json", "lean-toolchain", "Gstyle.lean"):
            shutil.copy2(PROJECT / filename, lean / filename)
        shutil.copytree(PROJECT / "Gstyle", lean / "Gstyle")
        (lean / ".lake").mkdir()
        (lean / ".lake/packages").symlink_to(PROJECT / ".lake/packages", target_is_directory=True)
        seed_build(lean)
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


def main(profile=None):
    profile = profile or length_profile()
    report = profile.report
    report.mkdir(parents=True, exist_ok=True)
    success = report / "result.json"
    success.unlink(missing_ok=True)
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
    success.write_text(json.dumps({
        "sources_sha256": {str(path.relative_to(ROOT)): hashlib.sha256(path.read_bytes()).hexdigest() for path in profile.sources},
        "verified": profile.verified, "unverified": profile.unverified,
        "rejected_mutations": mutants,
        "mutation_shard": {"index": shard[0], "count": shard[1], "total_mutations": len(profile.mutations)},
    }, indent=2) + "\n")
    print(f"PASS {profile.verified}; source binding, axiom audit, and {len(mutants)} mutations (shard {shard[0]}/{shard[1]})", flush=True)


if __name__ == "__main__":
    try:
        import sys
        profiles = {(): length_profile, ("--pdg",): pdg_profile, ("--output",): output_profile, ("--interval",): interval_profile, ("--runtime-value",): runtime_value_profile, ("--state-map",): state_map_profile, ("--numeric-transfer",): numeric_transfer_profile, ("--signed-step",): signed_step_profile, ("--assignment",): assignment_profile, ("--branch",): branch_profile, ("--pdg-text",): pdg_text_profile, ("--operator",): operator_profile, ("--pdg-list",): pdg_list_profile}
        arguments = tuple(sys.argv[1:])
        if arguments not in profiles:
            raise SystemExit("usage: verify.py [--pdg | --output | --interval | --runtime-value | --state-map | --numeric-transfer | --signed-step | --assignment | --branch | --pdg-text | --operator | --pdg-list]")
        main(profiles[arguments]())
    except (OSError, RuntimeError, subprocess.SubprocessError) as error:
        raise SystemExit(str(error)) from error
