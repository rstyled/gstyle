#!/usr/bin/env python3
"""Check source binding, Lean proofs, axioms, and deliberate Go mutations."""

from dataclasses import dataclass
import hashlib
import json
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


def length_profile():
    return Profile(REPORT, PACKAGE, GO_PACKAGE, SOURCE, (SOURCE,), THEOREMS,
                   MUTATIONS, "Audit.lean", "Gstyle/Proof/Length.lean", "ordered token-span accumulator and strict threshold",
                   ("Go scanner", "token-to-span extraction", "configuration and diagnostics"))


def pdg_profile():
    prefix = "Perennial.github_com.rstyled.gstyle.domain.pdg.query."
    source = ROOT / "domain/pdg/query/query.go"
    names = ("ReferenceIndex", "ReferenceReject", "EvidenceAt", "EvidenceAbsent",
             "EvidenceStatus", "EvidenceUnknown", "CoverageSupported", "DependencySupported",
             "DependencyAdmitted", "DependencyUnavailable", "BooleanValue", "KnownFalse")
    return Profile(
        ROOT / "dist/verification/pdg-queries", "github_com/rstyled/gstyle/domain/pdg/query.lean",
        "./domain/pdg/query", source, (source, source.with_name("records.go")),
        {prefix + "wp_" + name for name in names},
        {
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


def output_profile():
    source = ROOT / "domain/patterns/outputcheck/output.go"
    prefix = "Perennial.github_com.rstyled.gstyle.domain.patterns.outputcheck."
    return Profile(
        ROOT / "dist/verification/unchecked-output", "github_com/rstyled/gstyle/domain/patterns/outputcheck.lean",
        "./domain/patterns/outputcheck", source, (source,),
        {prefix + name for name in ("wp_Discards", "wp_Select", "wp_Formatted", "wp_Memory", "wp_hasSuppression", "wp_Find", "wp_describe", "wp_Report")},
        {
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
        "AuditOutput.lean", "Gstyle/Proof/Output.lean", "unchecked-output identities, decision, suppression, selection and advisory payloads",
        ("AST/type fact extraction", "corpus traversal, input/Candidate projection and presentation", "Go compiler and stdlib behavior"),
    )


def generate(profile, goose, proofgen, source_root, output, log_prefix):
    for tool, directory in ((goose, "Code"), (proofgen, "GeneratedProof")):
        run([str(tool), "-out", str(output / directory),
             "-configdir", str(PROJECT / "Gstyle/Code"),
             "-lean-root", "github.com/rstyled/gstyle=Gstyle", "-dir", str(source_root), profile.go_package],
            ROOT, f"{log_prefix}-{directory}.log", report=profile.report)


def check_mutations(profile, goose, proofgen):
    results = []
    scratch = profile.report / ".scratch"
    scratch.mkdir(exist_ok=True)
    for name, (before, after) in profile.mutations.items():
        with tempfile.TemporaryDirectory(prefix=name + "-", dir=scratch) as directory:
            work = Path(directory)
            for original_source in profile.sources:
                target = work / "go" / original_source.relative_to(ROOT)
                target.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(original_source, target)
            original = profile.source.read_text()
            if original.count(before) != 1:
                raise RuntimeError(f"mutation target is not unique: {name}")
            (work / "go" / profile.source.relative_to(ROOT)).write_text(original.replace(before, after))
            (work / "go/go.mod").write_text("module github.com/rstyled/gstyle\n\ngo 1.26.0\n")
            lean = work / "lean"
            lean.mkdir()
            for filename in ("lakefile.toml", "lake-manifest.json", "lean-toolchain", "Gstyle.lean"):
                shutil.copy2(PROJECT / filename, lean / filename)
            shutil.copytree(PROJECT / "Gstyle", lean / "Gstyle")
            (lean / ".lake").mkdir()
            (lean / ".lake/packages").symlink_to(PROJECT / ".lake/packages", target_is_directory=True)
            generate(profile, goose, proofgen, work / "go", lean / "Gstyle", name)
            run(["lake", "build"], lean, f"{name}-proof.log", expect_failure=True, report=profile.report, proof_file=profile.proof_file)
            results.append(name)
            print(f"REJECTED {name}", flush=True)
    return results


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
    for directory in ("Code", "GeneratedProof"):
        actual = PROJECT / "Gstyle" / directory / profile.package
        if actual.read_bytes() != (generated / directory / profile.package).read_bytes():
            raise RuntimeError(f"stale generated source: {actual}")
    check_axioms(run(["lake", "env", "lean", profile.audit], PROJECT, "axioms.log", report=report), profile.theorems)
    mutants = check_mutations(profile, goose, proofgen)
    success.write_text(json.dumps({
        "sources_sha256": {str(path.relative_to(ROOT)): hashlib.sha256(path.read_bytes()).hexdigest() for path in profile.sources},
        "verified": profile.verified, "unverified": profile.unverified,
        "rejected_mutations": mutants,
    }, indent=2) + "\n")
    print(f"PASS {profile.verified}; source binding, axiom audit, and {len(mutants)} mutations", flush=True)


if __name__ == "__main__":
    try:
        import sys
        profiles = {(): length_profile, ("--pdg",): pdg_profile, ("--output",): output_profile}
        arguments = tuple(sys.argv[1:])
        if arguments not in profiles:
            raise SystemExit("usage: verify.py [--pdg | --output]")
        main(profiles[arguments]())
    except (OSError, RuntimeError, subprocess.SubprocessError) as error:
        raise SystemExit(str(error)) from error
