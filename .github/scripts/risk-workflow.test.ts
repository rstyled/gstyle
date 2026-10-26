import { afterEach, expect, test } from "bun:test";
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

type Workflow = {
  permissions: Record<string, string>;
  jobs: Record<string, { steps: { name?: string; run?: string }[] }>;
};

const workflow = Bun.YAML.parse(
  readFileSync(new URL("../workflows/risk.yml", import.meta.url), "utf8"),
) as Workflow;
const temporaryDirectories: string[] = [];

afterEach(() => {
  for (const directory of temporaryDirectories.splice(0)) {
    rmSync(directory, { recursive: true, force: true });
  }
});

function runStep(name: string, env: Record<string, string | undefined>) {
  const step = workflow.jobs["risk-gate"].steps.find((step) => step.name === name);
  if (!step?.run) throw new Error(`Missing workflow step: ${name}`);
  return Bun.spawnSync(
    ["bash", "--noprofile", "--norc", "-e", "-o", "pipefail", "-c", step.run],
    { env, stdout: "pipe", stderr: "pipe" },
  );
}

function runner(status: number, report: string) {
  const directory = mkdtempSync(join(tmpdir(), "gstyle-risk-workflow-"));
  temporaryDirectories.push(directory);
  const eventPath = join(directory, "event.json");
  writeFileSync(eventPath, JSON.stringify({
    number: 270,
    pull_request: { base: { sha: "a".repeat(40) }, head: { sha: "b".repeat(40) } },
  }));
  writeFileSync(join(directory, "gstyle-risk"),
    '#!/bin/sh\nprintf "%s" "$MOCK_REPORT"\nexit "$MOCK_STATUS"\n',
    { mode: 0o755 });
  const env = {
    ...process.env,
    RUNNER_TEMP: directory,
    GITHUB_EVENT_PATH: eventPath,
    GITHUB_STEP_SUMMARY: join(directory, "summary.md"),
    PR_BASE: "a".repeat(40),
    PR_HEAD: "b".repeat(40),
    MOCK_REPORT: report,
    MOCK_STATUS: String(status),
  };
  expect(runStep("Prepare the description report", env).exitCode).toBe(0);
  return { directory, env };
}

test.each([0, 1])("exit %i preserves the full report and gate result", (status) => {
  const report = "**Merge risk: high** ⚠️\n\nMeasured changes.\n";
  const { directory, env } = runner(status, report);
  expect(runStep("Require low merge risk", env).exitCode).toBe(status);
  expect(runStep("Show the measured report", env).exitCode).toBe(0);
  expect(readFileSync(join(directory, "merge-risk-report/merge-risk.md"), "utf8")).toBe(report);
  expect(readFileSync(env.GITHUB_STEP_SUMMARY, "utf8")).toBe(report);
  expect(JSON.parse(readFileSync(join(directory, "merge-risk-report/metadata.json"), "utf8"))).toEqual({
    number: 270, base: env.PR_BASE, head: env.PR_HEAD,
  });
});

test("an operational failure publishes unavailable instead of partial stdout", () => {
  const { directory, env } = runner(2, "partial analysis output");
  expect(runStep("Require low merge risk", env).exitCode).toBe(2);
  expect(runStep("Show the measured report", env).exitCode).toBe(0);
  const report = readFileSync(join(directory, "merge-risk-report/merge-risk.md"), "utf8");
  expect(report).toContain("Merge risk: unavailable");
  expect(report).not.toContain("partial analysis output");
});

test("an empty report cannot pass the gate", () => {
  const { env } = runner(0, "");
  expect(runStep("Require low merge risk", env).exitCode).not.toBe(0);
  expect(runStep("Show the measured report", env).exitCode).toBe(0);
  expect(readFileSync(env.GITHUB_STEP_SUMMARY, "utf8")).toContain("Merge risk: unavailable");
});

test("setup failure still leaves an unavailable report", () => {
  const { env } = runner(0, "unused");
  expect(runStep("Show the measured report", env).exitCode).toBe(0);
  expect(readFileSync(env.GITHUB_STEP_SUMMARY, "utf8")).toContain("Merge risk: unavailable");
});

test("candidate analysis has only read permission", () => {
  expect(workflow.permissions).toEqual({ contents: "read" });
});
