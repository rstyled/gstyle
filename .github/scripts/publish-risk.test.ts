import { describe, expect, test } from "bun:test";
import { mkdtemp, readFile, rm, symlink, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  BODY_LIMIT, END, START, PublishError, publishRisk, runFromEnvironment, updateBody,
  type Fetch, type PublishInput,
} from "./publish-risk";

const BASE = "a".repeat(40);
const HEAD = "b".repeat(40);
const OTHER = "c".repeat(40);
const REPOSITORY = "owner/project";
const SOURCE_URL = "https://github.com/owner/project/actions/runs/100";
const METADATA = { number: 42, base: BASE, head: HEAD };
const REPORT = "**Merge risk: high** ⚠️\n\nChanged exported API.\n";

function sourceEvent() {
  return {
    action: "completed",
    repository: { id: 1, full_name: REPOSITORY },
    workflow_run: {
      id: 100,
      name: "Merge risk",
      path: ".github/workflows/risk.yml",
      event: "pull_request",
      status: "completed",
      conclusion: "failure",
      head_sha: HEAD,
      repository: { id: 1, full_name: REPOSITORY },
      head_repository: { id: 2, full_name: "contributor/fork" },
      pull_requests: [{ number: 42, base: { repo: { id: 1, name: "project", url: "https://api.github.com/repos/owner/project" } } }],
    },
  };
}

function pullRequest(body: string | null = "Author description.\r\n\r\nKeep spaces.  ") {
  return {
    number: 42,
    state: "open",
    body,
    base: { sha: BASE, repo: { id: 1, full_name: REPOSITORY } },
    head: { sha: HEAD, repo: { id: 2, full_name: "contributor/fork" } },
  };
}

function input(overrides: Partial<PublishInput> = {}): PublishInput {
  return {
    repository: REPOSITORY,
    apiURL: "https://api.github.com",
    serverURL: "https://github.com",
    token: "test-token",
    event: sourceEvent(),
    metadata: METADATA,
    report: REPORT,
    ...overrides,
  };
}

type Call = { url: string; init: RequestInit };

function http(...responses: Array<unknown | Error>) {
  const calls: Call[] = [];
  const fetcher: Fetch = async (url, init) => {
    calls.push({ url, init });
    if (responses.length === 0) throw new Error("Unexpected HTTP request");
    const next = responses.shift();
    if (next instanceof Error) throw next;
    if (next instanceof Response) return next;
    return Response.json(next);
  };
  return { calls, fetcher };
}

function patchedBody(calls: Call[]): string {
  const patch = calls.find(call => call.init.method === "PATCH");
  expect(patch).toBeDefined();
  return JSON.parse(patch!.init.body as string).body;
}

function expectNoWrite(calls: Call[]) {
  expect(calls.filter(call => call.init.method !== "GET")).toHaveLength(0);
}

describe("description section", () => {
  test("appends while preserving all author text, including CRLF and trailing spaces", () => {
    const body = "Author text\r\nwith spaces.  \r\n";
    const update = updateBody(body, REPORT, METADATA, SOURCE_URL);
    expect(update.body.startsWith(`${body}\n\n${START}\n${REPORT}`)).toBe(true);
    expect(update.body).toContain(BASE);
    expect(update.body).toContain(HEAD);
    expect(update.body).toContain(SOURCE_URL);
    expect(update.body.endsWith(END)).toBe(true);
    expect(update.compact).toBe(false);
  });

  test("supports an empty description", () => {
    expect(updateBody("", REPORT, METADATA, SOURCE_URL).body.startsWith(START)).toBe(true);
  });

  test("replaces only one existing section, preserving text on both sides exactly", () => {
    const before = "\r\nAuthor text.  \r\n\n";
    const after = "\r\nAfter the report.  \r\n\r\n";
    const body = `${before}${START}\r\nOld report\r\n${END}${after}`;
    const update = updateBody(body, REPORT, METADATA, SOURCE_URL);
    expect(update.body.slice(0, before.length)).toBe(before);
    expect(update.body.slice(-after.length)).toBe(after);
    expect(update.body).not.toContain("Old report");
    expect(update.body).toContain(REPORT);
  });

  test("repeated updates are byte-for-byte identical", () => {
    const once = updateBody("Author text.", REPORT, METADATA, SOURCE_URL);
    expect(updateBody(once.body, REPORT, METADATA, SOURCE_URL)).toEqual(once);
  });

  test.each([
    START,
    END,
    `${END}\n${START}`,
    `${START}\n${START}\n${END}`,
    `${START}\n${END}\n${END}`,
    `${START}\n${END}\n${START}\n${END}`,
    "<!-- gstyle:merge-risk:start --",
    "<!-- gstyle:merge-risk:stop -->",
    "<!-- GSTYLE:MERGE-RISK:start -->",
    `${START}\n<!--gstyle:merge-risk:end-->`,
  ])("rejects malformed/duplicate markers: %s", body => {
    expect(() => updateBody(body, REPORT, METADATA, SOURCE_URL)).toThrow("markers");
  });

  test.each(["", "  \r\n", START, END, "<!--gstyle:merge-risk:other-->"])("rejects empty reports and reserved report markers: %s", report => {
    expect(() => updateBody("author", report, METADATA, SOURCE_URL)).toThrow(PublishError);
  });

  test("validates a reserved marker beyond the size limit before compacting", () => {
    expect(() => updateBody("", `${REPORT}${"x".repeat(BODY_LIMIT)}${START}`, METADATA, SOURCE_URL)).toThrow("reserved markers");
  });
});

describe("description size", () => {
  test("counts the entire UTF-8 body at the exact size boundary", () => {
    const emptySize = Buffer.byteLength(updateBody("", REPORT, METADATA, SOURCE_URL).body);
    const author = "x".repeat(BODY_LIMIT - emptySize - 2);
    const full = updateBody(author, REPORT, METADATA, SOURCE_URL);
    expect(Buffer.byteLength(full.body)).toBe(BODY_LIMIT);
    expect(full.compact).toBe(false);
    // This short report has no smaller compact form, so one more byte must fail.
    expect(() => updateBody(`${author}x`, REPORT, METADATA, SOURCE_URL)).toThrow("no room");
  });

  test.each(["**Merge risk: high** ⚠️", "**Merge risk: unavailable ⚠️**"])("uses the first verdict and provenance when full output exceeds the limit: %s", verdict => {
    const author = "Author description.\r\n";
    const report = `${verdict}\n\n${"details ".repeat(10_000)}`;
    const update = updateBody(author, report, METADATA, SOURCE_URL);
    expect(update.compact).toBe(true);
    expect(update.body.startsWith(author)).toBe(true);
    expect(update.body).toContain(verdict);
    expect(update.body).not.toContain("details details");
    expect(update.body).toContain(BASE);
    expect(update.body).toContain(HEAD);
    expect(update.body).toContain(SOURCE_URL);
    expect(Buffer.byteLength(update.body)).toBeLessThanOrEqual(BODY_LIMIT);
    expect(updateBody(update.body, report, METADATA, SOURCE_URL)).toEqual(update);
  });

  test("uses UTF-8 bytes for emoji instead of JS string length", () => {
    const report = `${REPORT}${"⚠️".repeat(12_000)}`;
    expect(report.length).toBeLessThan(BODY_LIMIT);
    expect(Buffer.byteLength(report)).toBeGreaterThan(BODY_LIMIT);
    expect(updateBody("", report, METADATA, SOURCE_URL).compact).toBe(true);
  });

  test("does not trim author text to fit even the compact section", () => {
    const author = "🔒".repeat(16_384);
    expect(() => updateBody(author, "**Merge risk: low**", METADATA, SOURCE_URL)).toThrow("author text is preserved");
  });
});

describe("publisher HTTP flow", () => {
  test("publishes a failed high-risk result using the current API body", async () => {
    const event = { ...sourceEvent(), pull_request: { body: "Stale event body." } };
    const current = pullRequest("Current author edit.\r\n");
    const api = http(current, {});
    const result = await publishRisk(input({ event }), api.fetcher);
    expect(result).toEqual({ status: "updated", number: 42, run: 100, compact: false });
    expect(api.calls.map(call => call.init.method)).toEqual(["GET", "PATCH"]);
    expect(api.calls.map(call => call.url)).toEqual([
      "https://api.github.com/repos/owner/project/pulls/42",
      "https://api.github.com/repos/owner/project/pulls/42",
    ]);
    expect(patchedBody(api.calls)).toContain("Current author edit.\r\n");
    expect(patchedBody(api.calls)).toContain(REPORT);
    expect(patchedBody(api.calls)).not.toContain("Stale event body.");
    expect(api.calls[1]!.init.headers).toMatchObject({ Authorization: "Bearer test-token", Accept: "application/vnd.github+json" });
    expect(api.calls[1]!.init.redirect).toBe("error");
  });

  test("does not PATCH an identical body", async () => {
    const body = updateBody("author", REPORT, METADATA, SOURCE_URL).body;
    const api = http(pullRequest(body));
    expect((await publishRisk(input(), api.fetcher)).status).toBe("unchanged");
    expectNoWrite(api.calls);
  });

  test("accepts a null PR description and publishes an unavailable fallback", async () => {
    const report = "**Merge risk: unavailable ⚠️**\nAnalyzer failed.";
    const api = http(pullRequest(null), {});
    await publishRisk(input({ report }), api.fetcher);
    expect(patchedBody(api.calls).startsWith(START)).toBe(true);
    expect(patchedBody(api.calls)).toContain(report);
  });

  test("replaces the current existing block, including an author edit after the block", async () => {
    const api = http(pullRequest(`before\r\n${START}\nold\n${END}\r\nLatest author edit.  `), {});
    await publishRisk(input(), api.fetcher);
    expect(patchedBody(api.calls)).toBe(updateBody(`before\r\n${START}\nold\n${END}\r\nLatest author edit.  `, REPORT, METADATA, SOURCE_URL).body);
  });

  test.each([
    ["closed", (pull: ReturnType<typeof pullRequest>) => { pull.state = "closed"; }, "pull_request_closed"],
    ["head", (pull: ReturnType<typeof pullRequest>) => { pull.head.sha = OTHER; }, "head_sha_mismatch"],
    ["base", (pull: ReturnType<typeof pullRequest>) => { pull.base.sha = OTHER; }, "base_sha_mismatch"],
    ["repository", (pull: ReturnType<typeof pullRequest>) => { pull.base.repo.full_name = "other/project"; }, "base_repository_mismatch"],
  ] as const)("skips a PR with changed %s", async (_name, change, reason) => {
    const pull = pullRequest();
    change(pull);
    const api = http(pull);
    expect(await publishRisk(input(), api.fetcher)).toMatchObject({ status: "skipped", reason });
    expectNoWrite(api.calls);
  });

  test("marker and size failures never PATCH", async () => {
    for (const body of [START, "x".repeat(BODY_LIMIT)]) {
      const api = http(pullRequest(body));
      await expect(publishRisk(input(), api.fetcher)).rejects.toBeInstanceOf(PublishError);
      expectNoWrite(api.calls);
    }
  });

  test("uses the configured GitHub Enterprise URLs", async () => {
    const api = http(pullRequest(), {});
    await publishRisk(input({ apiURL: "https://git.example/api/v3/", serverURL: "https://git.example/" }), api.fetcher);
    expect(api.calls[0]!.url).toBe("https://git.example/api/v3/repos/owner/project/pulls/42");
    expect(patchedBody(api.calls)).toContain("https://git.example/owner/project/actions/runs/100");
  });

  test.each([403, 404, 422, 500])("propagates GET API failure %s without a write", async status => {
    const api = http(new Response("not logged", { status }));
    await expect(publishRisk(input(), api.fetcher)).rejects.toThrow(`status ${status}`);
    expectNoWrite(api.calls);
  });

  test("propagates a PATCH API failure", async () => {
    const api = http(pullRequest(), new Response("denied", { status: 403 }));
    await expect(publishRisk(input(), api.fetcher)).rejects.toThrow("PATCH /repos/owner/project/pulls/42 failed with status 403");
  });

  test("propagates network and response JSON errors", async () => {
    const api = http(new Error("connection failed"));
    await expect(publishRisk(input(), api.fetcher)).rejects.toThrow("connection failed");
    const invalidJSON = http(new Response("not json"));
    await expect(publishRisk(input(), invalidJSON.fetcher)).rejects.toThrow();
    expectNoWrite(invalidJSON.calls);
  });

  test("rejects an incorrect response number or body type", async () => {
    for (const pull of [{ ...pullRequest(), number: 43 }, { ...pullRequest(), body: 17 }]) {
      const api = http(pull);
      await expect(publishRisk(input(), api.fetcher)).rejects.toBeInstanceOf(PublishError);
      expectNoWrite(api.calls);
    }
  });
});

describe("source and artifact binding", () => {
  test.each([
    ["action", (event: ReturnType<typeof sourceEvent>) => { event.action = "requested"; }],
    ["status", (event: ReturnType<typeof sourceEvent>) => { event.workflow_run.status = "in_progress"; }],
    ["name", (event: ReturnType<typeof sourceEvent>) => { event.workflow_run.name = "Other workflow"; }],
    ["path", (event: ReturnType<typeof sourceEvent>) => { event.workflow_run.path = ".github/workflows/other.yml"; }],
    ["event", (event: ReturnType<typeof sourceEvent>) => { event.workflow_run.event = "push"; }],
    ["event repo", (event: ReturnType<typeof sourceEvent>) => { event.repository.full_name = "other/project"; }],
    ["run repo", (event: ReturnType<typeof sourceEvent>) => { event.workflow_run.repository.full_name = "other/project"; }],
    ["run repo ID", (event: ReturnType<typeof sourceEvent>) => { event.workflow_run.repository.id = 2; }],
    ["run head", (event: ReturnType<typeof sourceEvent>) => { event.workflow_run.head_sha = OTHER; }],
    ["run ID", (event: ReturnType<typeof sourceEvent>) => { event.workflow_run.id = 0; }],
    ["unrelated PR", (event: ReturnType<typeof sourceEvent>) => { event.workflow_run.pull_requests[0]!.number = 99; }],
  ] as const)("rejects mismatched source %s before making any HTTP request", async (_name, change) => {
    const event = sourceEvent();
    change(event);
    const api = http();
    await expect(publishRisk(input({ event }), api.fetcher)).rejects.toBeInstanceOf(PublishError);
    expect(api.calls).toHaveLength(0);
  });

  test.each([
    null, {}, { ...METADATA, number: 0 }, { ...METADATA, number: -1 }, { ...METADATA, number: 1.1 },
    { ...METADATA, number: "42" }, { ...METADATA, number: Number.MAX_SAFE_INTEGER + 1 },
    { ...METADATA, base: "main" }, { ...METADATA, head: "b".repeat(41) },
    { ...METADATA, head: `$(touch /tmp/never)${HEAD}` }, { ...METADATA, number: 99 },
  ])("rejects invalid or unbound artifact metadata: %j", async metadata => {
    const api = http();
    await expect(publishRisk(input({ metadata }), api.fetcher)).rejects.toBeInstanceOf(PublishError);
    expect(api.calls).toHaveLength(0);
  });

  test("rejects absent source PR array instead of treating it as an empty fork list", async () => {
    const event = sourceEvent();
    const { pull_requests: _unused, ...run } = event.workflow_run;
    const api = http();
    await expect(publishRisk(input({ event: { ...event, workflow_run: run } }), api.fetcher)).rejects.toThrow("pull_requests");
    expect(api.calls).toHaveLength(0);
  });

  test("ignores an artifact source URL and constructs the URL from trusted run identity", async () => {
    const api = http(pullRequest(), {});
    await publishRisk(input({ metadata: { ...METADATA, sourceURL: "https://evil.example" } }), api.fetcher);
    expect(patchedBody(api.calls)).toContain(SOURCE_URL);
    expect(patchedBody(api.calls)).not.toContain("evil.example");
  });
});

describe("fork source with no PR list", () => {
  function forkInput() {
    const event = sourceEvent();
    event.workflow_run.pull_requests = [];
    return input({ event });
  }

  test("binds the number through the commit association and reads the latest PR immediately before PATCH", async () => {
    const otherPull = { ...pullRequest(), number: 7 };
    const api = http([otherPull, pullRequest("Old cached association body")], pullRequest("New author body"), {});
    await publishRisk(forkInput(), api.fetcher);
    expect(api.calls.map(call => call.init.method)).toEqual(["GET", "GET", "PATCH"]);
    expect(api.calls[0]!.url).toBe(`https://api.github.com/repos/owner/project/commits/${HEAD}/pulls?per_page=100&page=1`);
    expect(patchedBody(api.calls)).toContain("New author body");
    expect(patchedBody(api.calls)).not.toContain("Old cached association body");
  });

  test("follows association pagination without trusting the artifact number alone", async () => {
    const firstPage = Array.from({ length: 100 }, (_, i) => ({ ...pullRequest(), number: 1_000 + i }));
    const api = http(firstPage, [pullRequest()], pullRequest(), {});
    await publishRisk(forkInput(), api.fetcher);
    expect(api.calls[1]!.url).toEndWith("?per_page=100&page=2");
    expect(patchedBody(api.calls)).toContain(REPORT);
  });

  test.each([
    { associations: [] },
    { associations: [{ ...pullRequest(), number: 7 }] },
    { associations: [{ ...pullRequest(), head: { sha: OTHER } }] },
    { associations: [{ ...pullRequest(), base: { sha: OTHER, repo: { full_name: REPOSITORY } } }] },
    { associations: [{ ...pullRequest(), base: { sha: BASE, repo: { full_name: "other/project" } } }] },
    { associations: [{ ...pullRequest(), state: "closed" }] },
  ])("rejects an absent or mismatched exact-head association: %j", async ({ associations }) => {
    const api = http(associations);
    await expect(publishRisk(forkInput(), api.fetcher)).rejects.toThrow("associat");
    expectNoWrite(api.calls);
  });

  test.each(["head", "base", "closed"])("rechecks the latest %s after a valid association", async field => {
    const latest = pullRequest("Author edits after the association response");
    if (field === "head") latest.head.sha = OTHER;
    if (field === "base") latest.base.sha = OTHER;
    if (field === "closed") latest.state = "closed";
    const api = http([pullRequest()], latest);
    expect((await publishRisk(forkInput(), api.fetcher)).status).toBe("skipped");
    expectNoWrite(api.calls);
  });

  test("propagates association API errors without fetching or patching the PR", async () => {
    const api = http(new Response("denied", { status: 403 }));
    await expect(publishRisk(forkInput(), api.fetcher)).rejects.toThrow("status 403");
    expect(api.calls).toHaveLength(1);
    expectNoWrite(api.calls);
  });
});

describe("CLI inputs", () => {
  test("reads artifact files only as data, with current-body HTTP behavior", async () => {
    const directory = await mkdtemp(join(tmpdir(), "gstyle-risk-publisher-"));
    const eventPath = join(directory, "event.json");
    const report = `${REPORT}\n\`$(touch /tmp/gstyle-never-run)\`\n<script>throw new Error('inert')</script>`;
    try {
      await Promise.all([
        writeFile(eventPath, JSON.stringify(sourceEvent())),
        writeFile(join(directory, "metadata.json"), JSON.stringify(METADATA)),
        writeFile(join(directory, "merge-risk.md"), report),
      ]);
      const api = http(pullRequest("Current body"), {});
      const result = await runFromEnvironment({
        GITHUB_EVENT_PATH: eventPath, RISK_REPORT_DIR: directory, GITHUB_REPOSITORY: REPOSITORY,
        GITHUB_API_URL: "https://api.github.com", GITHUB_SERVER_URL: "https://github.com", GH_TOKEN: "test-token",
      }, api.fetcher);
      expect(result.status).toBe("updated");
      expect(patchedBody(api.calls)).toContain(report);
      expect(await readFile(join(directory, "merge-risk.md"), "utf8")).toBe(report);
    } finally {
      await rm(directory, { recursive: true, force: true });
    }
  });

  test("requires explicit environment values", async () => {
    const api = http();
    await expect(runFromEnvironment({}, api.fetcher)).rejects.toThrow("GITHUB_EVENT_PATH is required");
    expect(api.calls).toHaveLength(0);
  });

  test("rejects an artifact symlink without reading or publishing its target", async () => {
    const directory = await mkdtemp(join(tmpdir(), "gstyle-risk-publisher-"));
    const eventPath = join(directory, "event.json");
    try {
      await Promise.all([
        writeFile(eventPath, JSON.stringify(sourceEvent())),
        writeFile(join(directory, "metadata.json"), JSON.stringify(METADATA)),
      ]);
      await symlink(eventPath, join(directory, "merge-risk.md"));
      const api = http();
      await expect(runFromEnvironment({
        GITHUB_EVENT_PATH: eventPath, RISK_REPORT_DIR: directory, GITHUB_REPOSITORY: REPOSITORY,
        GITHUB_API_URL: "https://api.github.com", GITHUB_SERVER_URL: "https://github.com", GH_TOKEN: "test-token",
      }, api.fetcher)).rejects.toThrow();
      expect(api.calls).toHaveLength(0);
    } finally {
      await rm(directory, { recursive: true, force: true });
    }
  });

  test.each([
    { repository: "owner/../project" }, { repository: "owner/.." },
    { apiURL: "https://user:password@api.github.com" }, { apiURL: "http://api.github.com" },
    { apiURL: "not a URL" }, { serverURL: "https://github.com/?query=value" }, { token: "" },
  ])("rejects invalid HTTP settings: %j", async overrides => {
    const api = http();
    await expect(publishRisk(input(overrides), api.fetcher)).rejects.toBeInstanceOf(PublishError);
    expect(api.calls).toHaveLength(0);
  });

  test("CLI errors are structured JSON and exit nonzero", async () => {
    const child = Bun.spawn([process.execPath, join(import.meta.dir, "publish-risk.ts")], {
      env: {}, stdout: "pipe", stderr: "pipe",
    });
    const [exitCode, stdout, stderr] = await Promise.all([child.exited, new Response(child.stdout).text(), new Response(child.stderr).text()]);
    expect(exitCode).toBe(1);
    expect(stdout).toBe("");
    expect(JSON.parse(stderr)).toEqual({ level: "error", event: "merge_risk_publish", code: "invalid_environment", message: "GITHUB_EVENT_PATH is required" });
  });
});
