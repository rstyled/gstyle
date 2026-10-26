import { constants } from "node:fs";
import { open, readFile } from "node:fs/promises";
import { join } from "node:path";

export const START = "<!-- gstyle:merge-risk:start -->";
export const END = "<!-- gstyle:merge-risk:end -->";
export const BODY_LIMIT = 65_536;

type ObjectValue = Record<string, unknown>;
type Metadata = { number: number; base: string; head: string };
type SourceRun = { id: number; head: string; pullNumbers: number[] };
type PullRequest = Metadata & { state: string; repository: string; body: string };
type Settings = { repository: string; apiURL: string; serverURL: string; token: string };
export type Fetch = (url: string, init: RequestInit) => Promise<Response>;
export type PublishInput = Settings & { event: unknown; metadata: unknown; report: string };
export type PublishResult = {
  status: "updated" | "unchanged" | "skipped";
  number: number;
  run: number;
  reason?: string;
  compact?: boolean;
};

export class PublishError extends Error {
  constructor(public readonly code: string, message: string) {
    super(message);
    this.name = "PublishError";
  }
}

function requireValue(condition: unknown, code: string, message: string): asserts condition {
  if (!condition) throw new PublishError(code, message);
}

function object(value: unknown, label: string): ObjectValue {
  requireValue(value !== null && typeof value === "object" && !Array.isArray(value), "invalid_data", `${label} must be an object`);
  return value as ObjectValue;
}

function positiveInteger(value: unknown, label: string): number {
  requireValue(Number.isSafeInteger(value) && (value as number) > 0, "invalid_data", `${label} must be a positive integer`);
  return value as number;
}

function sha(value: unknown, label: string): string {
  requireValue(typeof value === "string" && /^[a-fA-F0-9]{40}$/.test(value), "invalid_data", `${label} must be a 40-character SHA`);
  return value.toLowerCase();
}

function repository(value: unknown): string {
  requireValue(typeof value === "string" && /^[a-zA-Z0-9_.-]+\/[a-zA-Z0-9_.-]+$/.test(value), "invalid_data", "Repository must be owner/name");
  requireValue(!value.split("/").some(part => part === "." || part === ".."), "invalid_data", "Repository has an invalid path segment");
  return value.toLowerCase();
}

function baseURL(value: string): string {
  let url: URL;
  try {
    url = new URL(value);
  } catch {
    throw new PublishError("invalid_environment", "GitHub URL is invalid");
  }
  requireValue(url.protocol === "https:" && !url.username && !url.password && !url.search && !url.hash, "invalid_environment", "GitHub URL must use HTTPS without credentials, query, or fragment");
  return url.href.replace(/\/+$/, "");
}

function parseMetadata(value: unknown): Metadata {
  const data = object(value, "metadata");
  return { number: positiveInteger(data.number, "metadata.number"), base: sha(data.base, "metadata.base"), head: sha(data.head, "metadata.head") };
}

function parseSource(eventValue: unknown, expectedRepository: string): SourceRun {
  const event = object(eventValue, "event");
  const run = object(event.workflow_run, "workflow_run");
  requireValue(event.action === "completed" && run.status === "completed", "invalid_source", "Source workflow run must be completed");
  requireValue(run.event === "pull_request" && run.name === "Merge risk" && run.path === ".github/workflows/risk.yml", "invalid_source", "Source workflow event, name, or path does not match");
  const eventRepository = object(event.repository, "event.repository");
  const runRepository = object(run.repository, "workflow_run.repository");
  requireValue(repository(eventRepository.full_name) === expectedRepository && repository(runRepository.full_name) === expectedRepository, "invalid_source", "Source repository does not match GITHUB_REPOSITORY");
  requireValue(positiveInteger(eventRepository.id, "event.repository.id") === positiveInteger(runRepository.id, "workflow_run.repository.id"), "invalid_source", "Source repository ID does not match");
  requireValue(Array.isArray(run.pull_requests), "invalid_source", "Source pull_requests must be an array");
  const pullNumbers = run.pull_requests.map(value => positiveInteger(object(value, "source pull request").number, "source pull request number"));
  return { id: positiveInteger(run.id, "workflow_run.id"), head: sha(run.head_sha, "workflow_run.head_sha"), pullNumbers };
}

function parsePull(value: unknown): PullRequest {
  const data = object(value, "pull request");
  const base = object(data.base, "pull request base");
  const head = object(data.head, "pull request head");
  const baseRepository = object(base.repo, "pull request base repository");
  requireValue(data.state === "open" || data.state === "closed", "invalid_data", "Pull request state is invalid");
  requireValue(data.body === null || typeof data.body === "string", "invalid_data", "Pull request body must be text or null");
  return {
    number: positiveInteger(data.number, "pull request number"),
    base: sha(base.sha, "pull request base SHA"),
    head: sha(head.sha, "pull request head SHA"),
    repository: repository(baseRepository.full_name),
    state: data.state,
    body: data.body ?? "",
  };
}

function staleReason(pull: PullRequest, metadata: Metadata, expectedRepository: string): string | undefined {
  if (pull.state !== "open") return "pull_request_closed";
  if (pull.repository !== expectedRepository) return "base_repository_mismatch";
  if (pull.head !== metadata.head) return "head_sha_mismatch";
  if (pull.base !== metadata.base) return "base_sha_mismatch";
  return undefined;
}

function validateReport(report: string): void {
  requireValue(typeof report === "string" && report.trim().length > 0, "invalid_report", "Merge-risk report must be nonempty text");
  requireValue(!/<!--\s*gstyle:merge-risk\b/i.test(report), "invalid_report", "Merge-risk report contains reserved markers");
}

function bodyParts(body: string): { before: string; after: string } {
  const reserved = body.match(/<!--\s*gstyle:merge-risk\b/gi) ?? [];
  const start = body.indexOf(START);
  const end = body.indexOf(END);
  if (reserved.length === 0) return { before: body + (body.length ? "\n\n" : ""), after: "" };
  requireValue(reserved.length === 2 && start >= 0 && end > start && body.indexOf(START, start + START.length) < 0 && body.indexOf(END, end + END.length) < 0, "invalid_markers", "PR body has malformed or duplicate merge-risk markers");
  return { before: body.slice(0, start), after: body.slice(end + END.length) };
}

function firstVerdict(report: string): string {
  const lines = report.split(/\r?\n/);
  return lines.find(line => /^\s*(?:\*\*|#{1,6}\s*)?merge risk\s*:/i.test(line))?.trim()
    ?? lines.find(line => line.trim().length > 0)!.trim();
}

/** Replace only the reserved section. All author text remains byte-for-byte intact. */
export function updateBody(body: string, report: string, metadata: Metadata, sourceURL: string): { body: string; compact: boolean } {
  validateReport(report);
  const { before, after } = bodyParts(body);
  const provenance = `Compared base: \`${metadata.base}\` · head: \`${metadata.head}\`\n\n[Source run](<${sourceURL}>)`;
  const render = (content: string) => `${before}${START}\n${content}\n\n${provenance}\n${END}${after}`;
  const full = render(report);
  if (Buffer.byteLength(full, "utf8") <= BODY_LIMIT) return { body: full, compact: false };
  const compact = render(`${firstVerdict(report)}\n\nReport shortened to fit the PR description limit.`);
  requireValue(Buffer.byteLength(compact, "utf8") <= BODY_LIMIT, "body_too_large", "PR description has no room for the compact merge-risk report; author text is preserved");
  return { body: compact, compact: true };
}

function githubRequest(settings: Settings, fetcher: Fetch) {
  return async (method: "GET" | "PATCH", path: string, body?: { body: string }): Promise<unknown> => {
    const response = await fetcher(`${settings.apiURL}${path}`, {
      method,
      headers: {
        Accept: "application/vnd.github+json",
        Authorization: `Bearer ${settings.token}`,
        "X-GitHub-Api-Version": "2022-11-28",
        "Content-Type": "application/json",
      },
      redirect: "error",
      body: body ? JSON.stringify(body) : undefined,
    });
    requireValue(response.ok, "github_api_error", `GitHub ${method} ${path} failed with status ${response.status}`);
    if (method === "PATCH") return undefined;
    return response.json();
  };
}

type Request = ReturnType<typeof githubRequest>;

async function verifyAssociation(request: Request, path: string, metadata: Metadata, expectedRepository: string): Promise<void> {
  for (let page = 1; ; page++) {
    const values = await request("GET", `${path}/commits/${metadata.head}/pulls?per_page=100&page=${page}`);
    requireValue(Array.isArray(values), "invalid_data", "Commit pull request associations must be an array");
    const match = values.find(value => object(value, "commit pull request association").number === metadata.number);
    if (match) {
      requireValue(staleReason(parsePull(match), metadata, expectedRepository) === undefined, "unbound_pull_request", "Commit association does not match the open PR, repository, base, and exact head");
      return;
    }
    requireValue(values.length === 100, "unbound_pull_request", "Artifact PR number is not associated with the source run head");
  }
}

/** HTTP is injected so tests exercise the same trust checks and request ordering as the CLI. */
export async function publishRisk(input: PublishInput, fetcher: Fetch = fetch): Promise<PublishResult> {
  const settings = { ...input, repository: repository(input.repository), apiURL: baseURL(input.apiURL), serverURL: baseURL(input.serverURL) };
  requireValue(typeof settings.token === "string" && settings.token.trim().length > 0, "invalid_environment", "GH_TOKEN is required");
  const source = parseSource(input.event, settings.repository);
  const metadata = parseMetadata(input.metadata);
  validateReport(input.report);
  requireValue(metadata.head === source.head, "invalid_source", "Artifact head does not match the source workflow run");
  const request = githubRequest(settings, fetcher);
  const path = `/repos/${settings.repository}`;
  if (source.pullNumbers.length === 0) {
    await verifyAssociation(request, path, metadata, settings.repository);
  } else {
    requireValue(source.pullNumbers.includes(metadata.number), "unbound_pull_request", "Artifact PR number is not in source workflow_run.pull_requests");
  }

  // This is the last HTTP read before PATCH. Revalidate this exact PR and use its current body.
  const pullPath = `${path}/pulls/${metadata.number}`;
  const pull = parsePull(await request("GET", pullPath));
  requireValue(pull.number === metadata.number, "invalid_data", "GitHub returned a different PR number");
  const result = { number: metadata.number, run: source.id };
  const reason = staleReason(pull, metadata, settings.repository);
  if (reason) return { ...result, status: "skipped", reason };
  const sourceURL = `${settings.serverURL}/${settings.repository}/actions/runs/${source.id}`;
  const update = updateBody(pull.body, input.report, metadata, sourceURL);
  if (pull.body === update.body) return { ...result, status: "unchanged", compact: update.compact };
  await request("PATCH", pullPath, { body: update.body });
  return { ...result, status: "updated", compact: update.compact };
}

function requiredEnvironment(env: NodeJS.ProcessEnv, key: string): string {
  const value = env[key];
  requireValue(typeof value === "string" && value.trim().length > 0, "invalid_environment", `${key} is required`);
  return value;
}

function parseJSON(text: string, label: string): unknown {
  try {
    return JSON.parse(text);
  } catch {
    throw new PublishError("invalid_json", `${label} must contain valid JSON`);
  }
}

async function readArtifact(directory: string, name: string): Promise<string> {
  const file = await open(join(directory, name), constants.O_RDONLY | constants.O_NOFOLLOW | constants.O_NONBLOCK);
  try {
    requireValue((await file.stat()).isFile(), "invalid_artifact", `${name} must be a regular file`);
    return await file.readFile("utf8");
  } finally {
    await file.close();
  }
}

export async function runFromEnvironment(env: NodeJS.ProcessEnv = process.env, fetcher: Fetch = fetch): Promise<PublishResult> {
  const eventPath = requiredEnvironment(env, "GITHUB_EVENT_PATH");
  const reportDirectory = requiredEnvironment(env, "RISK_REPORT_DIR");
  const settings = {
    repository: requiredEnvironment(env, "GITHUB_REPOSITORY"),
    apiURL: requiredEnvironment(env, "GITHUB_API_URL"),
    serverURL: requiredEnvironment(env, "GITHUB_SERVER_URL"),
    token: requiredEnvironment(env, "GH_TOKEN"),
  };
  const [event, metadata, report] = await Promise.all([
    readFile(eventPath, "utf8"),
    readArtifact(reportDirectory, "metadata.json"),
    readArtifact(reportDirectory, "merge-risk.md"),
  ]);
  return publishRisk({ ...settings, event: parseJSON(event, "GitHub event"), metadata: parseJSON(metadata, "Artifact metadata"), report }, fetcher);
}

if (import.meta.main) {
  try {
    console.log(JSON.stringify({ level: "info", event: "merge_risk_publish", ...await runFromEnvironment() }));
  } catch (error) {
    console.error(JSON.stringify({ level: "error", event: "merge_risk_publish", code: error instanceof PublishError ? error.code : "publish_failed", message: error instanceof Error ? error.message : "Unknown publishing error" }));
    process.exitCode = 1;
  }
}
