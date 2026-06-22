import { FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";

import { changePassword } from "../api/auth";
import { ApiError } from "../api/client";
import { useAuth } from "../auth/AuthContext";
import { PageHeader } from "../components/PageHeader";

function mapChangePasswordError(error: ApiError, t: ReturnType<typeof useTranslation>["t"]): string {
  switch (error.code) {
    case "current_password_invalid":
      return t("pages.changePassword.errors.currentPasswordInvalid");
    case "new_password_same_as_current":
      return t("pages.changePassword.errors.sameAsCurrent");
    case "password_policy_violation":
      return t("pages.changePassword.errors.passwordPolicyViolation");
    default:
      return error.message || t("pages.changePassword.errors.generic");
  }
}

export function ChangePasswordPage(): JSX.Element {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { logout } = useAuth();
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>): Promise<void> => {
    event.preventDefault();

    const trimmedCurrentPassword = currentPassword.trim();
    const trimmedNewPassword = newPassword.trim();
    const trimmedConfirmPassword = confirmPassword.trim();

    if (!trimmedCurrentPassword) {
      setError(t("pages.changePassword.errors.currentPasswordRequired"));
      return;
    }

    if (!trimmedNewPassword) {
      setError(t("pages.changePassword.errors.newPasswordRequired"));
      return;
    }

    if (!trimmedConfirmPassword) {
      setError(t("pages.changePassword.errors.confirmPasswordRequired"));
      return;
    }

    if (trimmedNewPassword !== trimmedConfirmPassword) {
      setError(t("pages.changePassword.errors.confirmPasswordMismatch"));
      return;
    }

    setError(null);
    setIsSubmitting(true);

    try {
      await changePassword({
        current_password: trimmedCurrentPassword,
        new_password: trimmedNewPassword,
      });
      logout();
      navigate("/login", { replace: true, state: { passwordChanged: true } });
    } catch (submitError) {
      if (submitError instanceof ApiError) {
        setError(mapChangePasswordError(submitError, t));
      } else if (submitError instanceof Error) {
        setError(submitError.message);
      } else {
        setError(t("pages.changePassword.errors.generic"));
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <section>
      <PageHeader title={t("pages.changePassword.title")} />
      <div className="settings-form-panel">
        <form className="form-grid" onSubmit={handleSubmit}>
          <label className="field">
            <span>{t("pages.changePassword.fields.currentPassword")}</span>
            <input
              type="password"
              value={currentPassword}
              onChange={(event) => setCurrentPassword(event.target.value)}
              autoComplete="current-password"
              required
            />
          </label>

          <label className="field">
            <span>{t("pages.changePassword.fields.newPassword")}</span>
            <input
              type="password"
              value={newPassword}
              onChange={(event) => setNewPassword(event.target.value)}
              autoComplete="new-password"
              required
            />
          </label>

          <label className="field">
            <span>{t("pages.changePassword.fields.confirmPassword")}</span>
            <input
              type="password"
              value={confirmPassword}
              onChange={(event) => setConfirmPassword(event.target.value)}
              autoComplete="new-password"
              required
            />
          </label>

          {error ? <p className="error-text">{error}</p> : null}

          <div className="settings-form-panel__actions">
            <button type="submit" className="btn btn-primary" disabled={isSubmitting}>
              {isSubmitting ? t("pages.changePassword.submitting") : t("pages.changePassword.submit")}
            </button>
          </div>
        </form>
      </div>
    </section>
  );
}
