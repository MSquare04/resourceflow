import { useTranslation } from "react-i18next";

interface LoadingStateProps {
  message?: string;
}

export function LoadingState({ message }: LoadingStateProps): JSX.Element {
  const { t } = useTranslation();

  return (
    <div className="feedback-card" role="status" aria-live="polite">
      <h3>{message ?? t("loading.title")}</h3>
      <p className="muted">{t("loading.description")}</p>
    </div>
  );
}
