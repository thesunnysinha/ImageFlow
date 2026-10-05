"use client";

import { useCallback, useEffect, useId, useRef, useState } from "react";
import { compressImage, type Options, type Result } from "../lib/compress";
import { ACCEPTED_TYPES, MAX_FILES, MAX_FILE_BYTES, isAccepted, outputName, type OutputChoice } from "../lib/compress-core";
import { formatBytes, formatChange } from "../lib/format";
import { zipFiles } from "../lib/zip";

export interface Preset {
  format?: OutputChoice;
  quality?: number;
  maxDimension?: number | null;
  targetKB?: number | null;
}

interface Item {
  id: number;
  file: File;
  status: "queued" | "working" | "done" | "error";
  result?: Result;
  url?: string;
  error?: string;
}

let nextId = 1;

export default function Tool({ preset = {} }: { preset?: Preset }) {
  const uid = useId();
  const input = useRef<HTMLInputElement>(null);
  const [items, setItems] = useState<Item[]>([]);
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState("");
  const [drag, setDrag] = useState(false);
  const [ready, setReady] = useState(false); // true once hydrated and the handlers are attached
  useEffect(() => setReady(true), []);
  const [format, setFormat] = useState<OutputChoice>(preset.format ?? "original");
  const [quality, setQuality] = useState(preset.quality ?? 0.75);
  const [maxDimension, setMaxDimension] = useState<number | null>(preset.maxDimension ?? null);
  const [targetKB, setTargetKB] = useState<number | null>(preset.targetKB ?? null);

  // Release the object URLs when the component goes away.
  const urls = useRef<string[]>([]);
  useEffect(() => () => urls.current.forEach(URL.revokeObjectURL), []);

  const addFiles = useCallback((list: FileList | File[]) => {
    const files = Array.from(list);
    const problems: string[] = [];
    const accepted: Item[] = [];
    for (const file of files) {
      if (!isAccepted(file.type)) problems.push(`${file.name}: only JPEG, PNG and WebP are supported`);
      else if (file.size > MAX_FILE_BYTES) problems.push(`${file.name}: larger than 25 MB`);
      else accepted.push({ id: nextId++, file, status: "queued" });
    }
    setItems((current) => {
      const room = Math.max(0, MAX_FILES - current.length);
      if (accepted.length > room) problems.push(`Up to ${MAX_FILES} images at a time`);
      return [...current, ...accepted.slice(0, room)];
    });
    setNotice(problems.join(". "));
  }, []);

  const options = (): Options => ({ format, quality, maxDimension, targetBytes: targetKB ? targetKB * 1024 : null });

  async function run() {
    setBusy(true);
    setNotice("");
    const opts = options();
    // Everything is (re)processed with the current settings, one image at a time so the page stays responsive.
    const queue = items.map((i) => i.id);
    for (const id of queue) {
      const item = items.find((i) => i.id === id);
      if (!item) continue;
      setItems((cur) => cur.map((i) => (i.id === id ? { ...i, status: "working", error: undefined } : i)));
      await new Promise((r) => setTimeout(r, 0)); // let the UI paint
      try {
        const result = await compressImage(item.file, opts);
        const url = URL.createObjectURL(result.blob);
        urls.current.push(url);
        setItems((cur) => cur.map((i) => (i.id === id ? { ...i, status: "done", result, url } : i)));
      } catch (e) {
        setItems((cur) => cur.map((i) => (i.id === id ? { ...i, status: "error", error: e instanceof Error ? e.message : "Failed" } : i)));
      }
    }
    setBusy(false);
  }

  async function downloadAll() {
    const done = items.filter((i) => i.result);
    const blob = await zipFiles(done.map((i) => ({ name: outputName(i.file.name, i.result!.type), blob: i.result!.blob })));
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "compressed-images.zip";
    a.click();
    setTimeout(() => URL.revokeObjectURL(url), 10_000);
  }

  const done = items.filter((i) => i.status === "done");
  const before = done.reduce((n, i) => n + i.file.size, 0);
  const after = done.reduce((n, i) => n + i.result!.blob.size, 0);

  return (
    <section className="tool" aria-labelledby={`${uid}-title`} data-ready={ready}>
      <h2 id={`${uid}-title`} className="sr-only">Image compressor</h2>

      <div
        className={`drop${drag ? " drop-active" : ""}`}
        onDragOver={(e) => { e.preventDefault(); setDrag(true); }}
        onDragLeave={() => setDrag(false)}
        onDrop={(e) => { e.preventDefault(); setDrag(false); addFiles(e.dataTransfer.files); }}
      >
        <p className="drop-title">Drop images here</p>
        <p className="muted">JPEG, PNG or WebP · up to {MAX_FILES} images · 25 MB each</p>
        <button type="button" className="btn" onClick={() => input.current?.click()}>Choose images</button>
        <input ref={input} type="file" multiple hidden accept={ACCEPTED_TYPES.join(",")} data-testid="file-input"
          onChange={(e) => { if (e.target.files) addFiles(e.target.files); e.target.value = ""; }} />
      </div>

      <div className="controls">
        <label>Format
          <select value={format} onChange={(e) => setFormat(e.target.value as OutputChoice)}>
            <option value="original">Keep original</option>
            <option value="image/jpeg">JPEG</option>
            <option value="image/webp">WebP</option>
            <option value="image/png">PNG</option>
          </select>
        </label>
        <label>Quality: {Math.round(quality * 100)}%
          <input type="range" min={0.1} max={0.95} step={0.05} value={quality} disabled={!!targetKB}
            onChange={(e) => setQuality(Number(e.target.value))} />
        </label>
        <label>Max size (KB)
          <input type="number" min={10} max={20000} placeholder="Off" value={targetKB ?? ""}
            onChange={(e) => setTargetKB(e.target.value ? Math.max(10, Number(e.target.value)) : null)} />
        </label>
        <label>Longest side
          <select value={maxDimension ?? ""} onChange={(e) => setMaxDimension(e.target.value ? Number(e.target.value) : null)}>
            <option value="">Original size</option>
            <option value="3840">3840 px</option>
            <option value="2560">2560 px</option>
            <option value="1920">1920 px</option>
            <option value="1280">1280 px</option>
            <option value="800">800 px</option>
          </select>
        </label>
      </div>

      <p className="muted small" role="status" aria-live="polite">{notice}</p>

      {items.length > 0 && (
        <>
          <div className="actions">
            <button type="button" className="btn btn-primary" onClick={run} disabled={busy} data-testid="compress">
              {busy ? "Compressing…" : done.length ? "Compress again" : "Compress"}
            </button>
            {done.length > 1 && <button type="button" className="btn" onClick={downloadAll}>Download all (ZIP)</button>}
            <button type="button" className="btn btn-plain" disabled={busy} onClick={() => { setItems([]); setNotice(""); }}>Clear</button>
          </div>
          {done.length > 0 && (
            <p className="summary" data-testid="summary">
              {formatBytes(before)} → <strong>{formatBytes(after)}</strong> {formatChange(before, after) && `(${formatChange(before, after)})`}
            </p>
          )}
          <ul className="results">
            {items.map((i) => (
              <li key={i.id} className="row" data-status={i.status}>
                <span className="name" title={i.file.name}>{i.file.name}</span>
                <span className="muted small">{formatBytes(i.file.size)}</span>
                {i.status === "queued" && <span className="muted small">Ready</span>}
                {i.status === "working" && <span className="small">Working…</span>}
                {i.status === "error" && <span className="error small">{i.error}</span>}
                {i.status === "done" && i.result && (
                  <>
                    <span className="small">
                      → <strong>{formatBytes(i.result.blob.size)}</strong> {formatChange(i.file.size, i.result.blob.size)}
                      {i.result.keptOriginal && " (already optimal, kept as is)"}
                      {!i.result.reachedTarget && " (smallest possible; try WebP or a smaller size)"}
                    </span>
                    <a className="btn btn-small" href={i.url} download={outputName(i.file.name, i.result.type)}>Download</a>
                  </>
                )}
              </li>
            ))}
          </ul>
        </>
      )}
      <p className="muted small privacy-note">Your images never leave your device: they are compressed in your browser.</p>
    </section>
  );
}
