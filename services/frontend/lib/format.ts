export function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(n < 10 * 1024 ? 1 : 0)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}

/** "-42%" when the output is smaller, "+8%" when larger, "" when nothing changed. */
export function formatChange(before: number, after: number): string {
  if (before <= 0 || before === after) return "";
  const pct = Math.round(Math.abs(1 - after / before) * 100);
  return `${after < before ? "-" : "+"}${pct}%`;
}
