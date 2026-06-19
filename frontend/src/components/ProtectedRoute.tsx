import { Navigate, Outlet, useLocation } from "react-router-dom";

import { isAuthenticated } from "../utils/auth";

export function ProtectedRoute(): JSX.Element {
  const location = useLocation();

  if (!isAuthenticated()) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  }

  return <Outlet />;
}