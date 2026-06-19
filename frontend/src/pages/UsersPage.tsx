import { useTranslation } from "react-i18next";

export function UsersPage(): JSX.Element {
  const { t } = useTranslation();

  return (
    <section>
      <h2>{t("pages.users.title")}</h2>
      <p>{t("pages.users.description")}</p>
    </section>
  );
}
