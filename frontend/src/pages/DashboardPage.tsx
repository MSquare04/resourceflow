import { useTranslation } from "react-i18next";

import { formatUtcDateTime } from "../utils/datetime";

export function DashboardPage(): JSX.Element {
  const { t } = useTranslation();

  return (
    <section>
      <h2>{t("pages.dashboard.title")}</h2>
      <p>{t("pages.dashboard.description")}</p>
      <p className="muted">
        {t("pages.dashboard.utcSample")} {formatUtcDateTime(new Date().toISOString())}
      </p>
    </section>
  );
}
