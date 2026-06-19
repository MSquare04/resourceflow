import { useTranslation } from "react-i18next";

export function BookingRulesPage(): JSX.Element {
  const { t } = useTranslation();

  return (
    <section>
      <h2>{t("pages.bookingRules.title")}</h2>
      <p>{t("pages.bookingRules.description")}</p>
    </section>
  );
}
