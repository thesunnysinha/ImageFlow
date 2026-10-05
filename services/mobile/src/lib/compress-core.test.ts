import assert from "node:assert/strict";
import { test } from "node:test";
import { fitDimensions, searchQuality } from "./compress-core";

test("fitDimensions keeps the aspect ratio and never scales up", () => {
  assert.deepEqual(fitDimensions(4000, 3000, 2000), { width: 2000, height: 1500 });
  assert.deepEqual(fitDimensions(3000, 4000, 2000), { width: 1500, height: 2000 });
  assert.deepEqual(fitDimensions(800, 600, 2000), { width: 800, height: 600 });
  assert.deepEqual(fitDimensions(800, 600, null), { width: 800, height: 600 });
});

// An encoder whose size grows with quality, like a real JPEG encoder; each result carries an id so disposal can be tracked.
function fakeEncoder() {
  const made: { id: number; size: number }[] = [];
  const encode = async (q: number) => {
    const e = { id: made.length, size: Math.round(10_000 + q * 190_000) };
    made.push(e);
    return e;
  };
  return { made, encode };
}

test("searchQuality returns the best fit and disposes every other encode", async () => {
  const { made, encode } = fakeEncoder();
  const disposed = new Set<number>();
  const r = await searchQuality(encode, 100_000, (e) => disposed.add(e.id));
  assert.ok(r.reachedTarget && r.encoded.size <= 100_000 && r.encoded.size > 90_000, `size ${r.encoded.size}`);
  assert.ok(made.length <= 10, "bounded number of encodes");
  assert.ok(!disposed.has(r.encoded.id), "the returned encode is kept");
  assert.equal(disposed.size, made.length - 1, "every other encode is disposed (no leaked temp files)");
});

test("searchQuality short-circuits when the best quality fits, and reports when nothing can", async () => {
  const fits = await searchQuality(fakeEncoder().encode, 1_000_000);
  assert.deepEqual([fits.reachedTarget, fits.quality], [true, 0.95]);
  const { made, encode } = fakeEncoder();
  const disposed: number[] = [];
  const impossible = await searchQuality(encode, 5_000, (e) => disposed.push(e.id));
  assert.equal(impossible.reachedTarget, false);
  assert.equal(impossible.quality, 0.05);
  assert.deepEqual(disposed, [0], `only the failed top attempt is disposed; made ${made.length}`);
});
