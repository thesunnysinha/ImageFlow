// On-device compression with the platform's own image codecs. Nothing is uploaded.
import { File } from "expo-file-system";
import { ImageManipulator, SaveFormat } from "expo-image-manipulator";
import { fitDimensions, searchQuality } from "./compress-core";

export interface DeviceOptions {
  quality: number; // 0.05 .. 0.95
  maxDimension: number | null; // longest side in px
  targetBytes: number | null; // search for the best quality under this size
}

export interface DeviceSource {
  uri: string;
  width: number;
  height: number;
  bytes?: number; // original size when the picker knows it
}

export interface DeviceResult {
  uri: string;
  width: number;
  height: number;
  bytes: number;
  originalBytes: number;
  reachedTarget: boolean;
  keptOriginal: boolean; // re-encoding would not have made the file smaller
}

interface Encoded {
  uri: string;
  width: number;
  height: number;
  size: number;
}

function sizeOf(uri: string): number {
  return new File(uri).size;
}

function remove(uri: string): void {
  try {
    new File(uri).delete();
  } catch {
    // A leftover cache file is harmless; the OS clears the cache directory.
  }
}

/** Re-encodes one photo as JPEG. Large photos are resized first, which saves far more than quality alone. */
export async function compressOnDevice(source: DeviceSource, options: DeviceOptions): Promise<DeviceResult> {
  const originalBytes = source.bytes ?? sizeOf(source.uri);
  const { width, height } = fitDimensions(source.width, source.height, options.maxDimension);
  const resized = width !== source.width || height !== source.height;

  const context = ImageManipulator.manipulate(source.uri);
  if (resized) context.resize({ width }); // height follows, keeping the aspect ratio
  const rendered = await context.renderAsync(); // decode and resize once, then encode at several qualities

  const encode = async (quality: number): Promise<Encoded> => {
    const saved = await rendered.saveAsync({ compress: quality, format: SaveFormat.JPEG });
    return { uri: saved.uri, width: saved.width, height: saved.height, size: sizeOf(saved.uri) };
  };

  let chosen: Encoded;
  let reachedTarget = true;
  if (options.targetBytes) {
    const found = await searchQuality(encode, options.targetBytes, (e) => remove(e.uri));
    chosen = found.encoded;
    reachedTarget = found.reachedTarget;
  } else {
    chosen = await encode(options.quality);
  }

  // Never hand back something bigger than what the user gave us.
  if (chosen.size >= originalBytes && !resized && !options.targetBytes) {
    remove(chosen.uri);
    return { uri: source.uri, width: source.width, height: source.height, bytes: originalBytes, originalBytes, reachedTarget: true, keptOriginal: true };
  }
  return { uri: chosen.uri, width: chosen.width, height: chosen.height, bytes: chosen.size, originalBytes, reachedTarget, keptOriginal: false };
}
