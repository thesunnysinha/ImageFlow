import type { Envelope, FieldError, Job, JobSummary, Page } from "./types";

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
    readonly details: FieldError[] = [],
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export interface ClientOptions {
  baseUrl: string;
  apiKey: string;
  fetch?: typeof fetch; // injectable for tests
}

export class ApiClient {
  private readonly base: string;
  private readonly doFetch: typeof fetch;

  constructor(private readonly opts: ClientOptions) {
    this.base = opts.baseUrl.replace(/\/+$/, "") + "/api/v1";
    this.doFetch = opts.fetch ?? fetch;
  }

  /** Headers an <Image> needs to download a protected output. */
  get authHeaders(): Record<string, string> {
    return { Authorization: `Bearer ${this.opts.apiKey}` };
  }

  outputUrl(jobId: string, position: number): string {
    return `${this.base}/jobs/${encodeURIComponent(jobId)}/items/${position}/output`;
  }

  async listJobs(opts: { limit?: number; before?: string | null } = {}): Promise<Page<JobSummary>> {
    const query = new URLSearchParams();
    if (opts.limit) query.set("limit", String(opts.limit));
    if (opts.before) query.set("before", opts.before);
    const qs = query.toString();
    const { data, meta } = await this.request<JobSummary[]>(`/jobs${qs ? "?" + qs : ""}`);
    const next = meta["next_before"];
    return { items: data, nextBefore: typeof next === "string" ? next : null };
  }

  async createJob(input: { sourceUrls: string[]; webhookUrl?: string }): Promise<Job> {
    const body: Record<string, unknown> = { source_urls: input.sourceUrls };
    if (input.webhookUrl) body["webhook_url"] = input.webhookUrl;
    const { data } = await this.request<Job>("/jobs", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    return data;
  }

  async getJob(id: string): Promise<Job> {
    const { data } = await this.request<Job>(`/jobs/${encodeURIComponent(id)}`);
    return data;
  }

  private async request<T>(path: string, init: RequestInit = {}): Promise<{ data: T; meta: Record<string, unknown> }> {
    let response: Response;
    try {
      response = await this.doFetch(this.base + path, {
        ...init,
        headers: { Accept: "application/json", ...this.authHeaders, ...(init.headers as Record<string, string> | undefined) },
      });
    } catch {
      throw new ApiError(0, "NETWORK_ERROR", "Could not reach the server. Check the address and your connection.");
    }
    let envelope: Envelope<T> | null = null;
    try {
      envelope = (await response.json()) as Envelope<T>;
    } catch {
      // Not JSON (a proxy or gateway error page): fall through to the generic error below.
    }
    if (!envelope || typeof envelope !== "object" || !("success" in envelope)) {
      throw new ApiError(response.status, "BAD_RESPONSE", `Unexpected response from the server (HTTP ${response.status}).`);
    }
    if (!response.ok || !envelope.success || envelope.data === null) {
      const details = Array.isArray(envelope.meta?.["details"]) ? (envelope.meta["details"] as FieldError[]) : [];
      throw new ApiError(response.status, envelope.code, envelope.message, details);
    }
    return { data: envelope.data, meta: envelope.meta ?? {} };
  }
}
