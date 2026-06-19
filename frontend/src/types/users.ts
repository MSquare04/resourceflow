import type { RoleCode } from "./auth";

export interface User {
  id: number;
  full_name: string;
  email: string;
  department_id: number | null;
  is_active: boolean;
  roles: RoleCode[];
}

export interface CreateUserPayload {
  full_name: string;
  email: string;
  password: string;
  department_id: number | null;
  is_active: boolean;
  roles: RoleCode[];
}

export interface UpdateUserPayload {
  full_name: string;
  email: string;
  password: string;
  department_id: number | null;
  is_active: boolean;
}

export interface Department {
  id: number;
  name: string;
  description: string;
  is_active: boolean;
}
