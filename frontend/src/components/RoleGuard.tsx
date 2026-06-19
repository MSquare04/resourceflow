import { Navigate, Outlet, useLocation } from "react-router-dom";

import { useRoles } from "../auth/useRoles";

interface RoleGuardProps {
  allowedRoles: string[];
}

export function RoleGuard({ allowedRoles }: RoleGuardProps): JSX.Element {
  const location = useLocation();
  const { hasAnyRole } = useRoles();

  if (!hasAnyRole(allowedRoles)) {
    return <Navigate to="/forbidden" replace state={{ from: location.pathname }} />;
  }

  return <Outlet />;
}
