import assert from "node:assert/strict";
import { test } from "node:test";
import { ApiClient, ApiError } from "./client";

type Call = { url: string; init: RequestInit };

function fake(status: number, body: unknown, calls: Call[] = []): typeof fetch {
  return (async (url: string, init: RequestInit) => {
    calls.push({ url, init });
    return new Response(typeof body === "string" ? body : JSON.stringify(body), { status });
  }) as unknown as typeof fetch;
}

const ok = (data: unknown, meta: Record<string, unknown> = {}) => ({ success: true, code: "OK", message: "", data, meta, trace_id: "t" });

test("sends the API key and builds URLs under /api/v1", async () => {
  const calls: Call[] = [];
  const client = new ApiClient({ baseUrl: "https://api.example.com/", apiKey: "k1", fetch: fake(200, ok([]), calls) });
  await client.listJobs({ limit: 5, before: "2026-01-01T00:00:00Z" });
  assert.equal(calls[0]?.url, "https://api.example.com/api/v1/jobs?limit=5&before=2026-01-01T00%3A00%3A00Z");
  assert.equal((calls[0]?.init.headers as Record<string, string>)["Authorization"], "Bearer k1");
});

test("listJobs exposes the pagination cursor only when the server sends one", async () => {
  const withCursor = new ApiClient({ baseUrl: "https://a", apiKey: "k", fetch: fake(200, ok([], { next_before: "c1" })) });
  assert.equal((await withCursor.listJobs()).nextBefore, "c1");
  const last = new ApiClient({ baseUrl: "https://a", apiKey: "k", fetch: fake(200, ok([])) });
  assert.equal((await last.listJobs()).nextBefore, null);
});

test("createJob posts snake_case JSON and omits an empty webhook", async () => {
  const calls: Call[] = [];
  const client = new ApiClient({ baseUrl: "https://a", apiKey: "k", fetch: fake(202, ok({ id: "j" }), calls) });
  await client.createJob({ sourceUrls: ["https://x/1.jpg"] });
  assert.equal(calls[0]?.init.method, "POST");
  assert.deepEqual(JSON.parse(String(calls[0]?.init.body)), { source_urls: ["https://x/1.jpg"] });
  await client.createJob({ sourceUrls: ["https://x/1.jpg"], webhookUrl: "https://h" });
  assert.equal(JSON.parse(String(calls[1]?.init.body)).webhook_url, "https://h");
});

test("API errors keep the code, status and field details", async () => {
  const body = { success: false, code: "VALIDATION_ERROR", message: "The request is invalid.", data: null, trace_id: "t",
    meta: { details: [{ field: "source_urls[0]", message: "The URL is not allowed." }] } };
  const client = new ApiClient({ baseUrl: "https://a", apiKey: "k", fetch: fake(422, body) });
  await assert.rejects(client.createJob({ sourceUrls: ["x"] }), (e: unknown) => {
    assert.ok(e instanceof ApiError);
    assert.equal(e.status, 422);
    assert.equal(e.code, "VALIDATION_ERROR");
    assert.equal(e.details[0]?.field, "source_urls[0]");
    return true;
  });
});

test("a non-JSON gateway error becomes BAD_RESPONSE, a dead network NETWORK_ERROR", async () => {
  const gateway = new ApiClient({ baseUrl: "https://a", apiKey: "k", fetch: fake(502, "<html>Bad gateway</html>") });
  await assert.rejects(gateway.getJob("j"), (e: unknown) => e instanceof ApiError && e.code === "BAD_RESPONSE" && e.status === 502);
  const offline = new ApiClient({ baseUrl: "https://a", apiKey: "k", fetch: (async () => { throw new TypeError("Network request failed"); }) as unknown as typeof fetch });
  await assert.rejects(offline.getJob("j"), (e: unknown) => e instanceof ApiError && e.code === "NETWORK_ERROR" && e.status === 0);
});

test("outputUrl and authHeaders let an <Image> fetch a protected output", () => {
  const client = new ApiClient({ baseUrl: "https://a.example", apiKey: "secret", fetch: fake(200, ok(null)) });
  assert.equal(client.outputUrl("abc", 3), "https://a.example/api/v1/jobs/abc/items/3/output");
  assert.deepEqual(client.authHeaders, { Authorization: "Bearer secret" });
});
