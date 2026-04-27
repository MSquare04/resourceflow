import { storage } from "./storage";

export function isAuthenticated(): boolean {
  return Boolean(storage.getAccessToken());
}

export function logout(): void {
  storage.clearAccessToken();
}