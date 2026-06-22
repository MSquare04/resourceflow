export type RoleCode = "admin" | "manager" | "employee" | "hr" | "interviewer";

export interface AuthUser {
  id: number;
  full_name: string;
  email: string;
  is_active: boolean;
  roles: RoleCode[];
}

export interface LoginResponse {
  access_token: string;
  token_type: "Bearer";
  expires_at: string;
  user: AuthUser;
}

export interface CurrentUserResponse {
  user: AuthUser;
}

export interface ChangePasswordPayload {
  current_password: string;
  new_password: string;
}
