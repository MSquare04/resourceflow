import { useTranslation } from "react-i18next";

export function MyBookingsPage(): JSX.Element {
  const { t } = useTranslation();

  return (
    <section>
      <h2>{t("pages.myBookings.title")}</h2>
      <p>{t("pages.myBookings.description")}</p>
    </section>
  );
}
