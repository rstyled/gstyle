#!/usr/bin/env python3
"""Check source binding, Lean proofs, axioms, and deliberate Go mutations."""

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
MUTATIONS = {
    "omit-last-line": ("last - first + 1", "last - first"),
    "count-overlap-twice": ("first++", "first += 0"),
    "accept-zero-line": ("first >= 1", "first >= 0"),
    "inclusive-limit": ("actual > limit", "actual >= limit"),
}


def run(command, cwd, log, *, expect_failure=False):
    result = subprocess.run(command, cwd=cwd, text=True, capture_output=True, timeout=1800)
    output = result.stdout + result.stderr
    (REPORT / log).write_text(output)
    if expect_failure:
        if result.returncode == 0 or "error: Gstyle/Proof/" not in output:
            raise RuntimeError(f"mutation does not fail its proof: {log}")
    elif result.returncode != 0:
        raise RuntimeError(f"command failed; see {REPORT / log}")
    return output


def check_axioms(output):
    entries = re.findall(r"'([^']+)' depends on axioms:\s*\[([^]]*)\]", output)
    empty = re.findall(r"'([^']+)' does not depend on any axioms", output)
    names = {name for name, _ in entries} | set(empty)
    if names != THEOREMS:
        raise RuntimeError("axiom audit is missing or adds theorem entries")
    for name, raw in entries:
        axioms = {item.strip() for item in raw.split(",") if item.strip()}
        if not axioms <= ALLOWED_AXIOMS:
            raise RuntimeError(f"unexpected axioms for {name}: {axioms - ALLOWED_AXIOMS}")


def generate(goose, proofgen, source_root, output, log_prefix):
    for tool, directory in ((goose, "Code"), (proofgen, "GeneratedProof")):
        run([str(tool), "-out", str(output / directory),
             "-configdir", str(PROJECT / "Gstyle/Code"),
             "-lean-root", "github.com/rstyled/gstyle=Gstyle", "-dir", str(source_root), GO_PACKAGE],
            ROOT, f"{log_prefix}-{directory}.log")


def check_mutations(goose, proofgen):
    results = []
    scratch = REPORT / ".scratch"
    scratch.mkdir(exist_ok=True)
    for name, (before, after) in MUTATIONS.items():
        with tempfile.TemporaryDirectory(prefix=name + "-", dir=scratch) as directory:
            work = Path(directory)
            source = work / "go/domain/quality/linecount"
            source.mkdir(parents=True)
            original = SOURCE.read_text()
            if original.count(before) != 1:
                raise RuntimeError(f"mutation target is not unique: {name}")
            (source / "counter.go").write_text(original.replace(before, after))
            (work / "go/go.mod").write_text("module github.com/rstyled/gstyle\n\ngo 1.26.0\n")
            lean = work / "lean"
            lean.mkdir()
            for filename in ("lakefile.toml", "lake-manifest.json", "lean-toolchain", "Gstyle.lean"):
                shutil.copy2(PROJECT / filename, lean / filename)
            shutil.copytree(PROJECT / "Gstyle", lean / "Gstyle")
            (lean / ".lake").mkdir()
            (lean / ".lake/packages").symlink_to(PROJECT / ".lake/packages", target_is_directory=True)
            generate(goose, proofgen, work / "go", lean / "Gstyle", name)
            run(["lake", "build"], lean, f"{name}-proof.log", expect_failure=True)
            results.append(name)
            print(f"REJECTED {name}", flush=True)
    return results


def main():
    REPORT.mkdir(parents=True, exist_ok=True)
    success = REPORT / "result.json"
    success.unlink(missing_ok=True)
    run(["lake", "build"], PROJECT, "build.log")
    perennial = PROJECT / ".lake/packages/perennial"
    manifest = json.loads((PROJECT / "lake-manifest.json").read_text())
    for package in manifest["packages"]:
        path = PROJECT / ".lake/packages" / package["name"]
        revision = run(["git", "rev-parse", "HEAD"], path, f"{package['name']}-revision.log").strip()
        if revision != package["rev"]:
            raise RuntimeError(f"dependency revision differs: {package['name']}")
        run(["git", "diff", "--exit-code", "HEAD"], path, f"{package['name']}-diff.log")
    goose, proofgen = REPORT / "goose", REPORT / "proofgen"
    for tool in (goose, proofgen):
        run(["go", "build", "-buildvcs=false", "-o", str(tool), f"./goose/cmd/{tool.name}"],
            perennial, f"build-{tool.name}.log")
    generated = REPORT / "regenerated/Gstyle"
    generate(goose, proofgen, ROOT, generated, "source")
    for directory in ("Code", "GeneratedProof"):
        actual = PROJECT / "Gstyle" / directory / PACKAGE
        if actual.read_bytes() != (generated / directory / PACKAGE).read_bytes():
            raise RuntimeError(f"stale generated source: {actual}")
    check_axioms(run(["lake", "env", "lean", "Audit.lean"], PROJECT, "axioms.log"))
    mutants = check_mutations(goose, proofgen)
    success.write_text(json.dumps({
        "source_sha256": hashlib.sha256(SOURCE.read_bytes()).hexdigest(),
        "verified": "ordered token-span accumulator and strict threshold",
        "unverified": ["Go scanner", "token-to-span extraction", "configuration and diagnostics"],
        "rejected_mutations": mutants,
    }, indent=2) + "\n")
    print("PASS function-length core; source binding, axiom audit, and four mutations", flush=True)


if __name__ == "__main__":
    try:
        main()
    except (OSError, RuntimeError, subprocess.SubprocessError) as error:
        raise SystemExit(str(error)) from error
