import type { Department } from "../types/users";
import { apiRequest } from "./client";

export function listDepartments(): Promise<Department[]> {
  return apiRequest<Department[]>("/departments");
}
