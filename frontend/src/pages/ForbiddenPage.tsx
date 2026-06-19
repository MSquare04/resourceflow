import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";

export function ForbiddenPage(): JSX.Element {
  const { t } = useTranslation();

  return (
    <div className="system-state-shell">
      <section className="system-state-card">
        <h2>{t("pages.forbidden.code")}</h2>
        <p className="muted">{t("pages.forbidden.title")}</p>
        <Link to="/" className="btn btn-primary">
          {t("pages.forbidden.back")}
        </Link>
      </section>
    </div>
  );
}
