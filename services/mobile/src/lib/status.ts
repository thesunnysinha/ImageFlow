import type { Counts, ItemStatus, JobStatus } from "../api/types";

export function isTerminal(status: JobStatus): boolean {
  return status === "completed" || status === "partial" || status === "failed";
}

export const statusLabel: Record<JobStatus | ItemStatus, string> = {
  queued: "Queued",
  processing: "Processing",
  completed: "Done",
  partial: "Partly done",
  failed: "Failed",
};

/** Share of images that have finished (successfully or not), 0..1. */
export function progress(counts: Counts): number {
  return counts.total === 0 ? 0 : (counts.completed + counts.failed) / counts.total;
}

export function formatBytes(n: number | undefined): string {
  if (n === undefined) return "";
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}

/** "-42%" style saving, or "" when it is not smaller. */
export function savings(before: number | undefined, after: number | undefined): string {
  if (!before || after === undefined || after >= before) return "";
  return `-${Math.round((1 - after / before) * 100)}%`;
}
