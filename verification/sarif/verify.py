"""Validate real CLI output against the complete OASIS SARIF schema."""
import json
from pathlib import Path
import subprocess
import tempfile

from jsonschema import Draft4Validator, FormatChecker

ROOT = Path(__file__).resolve().parents[2]
SCHEMA = json.loads(Path(__file__).with_name("schema.json").read_text())
Draft4Validator.check_schema(SCHEMA)
VALIDATOR = Draft4Validator(SCHEMA, format_checker=FormatChecker())


def run_report(binary, cwd, args, expected_exit):
    result = subprocess.run([str(binary), *args], cwd=cwd, capture_output=True, text=True)
    assert result.returncode == expected_exit, (args, result.returncode, result.stderr)
    report = json.loads(result.stdout)
    VALIDATOR.validate(report)
    assert report["version"] == "2.1.0"
    assert report["runs"][0]["tool"]["driver"]["name"] == "gstyle"
    return report["runs"][0]["results"]


with tempfile.TemporaryDirectory(prefix="gstyle-sarif-") as directory:
    work = Path(directory)
    binary = work / "gstyle"
    subprocess.run(["go", "build", "-buildvcs=false", "-o", str(binary), "."], cwd=ROOT, check=True)
    (work / "go.mod").write_text("module example.com/sariftest\n\ngo 1.26.0\n")
    source = work / "input #é.go"
    source.write_text("package sample\nfunc Clean() {}\n")
    for command in ("check", "patterns"):
        assert run_report(binary, work, [command, "--format", "sarif", "."], 0) == []
    source.write_text('''package sample
func Bad(err error) error {
    if err != nil { return nil }
    return err
}
func Many(a, b, c, d, e, f, g, h int) int { return a+b+c+d+e+f+g+h }
''')
    quality = run_report(binary, work, ["check", "--format", "sarif", "."], 1)
    assert any(item["ruleId"] == "fn_params" for item in quality), quality
    patterns = run_report(binary, work, ["patterns", "--format", "sarif", "."], 0)
    assert any(item["ruleId"] == "nilerr" for item in patterns), patterns
    for item in quality + patterns:
        uri = item["locations"][0]["physicalLocation"]["artifactLocation"]["uri"]
        assert "%20%23%C3%A9.go" in uri, uri
    facts = subprocess.run([str(binary), "check", "--format", "facts", "."], cwd=work, capture_output=True, check=True)
    (work / "facts.json").write_bytes(facts.stdout)
    assert run_report(binary, work, ["check", "--facts-in", "facts.json", "--format", "sarif"], 1) == quality
    # The validator must reject malformed SARIF.
    assert not VALIDATOR.is_valid({"version": "2.0", "runs": []})
    assert not VALIDATOR.is_valid({"version": "2.1.0", "runs": [{"tool": {}}]})
print("SARIF schema and CLI checks pass")
