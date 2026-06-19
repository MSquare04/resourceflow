import { useTranslation } from "react-i18next";
import { Navigate, Outlet, useLocation } from "react-router-dom";

import { useAuth } from "../auth/AuthContext";

export function ProtectedRoute(): JSX.Element {
  const location = useLocation();
  const { t } = useTranslation();
  const { isAuthenticated, isInitializing, refreshSession, sessionError } = useAuth();

  if (isInitializing) {
    return (
      <div className="system-state-shell">
        <section className="system-state-card">
          <h2>{t("system.checkingSession")}</h2>
          <p className="muted">{t("system.verifyingAccess")}</p>
        </section>
      </div>
    );
  }

  if (sessionError) {
    return (
      <div className="system-state-shell">
        <section className="system-state-card">
          <h2>{t("system.sessionCheckFailed")}</h2>
          <p className="error-text">{sessionError}</p>
          <button type="button" className="btn btn-primary" onClick={() => void refreshSession()}>
            {t("common.retry")}
          </button>
        </section>
      </div>
    );
  }

  if (!isAuthenticated) {
    const from = `${location.pathname}${location.search}${location.hash}`;
    return <Navigate to="/login" replace state={{ from }} />;
  }

  return <Outlet />;
}
