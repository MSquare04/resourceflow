import { Navigate, Outlet, useLocation } from "react-router-dom";

import { useAuth } from "../auth/AuthContext";

export function ProtectedRoute(): JSX.Element {
  const location = useLocation();
  const { isAuthenticated, isInitializing, refreshSession, sessionError } = useAuth();

  if (isInitializing) {
    return (
      <section>
        <h2>Checking session</h2>
        <p className="muted">Verifying your access...</p>
      </section>
    );
  }

  if (sessionError) {
    return (
      <section>
        <h2>Session check failed</h2>
        <p className="error-text">{sessionError}</p>
        <button type="button" className="btn btn-primary" onClick={() => void refreshSession()}>
          Retry
        </button>
      </section>
    );
  }

  if (!isAuthenticated) {
    const from = `${location.pathname}${location.search}${location.hash}`;
    return <Navigate to="/login" replace state={{ from }} />;
  }

  return <Outlet />;
}
