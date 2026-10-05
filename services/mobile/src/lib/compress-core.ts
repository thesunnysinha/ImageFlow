// Pure parts of on-device compression (no native modules), so they can be unit tested in Node.

/** Scale (width, height) down so the longest side is at most max. Never scales up. */
export function fitDimensions(width: number, height: number, max: number | null): { width: number; height: number } {
  if (!max || (width <= max && height <= max)) return { width, height };
  const scale = max / Math.max(width, height);
  return { width: Math.max(1, Math.round(width * scale)), height: Math.max(1, Math.round(height * scale)) };
}

export interface SearchResult<T extends { size: number }> {
  encoded: T;
  quality: number;
  reachedTarget: boolean;
}

/**
 * Finds the highest quality whose output is at most targetBytes, by bisection over [minQ, maxQ].
 * `dispose` is called for every encode that is not the one returned (so temp files can be removed).
 */
export async function searchQuality<T extends { size: number }>(
  encode: (quality: number) => Promise<T>,
  targetBytes: number,
  dispose: (encoded: T) => void = () => {},
  { minQ = 0.05, maxQ = 0.95, steps = 6 }: { minQ?: number; maxQ?: number; steps?: number } = {},
): Promise<SearchResult<T>> {
  const top = await encode(maxQ);
  if (top.size <= targetBytes) return { encoded: top, quality: maxQ, reachedTarget: true };
  dispose(top);
  const bottom = await encode(minQ);
  if (bottom.size > targetBytes) return { encoded: bottom, quality: minQ, reachedTarget: false };
  let best = { encoded: bottom, quality: minQ };
  let lo = minQ;
  let hi = maxQ;
  for (let i = 0; i < steps; i++) {
    const mid = (lo + hi) / 2;
    const candidate = await encode(mid);
    if (candidate.size <= targetBytes) {
      dispose(best.encoded);
      best = { encoded: candidate, quality: mid };
      lo = mid;
    } else {
      dispose(candidate);
      hi = mid;
    }
  }
  return { ...best, reachedTarget: true };
}

export const QUALITY_PRESETS = [
  { id: "high", label: "High", quality: 0.8 },
  { id: "balanced", label: "Balanced", quality: 0.65 },
  { id: "small", label: "Small", quality: 0.45 },
] as const;

export const SIZE_PRESETS: { label: string; maxDimension: number | null }[] = [
  { label: "Original", maxDimension: null },
  { label: "2560", maxDimension: 2560 },
  { label: "1920", maxDimension: 1920 },
  { label: "1280", maxDimension: 1280 },
];

export const MAX_PHOTOS = 20;
