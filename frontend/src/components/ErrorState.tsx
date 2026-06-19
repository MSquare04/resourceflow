import { useTranslation } from "react-i18next";

interface ErrorStateProps {
  message: string;
  onRetry?: () => void;
}

export function ErrorState({ message, onRetry }: ErrorStateProps): JSX.Element {
  const { t } = useTranslation();

  return (
    <div className="feedback-card" role="alert">
      <h3>{t("errors.title")}</h3>
      <p className="error-text">{message}</p>
      {onRetry ? (
        <button type="button" className="btn btn-primary" onClick={onRetry}>
          {t("common.retry")}
        </button>
      ) : null}
    </div>
  );
}
