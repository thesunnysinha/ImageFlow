// Browser-side compressor: everything happens on the visitor's device, nothing is uploaded.
import UPNG from "upng-js";
import {
  MAX_FILE_BYTES,
  fitDimensions,
  isAccepted,
  pngColors,
  resolveType,
  searchQuality,
  type ImageType,
  type OutputChoice,
} from "./compress-core";

export interface Options {
  format: OutputChoice;
  quality: number; // 0.05 .. 0.95
  maxDimension: number | null; // longest side in px, null keeps the size
  targetBytes: number | null; // search for the best quality under this size
}

export interface Result {
  blob: Blob;
  type: ImageType;
  width: number;
  height: number;
  reachedTarget: boolean;
  keptOriginal: boolean; // re-encoding would not have made the file smaller
}

type AnyCanvas = OffscreenCanvas | HTMLCanvasElement;

function makeCanvas(width: number, height: number): AnyCanvas {
  if (typeof OffscreenCanvas !== "undefined") return new OffscreenCanvas(width, height);
  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;
  return canvas;
}

async function canvasToBlob(canvas: AnyCanvas, type: ImageType, quality: number): Promise<Blob> {
  const blob =
    "convertToBlob" in canvas
      ? await canvas.convertToBlob({ type, quality })
      : await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, type, quality));
  // Browsers silently fall back to PNG for formats they cannot encode.
  if (!blob || blob.type !== type) throw new Error(`This browser cannot create ${type.replace("image/", "").toUpperCase()} files.`);
  return blob;
}

export async function compressImage(file: File, options: Options): Promise<Result> {
  if (!isAccepted(file.type)) throw new Error("Only JPEG, PNG and WebP images are supported.");
  if (file.size > MAX_FILE_BYTES) throw new Error("This file is larger than 25 MB.");

  const type = resolveType(file.type, options.format);
  let bitmap: ImageBitmap;
  try {
    bitmap = await createImageBitmap(file); // applies the EXIF orientation
  } catch {
    throw new Error("This image could not be read. It may be damaged.");
  }
  const { width, height } = fitDimensions(bitmap.width, bitmap.height, options.maxDimension);
  const resized = width !== bitmap.width || height !== bitmap.height;
  const canvas = makeCanvas(width, height);
  const ctx = canvas.getContext("2d") as OffscreenCanvasRenderingContext2D | CanvasRenderingContext2D | null;
  if (!ctx) throw new Error("Canvas is not available in this browser.");
  if (type === "image/jpeg") {
    ctx.fillStyle = "#fff"; // JPEG has no transparency: flatten onto white instead of black
    ctx.fillRect(0, 0, width, height);
  }
  ctx.drawImage(bitmap, 0, 0, width, height);
  bitmap.close();

  const encode = async (quality: number): Promise<Blob> => {
    if (type === "image/png") {
      const data = ctx.getImageData(0, 0, width, height);
      const png = UPNG.encode([data.data.buffer as ArrayBuffer], width, height, pngColors(quality));
      return new Blob([png], { type: "image/png" });
    }
    return canvasToBlob(canvas, type, quality);
  };

  let blob: Blob;
  let reachedTarget = true;
  if (options.targetBytes) {
    const found = await searchQuality(encode, options.targetBytes);
    blob = found.blob;
    reachedTarget = found.reachedTarget;
  } else {
    blob = await encode(options.quality);
  }

  // Never hand back something bigger than what the visitor gave us.
  if (blob.size >= file.size && type === file.type && !resized && !options.targetBytes) {
    return { blob: file, type, width, height, reachedTarget: true, keptOriginal: true };
  }
  return { blob, type, width, height, reachedTarget, keptOriginal: false };
}
