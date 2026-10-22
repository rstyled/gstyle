#!/usr/bin/env python3
"""Run the preregistered finite experiment; preserve setup failures as errors."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys


BASE = "98ad52cb09ef1a38090edd9cad9c1248a36cca2c"
ROOT = Path(__file__).resolve().parents[3]
MUTANTS = {
    "external-mutations-ignored": (
        "domain/quality/evaluate.go",
        "if mutation.Provenance == Local {\n\t\t\tcontinue\n\t\t}",
        "if mutation.Provenance == Local || mutation.Provenance == External {\n\t\t\tcontinue\n\t\t}",
    ),
    "local-elements-owned": (
        "adapters/goquality/mutation.go",
        "func referencedOrigin(input quality.Provenance) quality.Provenance {\n"
        "\tif input == quality.External {\n\t\treturn quality.External\n\t}\n\treturn quality.Unknown\n}",
        "func referencedOrigin(input quality.Provenance) quality.Provenance {\n"
        "\tif input == quality.External {\n\t\treturn quality.External\n\t}\n\treturn quality.Local\n}",
    ),
    "mixed-binding-owned": (
        "adapters/goquality/mutation.go",
        "if result != x.bindingValueOrigin(value, seen) {\n\t\t\treturn quality.Unknown\n\t\t}",
        "if result != x.bindingValueOrigin(value, seen) {\n\t\t\treturn quality.Local\n\t\t}",
    ),
}


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def execute(args, cwd, log, env=None):
    result = subprocess.run(args, cwd=cwd, env=env, capture_output=True)
    log.write_bytes(result.stdout + result.stderr)
    return result


def require(args, cwd, log, env=None):
    result = execute(args, cwd, log, env)
    if result.returncode:
        raise RuntimeError(f"setup/validation failed ({result.returncode}): {args}; see {log}")
    return result


def summarize(assessment):
    groups = {}
    for row in assessment["Rows"]:
        family = groups.setdefault(row["Family"], {
            "executions": 0, "runtime_writes": 0, "classified": 0,
            "uncertain": 0, "missing": 0, "incomplete": 0,
            "effects_without_observed_write": 0, "holdout": row["Holdout"],
        })
        family["executions"] += 1
        family["runtime_writes"] += len(row["Writes"] or [])
        family["classified"] += row["Classified"]
        family["uncertain"] += row["Uncertain"]
        family["missing"] += len(row["Missing"] or [])
        family["incomplete"] += not row["Complete"]
        family["effects_without_observed_write"] += bool(row["Effects"]) and not row["Writes"]
    return groups


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, required=True, help="new artifact directory")
    parser.add_argument("--revision", default=BASE, help="frozen analyzer revision; default is the original registered baseline")
    args = parser.parse_args()
    out = args.output.resolve()
    out.mkdir(parents=True, exist_ok=False)
    temp = out / "tmp"
    temp.mkdir()
    env = dict(os.environ, TMPDIR=str(temp))
    source = out / "source"
    source.mkdir()
    revision = subprocess.check_output(["git", "rev-parse", "--verify", args.revision + "^{commit}"], cwd=ROOT, text=True).strip()
    archive = subprocess.check_output(["git", "archive", revision], cwd=ROOT)
    subprocess.run(["tar", "-x", "-C", str(source)], input=archive, check=True)
    env_data = require(["go", "env", "-json", "GOVERSION", "GOOS", "GOARCH", "GOFLAGS", "GOTOOLCHAIN"], ROOT, out / "go-env.log", env)
    manifest = {
        "schema_version": 1, "baseline_revision": revision,
        "checker_revision": subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        "go_environment": json.loads(env_data.stdout),
        "checker_files": {str(p.relative_to(ROOT)): digest(p) for p in [
            ROOT / "examples/quality-hunt/assurance_test.go",
            ROOT / "examples/quality-hunt/assurance_oracle_test.go",
            ROOT / "examples/quality-hunt/check.go", Path(__file__),
        ]},
        "runs": {},
    }
    red = out / "harness-red"
    red.mkdir()
    checker_path = ROOT / "examples/quality-hunt/check.go"
    old = "return changed && result.Complete && !hasMutation(result.Effects)"
    replacement = checker_path.read_text().replace(old, "return changed && result.Complete && len(result.Effects) == 0", 1)
    if replacement == checker_path.read_text():
        raise RuntimeError("old harness predicate anchor missing")
    red_file = red / "check.go.txt"
    red_file.write_text(replacement)
    red_overlay = red / "overlay.json"
    red_overlay.write_text(json.dumps({"Replace": {str(checker_path): str(red_file)}}))
    failed = execute(["go", "test", "-buildvcs=false", "-overlay", str(red_overlay), "./examples/quality-hunt", "-run", "^TestHiddenWriteCannotHideBehindUnrelatedIO$", "-count=1"], ROOT, red / "test.log", env)
    if failed.returncode != 1 or b"unrelated IO hides an observed caller write" not in failed.stdout:
        raise RuntimeError("old harness predicate did not fail for the intended reason")
    require(["go", "test", "-buildvcs=false", "./examples/quality-hunt", "-run", "TestHiddenWriteCannotHideBehindUnrelatedIO|TestGeneratedOwnershipAndSyntaxInvariants|TestFrozenPolicyReportIsNotReevaluated|TestMutationAssuranceRejects", "-count=1", "-v"], ROOT, out / "controls.log", env)
    for name in ["baseline", *MUTANTS]:
        run = out / name
        run.mkdir()
        binary = run / "gstyle"
        build = ["go", "build", "-trimpath", "-buildvcs=false"]
        if name in MUTANTS:
            source_file, before, after = MUTANTS[name]
            original = (source / source_file).read_text()
            if original.count(before) != 1:
                raise RuntimeError(f"mutant anchor is not unique: {name}")
            replacement = run / "mutation.go.txt"
            replacement.write_text(original.replace(before, after, 1))
            overlay = run / "overlay.json"
            overlay.write_text(json.dumps({"Replace": {str(source / source_file): str(replacement)}}))
            build += ["-overlay", str(overlay)]
        require(build + ["-o", str(binary), "."], source, run / "build.log", env)
        run_env = dict(env, GSTYLE_ASSURANCE_BINARY=str(binary), GSTYLE_ASSURANCE_OUTPUT=str(run))
        result = execute(["go", "test", "-buildvcs=false", "./examples/quality-hunt", "-run", "^TestMutationAssurance$", "-count=1", "-v"], ROOT, run / "test.log", run_env)
        assessment_path = run / "assessment.json"
        if not assessment_path.exists():
            raise RuntimeError(f"no semantic assessment for {name}; setup failure: {run / 'test.log'}")
        assessment = json.loads(assessment_path.read_text())
        entry = {"binary_sha256": digest(binary), "exit": result.returncode,
                 "input_sha256": assessment["SourceHash"], "trace_sha256": assessment["TraceHash"],
                 "executions": assessment["Executions"], "classified": assessment["Classified"],
                 "uncertain": assessment["Uncertain"], "missing": assessment["Missing"],
                 "families": summarize(assessment)}
        manifest["runs"][name] = entry
        (out / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
        if name == "baseline" and (result.returncode or assessment["Missing"]):
            raise RuntimeError(f"baseline violated contract; first evidence retained: {run}")
        if name != "baseline" and (result.returncode != 1 or not assessment["Missing"]):
            raise RuntimeError(f"mutant survived or failed for a non-semantic reason: {name}")
        if name != "baseline":
            first = next(row for row in assessment["Rows"] if row["Missing"])
            reduced = run / "reduced"
            reduced_env = dict(run_env, GSTYLE_ASSURANCE_OUTPUT=str(reduced), GSTYLE_ASSURANCE_CASE=first["Name"])
            checked = execute(["go", "test", "-buildvcs=false", "./examples/quality-hunt", "-run", "^TestMutationAssurance$", "-count=1", "-v"], ROOT, run / "reduction.log", reduced_env)
            if not (reduced / "assessment.json").exists():
                raise RuntimeError(f"reduced witness failed setup: {name}")
            small = json.loads((reduced / "assessment.json").read_text())
            if checked.returncode != 1 or not small["Missing"]:
                raise RuntimeError(f"reduced witness lost mismatch: {name}")
            original_runtime = json.loads((run / "runtime-traced.json").read_text())
            reduced_runtime = json.loads((reduced / "runtime-traced.json").read_text())
            for key, observation in reduced_runtime.items():
                old = original_runtime[key]
                if observation["values"] != old["values"] or [w["target"] for w in (observation["writes"] or [])] != [w["target"] for w in (old["writes"] or [])]:
                    raise RuntimeError(f"reduction changed runtime witness: {name}/{key}")
            entry["reduced"] = {"case": first["Name"], "source_sha256": small["SourceHash"], "source_bytes": (reduced / "input.go.txt").stat().st_size, "missing": small["Missing"]}
            (out / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
        print(f"{name}: {assessment['Executions']} executions; {assessment['Missing']} missing write witnesses", flush=True)
    hashes = {(run["input_sha256"], run["trace_sha256"]) for run in manifest["runs"].values()}
    if len(hashes) != 1:
        raise RuntimeError("mutants did not receive identical source and instrumentation")
    print(f"PASS: finite experiment complete; raw evidence: {out}")


if __name__ == "__main__":
    try:
        main()
    except (OSError, RuntimeError, subprocess.CalledProcessError) as error:
        print(f"ERROR: {error}", file=sys.stderr)
        sys.exit(1)
