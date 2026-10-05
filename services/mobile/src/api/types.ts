// Shapes of the ImageFlow API (services/backend/openapi.yaml). Every response uses the envelope.

export type JobStatus = "queued" | "processing" | "completed" | "partial" | "failed";
export type ItemStatus = "queued" | "processing" | "completed" | "failed";

export interface Counts {
  total: number;
  completed: number;
  failed: number;
}

export interface JobSummary {
  id: string;
  status: JobStatus;
  created_at: string;
  updated_at: string;
  counts: Counts;
}

export interface Item {
  id: string;
  position: number;
  source_url: string;
  status: ItemStatus;
  output_key?: string;
  bytes_in?: number;
  bytes_out?: number;
  error?: string;
}

export interface Job {
  id: string;
  status: JobStatus;
  webhook_url?: string;
  created_at: string;
  updated_at: string;
  items: Item[];
}

export interface FieldError {
  field: string;
  message: string;
}

export interface Envelope<T> {
  success: boolean;
  code: string;
  message: string;
  data: T | null;
  meta: Record<string, unknown>;
  trace_id: string;
}

export interface Page<T> {
  items: T[];
  nextBefore: string | null;
}
