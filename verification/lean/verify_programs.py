#!/usr/bin/env python3
"""Certify actual analyzer ranges over exhaustive program grammars and all int64 inputs."""
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess

import verify

ROOT = verify.ROOT
PROJECT = verify.PROJECT
REPORT = ROOT / "dist/verification/programs"
THEOREMS = {"Perennial.GstylePrograms." + name for name in
            ("enumerate_complete", "certified_programs", "certified_admitted", "reported_interval", "Certificate.interval_sound")}


def fingerprint():
    files = subprocess.check_output(
        ["git", "ls-files", "-z", "--cached", "--others", "--exclude-standard", "--",
         "*.go", "*.lean", "*.py", "go.mod", "go.sum", "verification/lean/*.toml",
         "verification/lean/lake-manifest.json", "verification/lean/lean-toolchain"], cwd=ROOT)
    return {name: hashlib.sha256((ROOT / name).read_bytes()).hexdigest()
            for name in sorted(set(files.decode().rstrip("\0").split("\0")))}


def run(command, cwd, log, *, rejected=False, theorems=()):
    environment = dict(os.environ, LEAN_NUM_THREADS="2")
    result = subprocess.run(command, cwd=cwd, text=True, capture_output=True,
                            timeout=1800, env=environment)
    output = result.stdout + result.stderr
    log.parent.mkdir(parents=True, exist_ok=True)
    log.write_text(output)
    if not rejected:
        if result.returncode:
            raise RuntimeError(f"command failed; see {log}")
        return output
    if not result.returncode or not proof_rejected(output, command[-1], theorems):
        raise RuntimeError(f"control did not fail its logical contract; see {log}")
    return output


def proof_rejected(output, filename, theorems):
    source = Path(filename).read_text().splitlines()
    diagnostic = re.compile(re.escape(str(filename)) + r":(\d+):\d+: error: (.*)")
    found = False
    for line in output.splitlines():
        if ": error:" not in line and not line.startswith("error:"):
            continue
        match = diagnostic.fullmatch(line)
        if match is None:
            return False
        number, message = match.groups()
        if message != "unsolved goals" and not message.startswith(("Tactic `decide` proved that the proposition", "Tactic `rfl` failed:")):
            return False
        preceding = source[:int(number)]
        declarations = [row.split()[1] for row in preceding if row.startswith("theorem ")]
        if not declarations or declarations[-1] not in theorems:
            return False
        found = True
    return found


def lean_check(directory, *, rejected=False, theorems=()):
    certificate = directory / "Certificate.lean"
    return run(["lake", "env", "lean", "-j2", str(certificate)], PROJECT,
               directory / "proof.log", rejected=rejected, theorems=theorems)


def collect(directory, *, overlay=None, controls=False):
    command = ["go", "run"]
    if overlay:
        command.append("-overlay=" + str(overlay))
    command.extend(["./verification/programs", "--output", str(directory)])
    if controls:
        command.append("--controls")
    run(command, ROOT, directory / "collection.log")


def validate_artifacts(directory):
    data = json.loads((directory / "observations.json").read_text())
    source = (directory / "input.go.txt").read_bytes()
    if hashlib.sha256(source).hexdigest() != data["source_sha256"]:
        raise RuntimeError("captured source hash does not match the observation manifest")
    expected = "package generated\nfunc observe(value int64) {}\n" + "".join(c["source"] for c in data["cases"])
    if source != expected.encode():
        raise RuntimeError("captured program sources do not match the analyzed file")
    names = [c["name"] for c in data["cases"]]
    if len(names) != len(set(names)):
        raise RuntimeError("duplicate program names")
    return data


def production_controls():
    original = ROOT / "domain/runtimeguard/flow.go"
    source = original.read_text()
    statement = "report.Calls = append(report.Calls, Call{Node: ref, FirstArgument: a.view.value(a.view.operand(ref, position), state)})"
    if source.count(statement) != 1:
        raise RuntimeError("production call observation control target is not unique")
    changes = {
        "forged-lower": "Value{HasLower: true, Lower: 1}",
        "erased-bounds": "Value{}",
    }
    for name, replacement in changes.items():
        directory = REPORT / "controls" / name
        directory.mkdir(parents=True, exist_ok=True)
        changed = directory / "flow.go.txt"
        changed.write_text(source.replace(statement, statement +
            "\n\treport.Calls[len(report.Calls)-1].FirstArgument = " + replacement))
        overlay = directory / "overlay.json"
        overlay.write_text(json.dumps({"Replace": {str(original): str(changed)}}))
        collect(directory, overlay=overlay, controls=True)
        lean_check(directory, rejected=True, theorems={"soundCase0", "soundCase1", "soundCase2", "soundCase3", "precisionCase1", "precisionCase2", "precisionCase3"}
                   if name == "forged-lower" else {"precisionCase1", "precisionCase2", "precisionCase3"})
        print(f"REJECTED production {name}", flush=True)


def certificate_controls():
    baseline = REPORT / "controls/baseline"
    collect(baseline, controls=True)
    verify.check_axioms(lean_check(baseline), {"GeneratedPrograms.complete0", "GeneratedPrograms.source_file"})
    text = (baseline / "Certificate.lean").read_text()
    changes = {
        "omitted-program": ("def certificates0 : List Certificate := [cCase0, cCase1, cCase2, cCase3]",
                            "def certificates0 : List Certificate := [cCase0, cCase1, cCase2]", {"coverage0"}),
        "wrong-source": (text.split("theorem source_file : ")[1].split("\n")[0],
                         text.split("theorem source_file : ")[1].split("\n")[0].replace("result = x", "result = 0"), {"source_file"}),
    }
    for name, (before, after, theorems) in changes.items():
        if text.count(before) != 1 or before == after:
            raise RuntimeError(f"certificate control target is not unique: {name}")
        directory = REPORT / "controls" / name
        directory.mkdir(parents=True, exist_ok=True)
        (directory / "Certificate.lean").write_text(text.replace(before, after))
        lean_check(directory, rejected=True, theorems=theorems)
        print(f"REJECTED certificate {name}", flush=True)


def main():
    REPORT.mkdir(parents=True, exist_ok=True)
    success = REPORT / "result.json"
    success.unlink(missing_ok=True)
    before = fingerprint()
    run(["lake", "build", "Gstyle.Proof.Programs.Contract"], PROJECT, REPORT / "build.log")
    verify.check_dependencies(REPORT)
    audit = run(["lake", "env", "lean", "AuditPrograms.lean"], PROJECT, REPORT / "axioms.log")
    verify.check_axioms(audit, THEOREMS)
    collect(REPORT)
    data = validate_artifacts(REPORT)
    verify.check_axioms(lean_check(REPORT),
                        {"GeneratedPrograms.complete" + str(i) for i in range(len(data["suites"]))} |
                        {"GeneratedPrograms.source_file"})
    print(f"PROVED {len(data['cases'])} production reports for every signed 64-bit input", flush=True)
    production_controls()
    certificate_controls()
    if fingerprint() != before:
        raise RuntimeError("verification sources changed during certification")
    success.write_text(json.dumps({
        "verified": "captured production ranges contain every result of each enumerated scalar program",
        "programs": len(data["cases"]), "input_domain": "all signed 64-bit integers",
        "source_sha256": data["source_sha256"], "sources_sha256": before,
        "controls": ["forged-lower", "erased-bounds", "omitted-program", "wrong-source"],
        "unverified": ["arbitrary program sizes and Go syntax", "formal Go operational semantics of the source renderer",
                       "universal correctness or termination of the analyzer implementation", "CLI finding and presentation projection"],
    }, indent=2) + "\n")
    print(f"PASS; evidence: {REPORT}", flush=True)


if __name__ == "__main__":
    try:
        main()
    except (OSError, RuntimeError, ValueError, subprocess.SubprocessError) as error:
        raise SystemExit(str(error)) from error
