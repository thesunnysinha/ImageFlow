// Pure parts of the compressor (no DOM), so they can be unit tested in Node.

export type ImageType = "image/jpeg" | "image/png" | "image/webp";
export type OutputChoice = "original" | ImageType;

export const ACCEPTED_TYPES: readonly ImageType[] = ["image/jpeg", "image/png", "image/webp"];
export const MAX_FILE_BYTES = 25 * 1024 * 1024;
export const MAX_FILES = 20;

export function isAccepted(type: string): type is ImageType {
  return (ACCEPTED_TYPES as readonly string[]).includes(type);
}

/** Output type for an input type and the user's format choice. */
export function resolveType(input: ImageType, choice: OutputChoice): ImageType {
  return choice === "original" ? input : choice;
}

const EXTENSIONS: Record<ImageType, string> = { "image/jpeg": "jpg", "image/png": "png", "image/webp": "webp" };

/** "photo.final.PNG" + image/webp -> "photo.final-compressed.webp" */
export function outputName(inputName: string, type: ImageType): string {
  const dot = inputName.lastIndexOf(".");
  const base = (dot > 0 ? inputName.slice(0, dot) : inputName).replace(/[\\/:*?"<>|\x00-\x1f]/g, "_") || "image";
  return `${base}-compressed.${EXTENSIONS[type]}`;
}

/** Scale (width, height) down so the longest side is at most max. Never scales up. */
export function fitDimensions(width: number, height: number, max: number | null): { width: number; height: number } {
  if (!max || (width <= max && height <= max)) return { width, height };
  const scale = max / Math.max(width, height);
  return { width: Math.max(1, Math.round(width * scale)), height: Math.max(1, Math.round(height * scale)) };
}

/** Number of PNG palette colours for a 0..1 quality (UPNG: 0 means lossless). */
export function pngColors(quality: number): number {
  if (quality >= 0.95) return 0;
  if (quality >= 0.8) return 256;
  if (quality >= 0.6) return 128;
  if (quality >= 0.4) return 64;
  return 32;
}

export interface SearchResult<T extends { size: number }> {
  blob: T;
  quality: number;
  reachedTarget: boolean;
}

/**
 * Finds the highest quality whose output is at most targetBytes, by bisection over [minQ, maxQ].
 * If even minQ is too big, returns the minQ result with reachedTarget=false.
 */
export async function searchQuality<T extends { size: number }>(
  encode: (quality: number) => Promise<T>,
  targetBytes: number,
  { minQ = 0.05, maxQ = 0.95, steps = 7 }: { minQ?: number; maxQ?: number; steps?: number } = {},
): Promise<SearchResult<T>> {
  const top = await encode(maxQ);
  if (top.size <= targetBytes) return { blob: top, quality: maxQ, reachedTarget: true };
  const bottom = await encode(minQ);
  if (bottom.size > targetBytes) return { blob: bottom, quality: minQ, reachedTarget: false };
  let best = { blob: bottom, quality: minQ };
  let lo = minQ;
  let hi = maxQ;
  for (let i = 0; i < steps; i++) {
    const mid = (lo + hi) / 2;
    const candidate = await encode(mid);
    if (candidate.size <= targetBytes) {
      best = { blob: candidate, quality: mid };
      lo = mid;
    } else {
      hi = mid;
    }
  }
  return { ...best, reachedTarget: true };
}
