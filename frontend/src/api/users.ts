import type { RoleCode } from "../types/auth";
import type { CreateUserPayload, UpdateUserPayload, User } from "../types/users";
import { apiRequest } from "./client";

export function listUsers(): Promise<User[]> {
  return apiRequest<User[]>("/users");
}

export function createUser(payload: CreateUserPayload): Promise<User> {
  return apiRequest<User>("/users", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function updateUser(id: number, payload: UpdateUserPayload): Promise<User> {
  return apiRequest<User>(`/users/${id}`, {
    method: "PUT",
    body: JSON.stringify(payload),
  });
}

export function replaceUserRoles(id: number, roles: RoleCode[]): Promise<User> {
  return apiRequest<User>(`/users/${id}/roles`, {
    method: "PUT",
    body: JSON.stringify({ roles }),
  });
}
