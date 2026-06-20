import { storage } from "../utils/storage";
import i18n from "../i18n";

const DEFAULT_API_BASE_URL = "/api/v1";

function normalizeBaseUrl(baseUrl: string): string {
  return baseUrl.endsWith("/") ? baseUrl.slice(0, -1) : baseUrl;
}

const API_BASE_URL = normalizeBaseUrl(import.meta.env.VITE_API_BASE_URL ?? DEFAULT_API_BASE_URL);

type UnauthorizedCallback = () => void;

let onUnauthorized: UnauthorizedCallback | null = null;

export class ApiError extends Error {
  status: number;
  code?: string;

  constructor(message: string, status: number, code?: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

interface RequestOptions extends RequestInit {
  skipAuth?: boolean;
}

export function setUnauthorizedCallback(callback: UnauthorizedCallback | null): void {
  onUnauthorized = callback;
}

function getFallbackErrorMessage(status: number): string {
  if (status >= 500) {
    return i18n.t("system.serverUnavailable");
  }

  return i18n.t("system.unableToConnect");
}

function parseResponseBody(raw: string): unknown {
  if (!raw) {
    return null;
  }

  try {
    return JSON.parse(raw) as unknown;
  } catch {
    return null;
  }
}

export async function apiRequest<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers = new Headers(options.headers ?? {});
  headers.set("Content-Type", "application/json");

  if (!options.skipAuth) {
    const token = storage.getAccessToken();
    if (token) {
      headers.set("Authorization", `Bearer ${token}`);
    }
  }

  let response: Response;

  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      ...options,
      headers,
    });
  } catch {
    throw new ApiError(i18n.t("system.unableToConnect"), 0);
  }

  const raw = await response.text();
  const data = parseResponseBody(raw) as { data?: T; error?: { message?: string; code?: string } } | null;
  const code = data?.error?.code;

  if (!response.ok) {
    if ((response.status === 401 || (response.status === 403 && code === "inactive_user")) && !options.skipAuth) {
      onUnauthorized?.();
    }

    const message = data?.error?.message || getFallbackErrorMessage(response.status);
    throw new ApiError(message, response.status, code);
  }

  return data?.data as T;
}
