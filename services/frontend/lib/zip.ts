import { zipSync } from "fflate";

/** Bundles files into a ZIP. Images are already compressed, so entries are stored (level 0). */
export async function zipFiles(files: { name: string; blob: Blob }[]): Promise<Blob> {
  const used = new Set<string>();
  const entries: Record<string, [Uint8Array, { level: 0 }]> = {};
  for (const f of files) {
    let name = f.name;
    for (let i = 2; used.has(name); i++) name = f.name.replace(/(\.[^.]+)?$/, `-${i}$1`);
    used.add(name);
    entries[name] = [new Uint8Array(await f.blob.arrayBuffer()), { level: 0 }];
  }
  return new Blob([zipSync(entries) as Uint8Array<ArrayBuffer>], { type: "application/zip" });
}
