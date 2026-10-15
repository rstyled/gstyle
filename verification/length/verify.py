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
        (("github_com/rstyled/gstyle/domain/runtimeguard/interval.lean", "./domain/runtimeguard/interval"),),
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
        profiles = {(): length_profile, ("--pdg",): pdg_profile, ("--output",): output_profile, ("--interval",): interval_profile, ("--runtime-value",): runtime_value_profile}
        arguments = tuple(sys.argv[1:])
        if arguments not in profiles:
            raise SystemExit("usage: verify.py [--pdg | --output | --interval | --runtime-value]")
        main(profiles[arguments]())
    except (OSError, RuntimeError, subprocess.SubprocessError) as error:
        raise SystemExit(str(error)) from error
