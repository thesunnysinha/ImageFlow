import assert from "node:assert/strict";
import { test } from "node:test";
import { fitDimensions, isAccepted, outputName, pngColors, resolveType, searchQuality } from "./compress-core";
import { formatBytes, formatChange } from "./format";

test("fitDimensions keeps the aspect ratio and never scales up", () => {
  assert.deepEqual(fitDimensions(4000, 3000, 2000), { width: 2000, height: 1500 });
  assert.deepEqual(fitDimensions(3000, 4000, 2000), { width: 1500, height: 2000 });
  assert.deepEqual(fitDimensions(800, 600, 2000), { width: 800, height: 600 });
  assert.deepEqual(fitDimensions(800, 600, null), { width: 800, height: 600 });
  assert.deepEqual(fitDimensions(10000, 1, 100), { width: 100, height: 1 });
});

test("outputName sanitises and swaps the extension", () => {
  assert.equal(outputName("photo.final.PNG", "image/webp"), "photo.final-compressed.webp");
  assert.equal(outputName("no-extension", "image/jpeg"), "no-extension-compressed.jpg");
  assert.equal(outputName('we/ird:na"me.jpg', "image/jpeg"), "we_ird_na_me-compressed.jpg");
  assert.equal(outputName(".hidden", "image/png"), ".hidden-compressed.png");
});

test("format choice and accepted types", () => {
  assert.equal(resolveType("image/png", "original"), "image/png");
  assert.equal(resolveType("image/png", "image/webp"), "image/webp");
  assert.ok(isAccepted("image/jpeg") && isAccepted("image/webp") && !isAccepted("image/gif") && !isAccepted("image/heic"));
  assert.equal(pngColors(1), 0);
  assert.ok(pngColors(0.9) > pngColors(0.5));
});

// A fake encoder whose size grows with quality, like a real JPEG/WebP encoder.
const fakeEncode = (calls: number[] = []) => async (q: number) => {
  calls.push(q);
  return { size: Math.round(10_000 + q * 190_000) };
};

test("searchQuality returns the best quality that fits the target", async () => {
  const calls: number[] = [];
  const r = await searchQuality(fakeEncode(calls), 100_000);
  assert.ok(r.reachedTarget);
  assert.ok(r.blob.size <= 100_000);
  assert.ok(r.blob.size > 90_000, `should use most of the budget, got ${r.blob.size}`);
  assert.ok(calls.length <= 10, "bounded number of encodes");
});

test("searchQuality short-circuits when the best quality already fits, and reports when nothing can", async () => {
  const fits = await searchQuality(fakeEncode(), 1_000_000);
  assert.deepEqual([fits.reachedTarget, fits.quality], [true, 0.95]);
  const impossible = await searchQuality(fakeEncode(), 5_000);
  assert.equal(impossible.reachedTarget, false);
  assert.equal(impossible.quality, 0.05);
});

test("formatters", () => {
  assert.equal(formatBytes(512), "512 B");
  assert.equal(formatBytes(2048), "2.0 KB");
  assert.equal(formatBytes(150 * 1024), "150 KB");
  assert.equal(formatBytes(3 * 1024 * 1024), "3.0 MB");
  assert.equal(formatChange(1000, 400), "-60%");
  assert.equal(formatChange(1000, 1100), "+10%");
  assert.equal(formatChange(1000, 1000), "");
});
