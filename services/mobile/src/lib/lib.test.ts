import assert from "node:assert/strict";
import { test } from "node:test";
import { formatBytes, isTerminal, progress, savings } from "./status";
import { normalizeBaseUrl, parseUrlList } from "./urls";

test("parseUrlList splits on whitespace and commas, dedupes, and reports bad entries", () => {
  const { urls, invalid } = parseUrlList(" https://a.com/1.jpg,https://a.com/2.png\nhttps://a.com/1.jpg ftp://x/y not-a-url ;http://b.org/3.jpg ");
  assert.deepEqual(urls, ["https://a.com/1.jpg", "https://a.com/2.png", "http://b.org/3.jpg"]);
  assert.deepEqual(invalid, ["ftp://x/y", "not-a-url"]);
  assert.deepEqual(parseUrlList("   \n "), { urls: [], invalid: [] });
});

test("normalizeBaseUrl adds https, strips trailing slashes and rejects junk", () => {
  assert.equal(normalizeBaseUrl("api.example.com"), "https://api.example.com");
  assert.equal(normalizeBaseUrl("  http://10.0.2.2:8080/ "), "http://10.0.2.2:8080");
  assert.equal(normalizeBaseUrl("https://x.com/prefix//"), "https://x.com/prefix");
  assert.equal(normalizeBaseUrl(""), null);
  assert.equal(normalizeBaseUrl("ftp://x.com"), null);
});

test("status helpers", () => {
  assert.ok(isTerminal("completed") && isTerminal("partial") && isTerminal("failed"));
  assert.ok(!isTerminal("queued") && !isTerminal("processing"));
  assert.equal(progress({ total: 4, completed: 1, failed: 1 }), 0.5);
  assert.equal(progress({ total: 0, completed: 0, failed: 0 }), 0);
  assert.equal(formatBytes(512), "512 B");
  assert.equal(formatBytes(2048), "2.0 KB");
  assert.equal(formatBytes(3 * 1024 * 1024), "3.0 MB");
  assert.equal(savings(1000, 400), "-60%");
  assert.equal(savings(1000, 1000), "");
  assert.equal(savings(undefined, 5), "");
});
