export interface ParsedUrls {
  urls: string[];
  invalid: string[];
}

/** Splits pasted text on whitespace, commas and semicolons; keeps http(s) URLs once each, in order. */
export function parseUrlList(text: string): ParsedUrls {
  const seen = new Set<string>();
  const urls: string[] = [];
  const invalid: string[] = [];
  for (const raw of text.split(/[\s,;]+/)) {
    const token = raw.trim();
    if (!token) continue;
    if (!isHttpUrl(token)) {
      invalid.push(token);
      continue;
    }
    if (!seen.has(token)) {
      seen.add(token);
      urls.push(token);
    }
  }
  return { urls, invalid };
}

export function isHttpUrl(value: string): boolean {
  try {
    const u = new URL(value);
    return (u.protocol === "http:" || u.protocol === "https:") && u.hostname.length > 0;
  } catch {
    return false;
  }
}

/** Accepts "api.example.com" or "https://api.example.com/" and returns "https://api.example.com", or null. */
export function normalizeBaseUrl(input: string): string | null {
  const trimmed = input.trim();
  if (!trimmed) return null;
  const withScheme = /^[a-z][a-z0-9+.-]*:\/\//i.test(trimmed) ? trimmed : `https://${trimmed}`;
  if (!isHttpUrl(withScheme)) return null;
  const u = new URL(withScheme);
  return (u.origin + u.pathname).replace(/\/+$/, "");
}
