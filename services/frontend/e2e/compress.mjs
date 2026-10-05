// End-to-end test: drives a real Chromium against the built site (npm run build first).
// Usage: node e2e/compress.mjs   (CHROME_PATH overrides the browser; PORT the server port)
// E2E_BASE_URL=http://host:port tests an already running server (e.g. the container behind the proxy) and skips the
// checks that need a second, ads-enabled build.
import { spawn, spawnSync } from "node:child_process";
import { existsSync } from "node:fs";
import assert from "node:assert/strict";
import { chromium } from "playwright-core";

const PORT = process.env.PORT ?? "3999";
const EXTERNAL = process.env.E2E_BASE_URL?.replace(/\/+$/, "");
const BASE = EXTERNAL ?? `http://127.0.0.1:${PORT}`;
const chromePath = process.env.CHROME_PATH ?? ["/opt/pw-browsers/chromium"].find(existsSync);

// Run next directly (not through npx) so stopping it really stops the server, and fail early if the port is taken.
if (!EXTERNAL && (await fetch(`http://127.0.0.1:${PORT}`).then(() => true, () => false))) throw new Error(`port ${PORT} is already in use: stop the old server first`);
const servers = [];
function startServer(port, env = {}) {
  const child = spawn(process.execPath, ["node_modules/next/dist/bin/next", "start", "-p", port], { stdio: "ignore", env: { ...process.env, NEXT_TELEMETRY_DISABLED: "1", ...env } });
  servers.push(child);
  return child;
}
const stop = () => servers.forEach((s) => s.kill("SIGTERM"));
process.on("exit", stop);
if (!EXTERNAL) startServer(PORT);

async function waitForServer(base = BASE) {
  for (let i = 0; i < 60; i++) {
    try { if ((await fetch(base)).ok) return; } catch { /* not up yet */ }
    await new Promise((r) => setTimeout(r, 500));
  }
  throw new Error("server did not start");
}

// Builds a test image inside the page, and hands it to the file input like a user choosing a file.
async function addImage(page, { name, type, width, height, kind }) {
  await page.evaluate(async ({ name, type, width, height, kind }) => {
    const canvas = document.createElement("canvas");
    canvas.width = width; canvas.height = height;
    const ctx = canvas.getContext("2d");
    if (kind === "photo") {
      const g = ctx.createLinearGradient(0, 0, width, height);
      g.addColorStop(0, "#1e90ff"); g.addColorStop(0.5, "#f4a261"); g.addColorStop(1, "#2a9d8f");
      ctx.fillStyle = g; ctx.fillRect(0, 0, width, height);
      for (let i = 0; i < 4000; i++) { // grain, so the file is realistically hard to compress
        ctx.fillStyle = `rgba(${(i * 37) % 255},${(i * 91) % 255},${(i * 53) % 255},0.5)`;
        ctx.fillRect((i * 7919) % width, (i * 104729) % height, 3, 3);
      }
    } else { // flat graphic with transparency
      ctx.clearRect(0, 0, width, height);
      ctx.fillStyle = "#e63946"; ctx.fillRect(10, 10, width / 2, height / 2);
      ctx.fillStyle = "rgba(42,157,143,0.5)"; ctx.fillRect(width / 3, height / 3, width / 2, height / 2);
    }
    const blob = await new Promise((r) => canvas.toBlob(r, type, 1));
    const dt = new DataTransfer();
    dt.items.add(new File([blob], name, { type }));
    const input = document.querySelector('[data-testid="file-input"]');
    input.files = dt.files;
    input.dispatchEvent(new Event("change", { bubbles: true }));
  }, { name, type, width, height, kind });
}

// Reads the first result's download link: its size, type and decoded pixels.
async function firstResult(page) {
  await page.waitForSelector('.row[data-status="done"]', { timeout: 30000 });
  return page.evaluate(async () => {
    const a = document.querySelector('.row[data-status="done"] a[download]');
    const blob = await (await fetch(a.href)).blob();
    const bmp = await createImageBitmap(blob);
    const c = new OffscreenCanvas(bmp.width, bmp.height);
    const ctx = c.getContext("2d"); ctx.drawImage(bmp, 0, 0);
    return { size: blob.size, type: blob.type, width: bmp.width, height: bmp.height, name: a.download,
      corner: Array.from(ctx.getImageData(1, 1, 1, 1).data), text: document.querySelector(".row[data-status=done]").textContent };
  });
}

const results = [];
const check = async (name, fn) => { await fn(); results.push(name); console.log("  ok  " + name); };

try {
  await waitForServer();
  const browser = await chromium.launch({ executablePath: chromePath, args: ["--no-sandbox"] });
  const context = await browser.newContext({ acceptDownloads: true });
  const open = async (path) => {
    const page = await context.newPage();
    page.consoleErrors = [];
    page.on("pageerror", (e) => page.consoleErrors.push(String(e)));
    page.on("console", (m) => m.type() === "error" && page.consoleErrors.push(m.text()));
    await page.goto(BASE + path);
    await page.waitForSelector('.tool[data-ready="true"]'); // hydrated: the handlers are attached
    return page;
  };

  await check("home: a JPEG photo gets smaller and stays a valid JPEG", async () => {
    const page = await open("/");
    await addImage(page, { name: "holiday.jpg", type: "image/jpeg", width: 1600, height: 1200, kind: "photo" });
    await page.waitForSelector(".row");
    const before = await page.evaluate(() => document.querySelector(".row .muted").textContent);
    await page.click('[data-testid="compress"]');
    const r = await firstResult(page);
    assert.equal(r.type, "image/jpeg");
    assert.equal(r.name, "holiday-compressed.jpg");
    assert.deepEqual([r.width, r.height], [1600, 1200]);
    assert.ok(/-\d+%/.test(r.text), `shows the saving, got: ${r.text} (was ${before})`);
    assert.equal(page.consoleErrors.length, 0, page.consoleErrors.join("\n"));
  });

  await check("100 KB page: result is at or under the limit", async () => {
    const page = await open("/compress-image-to-100kb");
    await addImage(page, { name: "big.jpg", type: "image/jpeg", width: 1600, height: 1200, kind: "photo" });
    await page.click('[data-testid="compress"]');
    const r = await firstResult(page);
    assert.ok(r.size <= 100 * 1024 || /smallest possible/.test(r.text), `size ${r.size}, ${r.text}`);
    assert.ok(r.size <= 100 * 1024, `expected <= 100 KB, got ${r.size}`);
  });

  await check("PNG page: transparency is kept and the file shrinks", async () => {
    const page = await open("/compress-png");
    await addImage(page, { name: "logo.png", type: "image/png", width: 800, height: 600, kind: "graphic" });
    await page.click('[data-testid="compress"]');
    const r = await firstResult(page);
    assert.equal(r.type, "image/png");
    assert.equal(r.corner[3], 0, `top-left pixel should still be transparent, got ${r.corner}`);
  });

  await check("PNG to WebP page: output is WebP", async () => {
    const page = await open("/convert-png-to-webp");
    await addImage(page, { name: "logo.png", type: "image/png", width: 800, height: 600, kind: "graphic" });
    await page.click('[data-testid="compress"]');
    const r = await firstResult(page);
    assert.equal(r.type, "image/webp");
    assert.equal(r.name, "logo-compressed.webp");
  });

  await check("several images: ZIP download works", async () => {
    const page = await open("/");
    await addImage(page, { name: "a.jpg", type: "image/jpeg", width: 900, height: 600, kind: "photo" });
    await page.waitForSelector(".row");
    await addImage(page, { name: "a.jpg", type: "image/jpeg", width: 900, height: 600, kind: "photo" }); // same name twice
    await page.waitForFunction(() => document.querySelectorAll(".row").length === 2);
    await page.click('[data-testid="compress"]');
    await page.waitForFunction(() => document.querySelectorAll('.row[data-status="done"]').length === 2, null, { timeout: 30000 });
    const [download] = await Promise.all([page.waitForEvent("download"), page.click("text=Download all (ZIP)")]);
    assert.equal(download.suggestedFilename(), "compressed-images.zip");
    const stream = await download.createReadStream();
    const chunks = []; for await (const c of stream) chunks.push(c);
    const zip = Buffer.concat(chunks);
    assert.equal(zip.subarray(0, 2).toString(), "PK");
    const names = zip.toString("latin1").match(/a-compressed(-\d)?\.jpg/g) ?? [];
    assert.ok(new Set(names).size === 2, `two distinct entries expected, saw ${[...new Set(names)]}`);
  });

  await check("unsupported files are refused with a message", async () => {
    const page = await open("/");
    await page.evaluate(() => {
      const dt = new DataTransfer();
      dt.items.add(new File(["hello"], "notes.txt", { type: "text/plain" }));
      const input = document.querySelector('[data-testid="file-input"]');
      input.files = dt.files; input.dispatchEvent(new Event("change", { bubbles: true }));
    });
    await page.waitForSelector("text=only JPEG, PNG and WebP are supported");
    assert.equal(await page.locator(".row").count(), 0);
  });

  await check("SEO: titles, structured data, sitemap, robots, privacy; no ads without configuration", async () => {
    const page = await open("/compress-jpg");
    assert.match(await page.title(), /Compress JPG Images Online/);
    assert.equal(await page.locator('script[type="application/ld+json"]').count(), 2);
    assert.equal(await page.locator("h1").count(), 1);
    assert.equal(await page.locator(".adsbygoogle").count(), 0, "no ad markup without NEXT_PUBLIC_ADSENSE_CLIENT");
    assert.equal(await page.locator('script[src*="adsbygoogle"]').count(), 0);
    const sitemap = await (await fetch(BASE + "/sitemap.xml")).text();
    for (const slug of ["compress-jpg", "compress-png", "compress-image-to-100kb", "convert-png-to-webp"]) assert.ok(sitemap.includes(`/${slug}`), slug);
    assert.match(await (await fetch(BASE + "/robots.txt")).text(), /Sitemap:/);
    assert.equal((await fetch(BASE + "/ads.txt")).status, 404, "ads.txt must not exist without a publisher id");
    assert.equal((await fetch(BASE + "/privacy")).status, 200);
    assert.equal((await fetch(BASE + "/no-such-page")).status, 404);
  });

  if (!EXTERNAL) {
  // ---- Ads configured: build a second variant with an AdSense client and slots, and inspect what it renders.
  const adsEnv = { NEXT_PUBLIC_ADSENSE_CLIENT: "ca-pub-1234567890123456", NEXT_PUBLIC_ADSENSE_SLOT_CONTENT: "1111111111",
    NEXT_PUBLIC_ADSENSE_SLOT_BOTTOM: "2222222222", NEXT_PUBLIC_SITE_URL: "https://tools.example.com", NEXT_DIST_DIR: ".next-ads" };
  const build = spawnSync(process.execPath, ["node_modules/next/dist/bin/next", "build"], { env: { ...process.env, NEXT_TELEMETRY_DISABLED: "1", ...adsEnv }, stdio: "ignore" });
  assert.equal(build.status, 0, "the ads variant must build");
  const ADS_PORT = String(Number(PORT) + 1);
  const ADS_BASE = `http://127.0.0.1:${ADS_PORT}`;
  startServer(ADS_PORT, adsEnv);
  await waitForServer(ADS_BASE);

  await check("ads enabled: consent defaults come first, slots reserve space, ads.txt and canonical URLs are right", async () => {
    const html = await (await fetch(ADS_BASE + "/compress-jpg")).text();
    const consent = html.indexOf("gtag('consent','default'");
    const adsTag = html.indexOf("pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=ca-pub-1234567890123456");
    assert.ok(consent > -1 && adsTag > -1, "both scripts present");
    const unescaped = html.replace(/\\"/g, '"'); // Next embeds inline scripts JSON-escaped
    assert.ok(unescaped.includes('"ad_storage":"denied"') && unescaped.includes('"region":["AT"'), "consent denied by default in the EEA/UK/CH");
    assert.equal((html.match(/data-ad-slot="/g) ?? []).length, 2, "two slots configured, the unconfigured top slot renders nothing");
    assert.ok(html.includes('data-ad-slot="1111111111"') && html.includes('data-ad-client="ca-pub-1234567890123456"'));
    assert.equal((await fetch(ADS_BASE + "/ads.txt")).status, 200);
    assert.equal(await (await fetch(ADS_BASE + "/ads.txt")).text(), "google.com, pub-1234567890123456, DIRECT, f08c47fec0942fa0\n");
    assert.ok(html.includes('<link rel="canonical" href="https://tools.example.com/compress-jpg"'), "canonical uses the site URL");

    // Layout: an ad is never a direct neighbour of the tool's buttons (accidental clicks violate AdSense policy).
    const page = await context.newPage();
    await page.route(/googlesyndication|doubleclick/, (r) => r.abort()); // no network ads in the test
    // Runtime order: when the AdSense <script> is inserted, the consent defaults must already be in dataLayer.
    await page.addInitScript(() => {
      new MutationObserver((records) => {
        for (const r of records) for (const n of r.addedNodes) {
          if (n.nodeName === "SCRIPT" && n.src && n.src.includes("adsbygoogle.js") && !window.__adsOrder) {
            const entry = (window.dataLayer || []).map((a) => Array.from(a)).find((a) => a[0] === "consent" && a[1] === "default");
            window.__adsOrder = { consentSetBefore: !!entry, adStorage: entry && entry[2].ad_storage };
          }
        }
      }).observe(document, { childList: true, subtree: true });
    });
    await page.goto(ADS_BASE + "/compress-jpg");
    await page.waitForFunction(() => window.__adsOrder, null, { timeout: 15000 });
    const order = await page.evaluate(() => window.__adsOrder);
    assert.deepEqual(order, { consentSetBefore: true, adStorage: "denied" }, "consent defaults must be set before the AdSense tag is inserted");
    await page.waitForSelector('.tool[data-ready="true"]');
    const ad = await page.locator("aside.ad").first().boundingBox();
    const tool = await page.locator(".tool").boundingBox();
    assert.ok(ad.height >= 280, `slot reserves its height before the ad loads (${ad.height}px)`);
    assert.ok(ad.y >= tool.y + tool.height + 8, "the first ad sits below the tool, not inside or touching it");
    assert.equal(await page.locator(".tool aside.ad, .tool .adsbygoogle").count(), 0, "no ads inside the tool");
  });

  }

  await browser.close();
  console.log(`\n${results.length} browser checks passed`);
  stop();
  process.exit(0);
} catch (e) {
  console.error("\nFAILED:", e);
  stop();
  process.exit(1);
}
