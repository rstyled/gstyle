"""Run labeled Go source through the public CLI and record bounded SAST coverage."""
import argparse
from collections import Counter
import json
from pathlib import Path
import shutil
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[2]
HERE = Path(__file__).resolve().parent


def scan(binary, folder, output_format):
    process = subprocess.run([str(binary), "patterns", "--format", output_format, "."],
                             cwd=folder, capture_output=True, text=True, check=True)
    return json.loads(process.stdout)


def evaluate(cases, report, sarif):
    observed = {case["function"]: [] for case in cases}
    for candidate in report.get("candidates") or []:
        security = candidate.get("security")
        if not security:
            continue
        name = candidate["Sites"][0]["Name"].rsplit(".", 1)[-1]
        assert name in observed, f"Unlabeled security finding: {name}"
        assert candidate["analysis_evidence"]["node_ids"], candidate
        assert security["cwe"].startswith("CWE-"), candidate
        observed[name].append(security["rule_id"])
    rows = []
    for case in cases:
        actual = sorted(observed[case["function"]])
        assert actual == sorted(case["expected"]), (case["function"], actual, case)
        wrong = bool(actual) != (case["label"] == "unsafe")
        assert wrong == bool(case.get("gap")), f"Keep gap labels current: {case}"
        rows.append({**case, "observed": actual})
    security_ids = Counter(rule for rules in observed.values() for rule in rules)
    sarif_ids = Counter(result["ruleId"] for result in sarif["runs"][0]["results"]
                        if result["ruleId"].startswith("go/"))
    assert security_ids == sarif_ids, (security_ids, sarif_ids)
    coverage = next(row for row in report["analysis_coverage"] if row["analysis"] == "taint_flow")
    assert sum(coverage[key] for key in ("complete", "partial", "not_evaluated", "unavailable")) == len(cases)
    return {"version": 1, "summary": {
        "unsafe": sum(row["label"] == "unsafe" for row in rows),
        "detected_unsafe": sum(row["label"] == "unsafe" and bool(row["observed"]) for row in rows),
        "safe": sum(row["label"] == "safe" for row in rows),
        "flagged_safe": sum(row["label"] == "safe" and bool(row["observed"]) for row in rows),
        "known_gaps": sum(bool(row.get("gap")) for row in rows),
    }, "analysis_coverage": coverage, "cases": rows}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--write", action="store_true", help="refresh the checked-in baseline")
    args = parser.parse_args()
    cases = json.loads((HERE / "cases.json").read_text())["cases"]
    assert len({row["function"] for row in cases}) == len(cases)
    with tempfile.TemporaryDirectory(prefix="gstyle-sast-") as temporary:
        folder = Path(temporary)
        for source in (HERE / "testdata").iterdir():
            shutil.copyfile(source, folder / source.name)
        binary = folder / "gstyle"
        subprocess.run(["go", "build", "-buildvcs=false", "-o", str(binary), "."], cwd=ROOT, check=True)
        report = scan(binary, folder, "json")
        result = evaluate(cases, report, scan(binary, folder, "sarif"))
    baseline = HERE / "baseline.json"
    if args.write:
        baseline.write_text(json.dumps(result, indent=2) + "\n")
    assert result == json.loads(baseline.read_text()), "Baseline changed: inspect and run make sast-baseline"
    print(json.dumps(result["summary"], sort_keys=True))


if __name__ == "__main__":
    main()
