// Lighthouse budget for the built site: protects what search ranking and ad policy care about.
// Usage: npm run build && node e2e/lighthouse.mjs      (CHROME_PATH selects the browser; PORT the server port)
// Accessibility, best practices, SEO and layout shift are deterministic, so they are held strictly. Performance varies
// on shared machines, so it only has a floor.
import { spawn, spawnSync } from "node:child_process";
import { existsSync, readFileSync, mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import assert from "node:assert/strict";

const PORT = process.env.PORT ?? "3996";
const BASE = `http://127.0.0.1:${PORT}`;
const PAGES = ["/", "/compress-jpg", "/compress-image-to-100kb", "/privacy"];
const MIN = { accessibility: 95, "best-practices": 95, seo: 95, performance: 80 };
const MAX_CLS = 0.1;

// CHROME_PATH wins; then the preinstalled sandbox browser; then the Chromium that `playwright-core install` fetched.
const { chromium } = await import("playwright-core");
const chrome = process.env.CHROME_PATH ?? ["/opt/pw-browsers/chromium", chromium.executablePath()].find((p) => p && existsSync(p));
assert.ok(chrome, "set CHROME_PATH to a Chrome or Chromium executable");

const server = spawn(process.execPath, ["node_modules/next/dist/bin/next", "start", "-p", PORT], { stdio: "ignore", env: { ...process.env, NEXT_TELEMETRY_DISABLED: "1" } });
const stop = () => server.kill("SIGTERM");
process.on("exit", stop);

try {
  for (let i = 0; ; i++) {
    try { if ((await fetch(BASE)).ok) break; } catch { /* not up yet */ }
    if (i > 60) throw new Error("server did not start");
    await new Promise((r) => setTimeout(r, 500));
  }
  const dir = mkdtempSync(join(tmpdir(), "lighthouse-"));
  let failures = 0;
  for (const path of PAGES) {
    const out = join(dir, "report.json");
    const run = spawnSync("npx", ["--yes", "lighthouse@12", BASE + path, "--chrome-flags=--headless=new --no-sandbox --disable-gpu",
      "--only-categories=performance,accessibility,best-practices,seo", "--output=json", `--output-path=${out}`, "--quiet"],
      { env: { ...process.env, CHROME_PATH: chrome }, encoding: "utf8" });
    if (run.status !== 0) throw new Error(`lighthouse failed on ${path}: ${run.stderr.split("\n").slice(0, 3).join(" ")}`);
    const report = JSON.parse(readFileSync(out, "utf8"));
    const scores = Object.fromEntries(Object.entries(report.categories).map(([k, v]) => [k, Math.round(v.score * 100)]));
    const cls = report.audits["cumulative-layout-shift"].numericValue;
    const problems = [
      ...Object.entries(MIN).filter(([k, min]) => scores[k] < min).map(([k, min]) => `${k} ${scores[k]} < ${min}`),
      ...(cls > MAX_CLS ? [`layout shift ${cls.toFixed(3)} > ${MAX_CLS}`] : []),
    ];
    console.log(`${problems.length ? "FAIL" : "  ok"}  ${path.padEnd(28)} ${JSON.stringify(scores)} CLS ${cls.toFixed(3)}${problems.length ? "  <- " + problems.join("; ") : ""}`);
    failures += problems.length;
  }
  stop();
  process.exit(failures ? 1 : 0);
} catch (e) {
  console.error(e);
  stop();
  process.exit(1);
}
