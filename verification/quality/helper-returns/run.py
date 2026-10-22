#!/usr/bin/env python3
"""Reproduce the fixed helper-return comparison and candidate unsafe variants."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess


ROOT = Path(__file__).resolve().parents[3]
BEFORE = "08ea36d3ed8ad554fa1193a0d89849c6aa3e8b72"
AFTER = "cfb7871933789293fd74b77539efffda553edc49"


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def call(args, cwd, log, env, expected=0):
    result = subprocess.run(args, cwd=cwd, env=env, capture_output=True)
    log.write_bytes(result.stdout + result.stderr)
    if result.returncode != expected:
        raise RuntimeError(f"unexpected exit {result.returncode}: {args}; see {log}")
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, required=True, help="new evidence directory")
    parser.add_argument("--after", default=AFTER, help="frozen candidate commit")
    args = parser.parse_args()
    out = args.output.resolve()
    out.mkdir(parents=True, exist_ok=False)
    temp = out / "tmp"
    temp.mkdir()
    env = dict(os.environ, TMPDIR=str(temp))
    after = subprocess.check_output(["git", "rev-parse", "--verify", args.after + "^{commit}"], cwd=ROOT, text=True).strip()
    before_source = out / "before-source"
    before_source.mkdir()
    archive = subprocess.check_output(["git", "archive", BEFORE], cwd=ROOT)
    subprocess.run(["tar", "-x", "-C", str(before_source)], input=archive, check=True)
    before_binary = out / "before-gstyle"
    call(["go", "build", "-trimpath", "-buildvcs=false", "-o", str(before_binary), "."], before_source, out / "before-build.log", env)

    # Demonstrate the final regression on unchanged production, without removing
    # the candidate tests or substituting a compile failure for the red result.
    red = out / "red"
    red.mkdir()
    replacements = {}
    for filename in ["factories.go", "mutation.go", "storage.go", "pointer_returns.go", "pointer_uses.go"]:
        target = ROOT / "adapters/goquality" / filename
        frozen = before_source / "adapters/goquality" / filename
        replacement = red / (filename + ".txt")
        replacement.write_bytes(frozen.read_bytes() if frozen.exists() else b"package goquality\n")
        replacements[str(target)] = str(replacement)
    overlay = red / "overlay.json"
    overlay.write_text(json.dumps({"Replace": replacements}))
    result = call(["go", "test", "-buildvcs=false", "-overlay", str(overlay), "./adapters/goquality", "-run", "^TestPointerReturnOwnershipKeepsEffects$", "-count=1"], ROOT, red / "test.log", env, expected=1)
    for message in [b"Fresh must retain private storage", b"Borrowed must classify its caller write"]:
        if message not in result.stdout:
            raise RuntimeError("regression fails for a different reason")
    call(["go", "test", "-buildvcs=false", "./adapters/goquality", "-run", "^TestPointerReturn", "-count=1"], ROOT, out / "green.log", env)

    unsafe = out / "unsafe"
    call(["python3", str(ROOT / "verification/quality/mutation-assurance/run.py"), "--revision", after, "--output", str(unsafe)], ROOT, out / "unsafe.log", env)
    after_binary = unsafe / "baseline/gstyle"
    contexts = {
        "matrix": "TestMutationAssurance",
        "development": "TestHelperReturnDevelopmentRuntime",
        "external": "TestHelperReturnExternalEvaluation",
    }
    manifest = {
        "schema_version": 1, "before_revision": BEFORE, "after_revision": after,
        "checker_revision": subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip(),
        "go_environment": json.loads(subprocess.check_output(["go", "env", "-json", "GOVERSION", "GOOS", "GOARCH", "GOFLAGS", "GOTOOLCHAIN"], env=env)),
        "binary_hashes": {"before": sha(before_binary), "after": sha(after_binary)},
        "checker_hashes": {str(p.relative_to(ROOT)): sha(p) for p in [
            ROOT / "examples/quality-hunt/assurance_test.go",
            ROOT / "examples/quality-hunt/assurance_oracle_test.go",
            ROOT / "examples/quality-hunt/helper_return_runtime_test.go",
            ROOT / "examples/quality-hunt/helper_return_evaluation_test.go", Path(__file__),
        ]},
        "external_sources": json.loads((ROOT / "examples/quality-hunt/testdata/helper-return-holdout/sources.json").read_text()),
        "comparisons": {},
    }
    for name, test in contexts.items():
        comparison = {}
        for phase, binary in [("before", before_binary), ("after", after_binary)]:
            directory = out / (phase + "-" + name)
            run_env = dict(env, GSTYLE_ASSURANCE_BINARY=str(binary), GSTYLE_ASSURANCE_OUTPUT=str(directory))
            call(["go", "test", "-buildvcs=false", "./examples/quality-hunt", "-run", "^" + test + "$", "-count=1", "-v"], ROOT, out / (phase + "-" + name + ".log"), run_env)
            assessment = json.loads((directory / "assessment.json").read_text())
            comparison[phase] = {k: assessment[k] for k in ["BinaryHash", "SourceHash", "TraceHash", "Executions", "Classified", "Uncertain", "Missing"]}
        if comparison["before"]["SourceHash"] != comparison["after"]["SourceHash"] or comparison["before"]["TraceHash"] != comparison["after"]["TraceHash"]:
            raise RuntimeError(f"input changed across phases: {name}")
        manifest["comparisons"][name] = comparison
    after_matrix = json.loads((out / "after-matrix/assessment.json").read_text())
    for row in after_matrix["Rows"]:
        if row["Family"] == "borrowed-helper-return" and (row["Uncertain"] or row["Classified"] != 1):
            raise RuntimeError("borrowed helper does not retain classified mutation")
        if row["Family"] == "private-helper-return" and (row["Effects"] or not row["Complete"]):
            raise RuntimeError("fresh helper does not retain private complete storage")
    manifest["unsafe_variants"] = json.loads((unsafe / "manifest.json").read_text())["runs"]

    # Sequential, same-source profiles avoid the initial exploratory overlap.
    benchmark = "adapters/goquality/pointer_benchmark_test.go"
    shutil.copyfile(ROOT / benchmark, before_source / benchmark)
    bench_env = dict(env, GOMAXPROCS="2", GSTYLE_BENCH_ROOT=str(before_source))
    for phase, directory in [("before", before_source), ("after", ROOT)]:
        call(["go", "test", "-buildvcs=false", "./adapters/goquality", "-run", "^$", "-bench", "^BenchmarkHelperReturnExtraction$", "-benchtime=3x", "-count=1", "-benchmem", "-cpuprofile", str(out / (phase + ".cpu.pprof")), "-memprofile", str(out / (phase + ".mem.pprof")), "-o", str(out / (phase + "-bench.test"))], directory, out / (phase + "-benchmark.log"), bench_env)
    (out / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
    print(json.dumps(manifest["comparisons"], indent=2))
    print(f"PASS: finite helper-return study; raw evidence: {out}")


if __name__ == "__main__":
    main()
