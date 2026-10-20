interface ApiEnvelope<T> {
  success: boolean;
  message: string;
  data?: T;
  error?: {
    code: string;
    details: string[];
  };
}

export class ApiError extends Error {
  status: number;
  code: string;
  details: string[];

  constructor(status: number, code: string, message: string, details: string[] = []) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.details = details;
  }
}

const API_BASE_URL =
  (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "http://localhost:8080/api/v1";

/**
 * Encodes a value that is interpolated into a single URL path segment. The
 * backend always validates identifiers, but encoding here guarantees a value
 * can never add path separators or query syntax to the request URL.
 */
export const encodePathSegment = (value: string): string => encodeURIComponent(value);

export async function apiGet<T>(path: string): Promise<T> {
  let response: Response;

  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      method: "GET",
      credentials: "include"
    });
  } catch {
    throw new ApiError(0, "NETWORK_ERROR", "Unable to reach the server.");
  }

  return handleResponse<T>(response);
}

export async function apiPost<T>(path: string, body: unknown): Promise<T> {
  let response: Response;

  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body)
    });
  } catch {
    throw new ApiError(0, "NETWORK_ERROR", "Unable to reach the server.");
  }

  return handleResponse<T>(response);
}

// apiUpload posts a multipart body. The browser sets the multipart boundary
// and content type; the caller only provides the FormData.
export async function apiUpload<T>(path: string, body: FormData): Promise<T> {
  let response: Response;

  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      method: "POST",
      credentials: "include",
      body
    });
  } catch {
    throw new ApiError(0, "NETWORK_ERROR", "Unable to reach the server.");
  }

  return handleResponse<T>(response);
}

const CONTENT_DISPOSITION_FILENAME = /filename\*?=(?:UTF-8''|")?([^";]+)/i;

const filenameFromDisposition = (disposition: string, fallback: string): string => {
  const match = CONTENT_DISPOSITION_FILENAME.exec(disposition);
  if (!match) {
    return fallback;
  }
  try {
    return decodeURIComponent(match[1].trim());
  } catch {
    return fallback;
  }
};

// downloadFile streams an authenticated binary download to the browser. Error
// responses are still JSON envelopes, so they are mapped to ApiError exactly
// like the JSON helpers.
export async function downloadFile(path: string, fallbackName: string): Promise<void> {
  let response: Response;

  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      method: "GET",
      credentials: "include"
    });
  } catch {
    throw new ApiError(0, "NETWORK_ERROR", "Unable to reach the server.");
  }

  if (!response.ok) {
    const envelope = (await response.json().catch(() => null)) as ApiEnvelope<unknown> | null;
    throw new ApiError(
      response.status,
      envelope?.error?.code ?? "API_ERROR",
      envelope?.message ?? "Unable to download the file.",
      envelope?.error?.details ?? []
    );
  }

  const blob = await response.blob();
  const filename = filenameFromDisposition(response.headers.get("Content-Disposition") ?? "", fallbackName);
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}

export async function apiPatch<T>(path: string, body: unknown): Promise<T> {
  let response: Response;

  try {
    const res = await fetch(`${API_BASE_URL}${path}`, {
      method: "PATCH",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body)
    });
    response = res;
  } catch {
    throw new ApiError(0, "NETWORK_ERROR", "Unable to reach the server.");
  }

  return handleResponse<T>(response);
}

export async function apiDelete<T>(path: string): Promise<T> {
  let response: Response;

  try {
    const res = await fetch(`${API_BASE_URL}${path}`, {
      method: "DELETE",
      credentials: "include"
    });
    response = res;
  } catch {
    throw new ApiError(0, "NETWORK_ERROR", "Unable to reach the server.");
  }

  return handleResponse<T>(response);
}

async function handleResponse<T>(response: Response): Promise<T> {
  const envelope = (await response.json().catch(() => null)) as ApiEnvelope<T> | null;

  if (!envelope) {
    throw new ApiError(response.status, "NETWORK_ERROR", "Unexpected server response.");
  }

  if (!envelope.success) {
    throw new ApiError(
      response.status,
      envelope.error?.code ?? "API_ERROR",
      envelope.message,
      envelope.error?.details ?? []
    );
  }

  return envelope.data as T;
}
