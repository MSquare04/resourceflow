import { useAuth } from "./AuthContext";

interface UseRolesResult {
  hasRole: (role: string) => boolean;
  hasAnyRole: (roles: string[]) => boolean;
}

export function useRoles(): UseRolesResult {
  const { user } = useAuth();
  const userRoles = user?.roles ?? [];

  const hasRole = (role: string): boolean => userRoles.some((userRole) => userRole === role);
  const hasAnyRole = (roles: string[]): boolean => roles.some((role) => hasRole(role));

  return {
    hasRole,
    hasAnyRole,
  };
}
