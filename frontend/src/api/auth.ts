import type { ChangePasswordPayload, CurrentUserResponse, LoginResponse } from "../types/auth";
import { apiRequest } from "./client";

interface LoginPayload {
  email: string;
  password: string;
}

export function login(payload: LoginPayload): Promise<LoginResponse> {
  return apiRequest<LoginResponse>("/auth/login", {
    method: "POST",
    skipAuth: true,
    body: JSON.stringify(payload),
  });
}

export function getCurrentUser(): Promise<CurrentUserResponse> {
  return apiRequest<CurrentUserResponse>("/auth/me");
}

export async function changePassword(payload: ChangePasswordPayload): Promise<void> {
  await apiRequest<Record<string, never>>("/auth/change-password", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}
