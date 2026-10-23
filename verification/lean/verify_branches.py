#!/usr/bin/env python3
"""Check arbitrary-depth source branches with a proved structural range validator."""
import argparse
import hashlib
import json
from pathlib import Path
import re

import verify
import verify_programs as programs

ROOT, PROJECT = programs.ROOT, programs.PROJECT
REPORT = ROOT / "dist/verification/branches"
FOUNDATION = PROJECT / "Gstyle/Proof/Programs/Structural.lean"
PREFIX = "Perennial.GstylePrograms."
THEOREMS = {PREFIX + name for name in (
    "inverse_holds", "restrict_sound", "covered_sound", "checkTree_sound",
    "checkPrecision_exact", "checkReport_sound", "certificateOfCheck", "checked_interval_sound")}


def rejected(output, path, expected):
    declarations = Path(path).read_text().splitlines()
    pattern = re.compile(re.escape(str(path)) + r":(\d+):\d+: error: (.*)")
    found = False
    for line in output.splitlines():
        if ": error:" not in line and not line.startswith("error:"):
            continue
        match = pattern.fullmatch(line)
        if match is None:
            return False
        number, diagnostic = match.groups()
        allowed = ("omega could not prove the goal:", "unsolved goals")
        if diagnostic not in allowed and not diagnostic.startswith(
                ("Tactic `decide` proved that the proposition", "Tactic `rfl` failed:")):
            return False
        names = [row.split()[1] for row in declarations[:int(number)] if row.startswith("theorem ")]
        if not names or names[-1] not in expected:
            return False
        found = True
    return found


def check_file(path, log, *, failure=()):
    import os
    import subprocess
    result = subprocess.run(["lake", "env", "lean", "-j2", str(path)], cwd=PROJECT,
                            env=dict(os.environ, LEAN_NUM_THREADS="2"), timeout=1800,
                            text=True, capture_output=True)
    output = result.stdout + result.stderr
    log.parent.mkdir(parents=True, exist_ok=True)
    log.write_text(output)
    if failure:
        if result.returncode == 0 or not rejected(output, path, failure):
            raise RuntimeError(f"expected logical rejection is absent: {log}")
    elif result.returncode != 0:
        raise RuntimeError(f"Lean check failed: {log}")
    return output


def collect(directory, source=None, overlay=None):
    command = ["go", "run"]
    if overlay:
        command.append("-overlay=" + str(overlay))
    command.extend(["./verification/programs", "--structural", "--output", str(directory)])
    command.extend(["--source", str(source)] if source else ["--controls"])
    programs.run(command, ROOT, directory / "collection.log")
    return programs.validate_artifacts(directory)


def check_capture(directory, source=None, overlay=None):
    data = collect(directory, source, overlay)
    output = check_file(directory / "Certificate.lean", directory / "proof.log")
    names = {"GeneratedPrograms.source_file"}
    names.update("GeneratedPrograms." + kind + c["name"]
                 for c in data["cases"] for kind in ("checked", "source"))
    names.update("GeneratedPrograms.complete" + str(i) for i in range(len(data["suites"] or [])))
    verify.check_axioms(output, names)
    return data


def production_controls():
    target = ROOT / "domain/runtimeguard/flow.go"
    original = target.read_text()
    statement = "report.Calls = append(report.Calls, Call{Node: ref, FirstArgument: a.view.value(a.view.operand(ref, position), state)})"
    if original.count(statement) != 1:
        raise RuntimeError("production observation target is not unique")
    for name, value in {"forged-lower": "Value{HasLower: true, Lower: 1}", "erased-bounds": "Value{}"}.items():
        directory = REPORT / "controls" / name
        directory.mkdir(parents=True, exist_ok=True)
        changed = directory / "flow.go.txt"
        changed.write_text(original.replace(statement, statement +
                           "\nreport.Calls[len(report.Calls)-1].FirstArgument = " + value))
        overlay = directory / "overlay.json"
        overlay.write_text(json.dumps({"Replace": {str(target): str(changed)}}))
        data = collect(directory, overlay=overlay)
        check_file(directory / "Certificate.lean", directory / "proof.log",
                   failure={"checked" + c["name"] for c in data["cases"]})
        print("REJECTED production " + name, flush=True)


def checker_controls():
    original = FOUNDATION.read_text()
    changes = {
        "wrong-inverse": ("| .lt => .ge | .le => .gt", "| .lt => .le | .le => .gt", {"inverse_holds"}),
        "wrong-endpoint": ("decide (r.lower ≤ d.lower)", "decide (r.lower ≤ d.upper)", {"covered_sound"}),
        "lost-inputs": ("⟨-(2^63), 2^63 - 1⟩", "⟨0, 2^63 - 1⟩", {"checkReport_sound"}),
    }
    for name, (before, after, expected) in changes.items():
        if original.count(before) != 1:
            raise RuntimeError("checker mutation target is not unique: " + name)
        directory = REPORT / "controls" / name
        directory.mkdir(parents=True, exist_ok=True)
        path = directory / "Structural.lean"
        path.write_text(original.replace(before, after))
        check_file(path, directory / "proof.log", failure=expected)
        print("REJECTED checker " + name, flush=True)


def binding_controls(data):
    import copy
    import shutil
    baseline = REPORT / "baseline"
    expected = {"changed-source": "captured source hash", "changed-report": "certificate range",
                "missing-program": "captured program sources"}
    for name, diagnostic in expected.items():
        directory = REPORT / "controls" / name
        directory.mkdir(parents=True, exist_ok=True)
        for file in ("input.go.txt", "observations.json", "Certificate.lean"):
            shutil.copy2(baseline / file, directory / file)
        changed = copy.deepcopy(data)
        if name == "changed-source":
            (directory / "input.go.txt").write_text("altered source")
        elif name == "changed-report":
            changed["cases"][0]["range"]["Lower"] += 1
        else:
            changed["cases"].pop()
        (directory / "observations.json").write_text(json.dumps(changed))
        try:
            programs.validate_artifacts(directory)
        except RuntimeError as error:
            if diagnostic not in str(error):
                raise RuntimeError("unexpected binding failure: " + name) from error
            (directory / "rejection.log").write_text(str(error) + "\n")
        else:
            raise RuntimeError("artifact mutation accepted: " + name)
        print("REJECTED binding " + name, flush=True)


def main():
    REPORT.mkdir(parents=True, exist_ok=True)
    (REPORT / "result.json").unlink(missing_ok=True)
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", type=Path, default=ROOT / "verification/programs/testdata/branches.go.txt")
    args = parser.parse_args()
    source = args.source.resolve()
    source_hash = hashlib.sha256(source.read_bytes()).hexdigest()
    before = programs.fingerprint()
    programs.run(["lake", "build", "Gstyle.Proof.Programs.Structural"], PROJECT, REPORT / "build.log")
    verify.check_dependencies(REPORT)
    audit = programs.run(["lake", "env", "lean", "AuditBranches.lean"], PROJECT, REPORT / "axioms.log")
    verify.check_axioms(audit, THEOREMS)
    data = check_capture(REPORT / "baseline", source=source)
    check_capture(REPORT / "control-baseline")
    production_controls()
    checker_controls()
    binding_controls(data)
    if programs.fingerprint() != before or hashlib.sha256(source.read_bytes()).hexdigest() != source_hash:
        raise RuntimeError("verification inputs changed during the run")
    (REPORT / "result.json").write_text(json.dumps({
        "verified": "structural checker sound for all finite branch trees; supplied-source production captures accepted",
        "source": str(source), "source_sha256": source_hash, "captured_programs": len(data["cases"]),
        "input_domain": "all signed 64-bit integers", "grammar_depth_bound": None,
        "sources_sha256": before,
        "controls": ["forged-lower", "erased-bounds", "wrong-inverse", "wrong-endpoint", "lost-inputs",
                     "changed-source", "changed-report", "missing-program"],
        "unverified": ["unconditional correctness or termination of the production CFG worklist",
                       "formal Go semantics of the canonical source renderer", "arbitrary Go syntax", "CLI projection"],
    }, indent=2) + "\n")
    print("PASS arbitrary-depth structural contract and production captures; " + str(REPORT), flush=True)


if __name__ == "__main__":
    import subprocess
    try:
        main()
    except (OSError, RuntimeError, ValueError, subprocess.SubprocessError) as error:
        raise SystemExit(str(error)) from error
