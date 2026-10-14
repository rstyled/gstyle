#!/usr/bin/env python3
"""Prove the production output decision and reject five logic mutations.

This does not prove the Go AST/type adapter or complete detector soundness.
"""
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys

IMAGE = "ghcr.io/viperproject/gobra@sha256:5e96431b06c2e6209aa767df9153bb4cf8b2620598598f1d7d3f3a74c09c14f9"
ROOT = Path(__file__).resolve().parent.parent
SOURCE = ROOT / "adapters/gopatterns/outputdecision.go"
OUTPUT = ROOT / "dist/verification/unchecked-output"
CONTRACT = "result == (input.Formatted && !input.Memory && (input.Standalone || (input.Values == 1 && input.Results == 2 && input.ErrorBlank)))"
MUTATIONS = {
    "unrecognized-function": ("!input.Formatted || input.Memory", "input.Memory"),
    "memory-writer": ("!input.Formatted || input.Memory", "!input.Formatted"),
    "wrong-result-count": ("input.Results == 2", "input.Results == 1"),
    "handled-error": (" && input.ErrorBlank", ""),
    "missed-standalone": ("if input.Standalone {\n\t\treturn true", "if input.Standalone {\n\t\treturn false"),
}


def run_verifier(source, name):
    container = f"gstyle-output-proof-{os.getpid()}-{name}"
    command = [
        "docker", "run", "--name", container, "--rm", "--network", "none", "--read-only",
        "--cpus", "2", "--memory", "2g", "--pids-limit", "256",
        "--tmpfs", "/tmp:rw,size=512m",
        "--mount", f"type=bind,source={source},target=/proof/outputdecision.go,readonly",
        IMAGE, "-i", "/proof/outputdecision.go", "--gobraDirectory", "/tmp",
        "--overflow", "--packageTimeout", "45s",
    ]
    try:
        result = subprocess.run(command, capture_output=True, text=True, timeout=60)
    except subprocess.TimeoutExpired:
        subprocess.run(["docker", "rm", "--force", container], capture_output=True)
        raise
    log = result.stdout + result.stderr
    (OUTPUT / (name + ".log")).write_text(log)
    return result.returncode, log


def verify():
    OUTPUT.mkdir(parents=True, exist_ok=True)
    # A failed rerun must not leave an older success report looking current.
    (OUTPUT / "report.json").unlink(missing_ok=True)
    source = SOURCE.read_text()
    specification, body = source.split("func discardsOutputDecision", 1)
    # Do not let weakened contracts or assumed bodies make this gate pass.
    if CONTRACT not in specification or "// @ decreases" not in specification:
        raise RuntimeError("expected exact result contract and termination obligation")
    for forbidden in ("trusted", "assume", "requires"):
        if forbidden in source:
            raise RuntimeError("proof source must not add assumptions: " + forbidden)
    OUTPUT.mkdir(parents=True, exist_ok=True)
    subprocess.run(["docker", "pull", IMAGE], check=True)
    code, log = run_verifier(SOURCE, "production")
    if code != 0 or "Gobra found 0 errors." not in log:
        raise RuntimeError("production proof failed; see " + str(OUTPUT / "production.log"))
    print("PASS: exact production decision contract and termination", flush=True)
    rejected = []
    for name, (old, new) in MUTATIONS.items():
        if body.count(old) != 1:
            raise RuntimeError("mutation no longer matches exactly once: " + name)
        path = OUTPUT / (name + ".go.txt")
        path.write_text(specification + "func discardsOutputDecision" + body.replace(old, new, 1))
        code, log = run_verifier(path, name)
        if code == 0 or "Postcondition might not hold" not in log:
            raise RuntimeError("mutant must fail its postcondition, not setup or parsing: " + name)
        rejected.append(name)
        print("PASS: postcondition rejects " + name, flush=True)
    report = {
        "image": IMAGE,
        "source": str(SOURCE.relative_to(ROOT)),
        "source_sha256": hashlib.sha256(SOURCE.read_bytes()).hexdigest(),
        "contract": CONTRACT,
        "decision_core_verified": True,
        "scope": "production decision only; AST/type adapter remains unverified",
        "rejected_mutations": rejected,
        "unverified": ["AST/type fact extraction", "candidate construction", "Go compiler and stdlib semantics", "Gobra/Viper/Z3 implementation"],
    }
    (OUTPUT / "report.json").write_text(json.dumps(report, indent=2) + "\n")


if __name__ == "__main__":
    try:
        verify()
    except (OSError, ValueError, RuntimeError, subprocess.SubprocessError) as error:
        print("verification failed: " + str(error), file=sys.stderr)
        sys.exit(1)
