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
