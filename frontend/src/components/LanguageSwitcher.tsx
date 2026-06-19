import { useTranslation } from "react-i18next";

import { changeAppLanguage, type SupportedLanguage, supportedLanguages } from "../i18n";

interface LanguageSwitcherProps {
  variant?: "default" | "dropdown";
}

export function LanguageSwitcher({ variant = "default" }: LanguageSwitcherProps): JSX.Element {
  const { i18n, t } = useTranslation();
  const currentLanguage = (i18n.resolvedLanguage ?? "ru") as SupportedLanguage;
  const switcher = (
    <div
      className={`language-switcher language-switcher--${variant}`}
      role="group"
      aria-label={t("language.switcherLabel")}
    >
      {supportedLanguages.map((language) => {
        const isActive = currentLanguage === language;

        return (
          <button
            key={language}
            type="button"
            className={`language-switcher__button ${isActive ? "active" : ""}`}
            onClick={() => void changeAppLanguage(language)}
            aria-pressed={isActive}
          >
            {t(`language.${language}`)}
          </button>
        );
      })}
    </div>
  );

  if (variant === "dropdown") {
    return (
      <div className="language-switcher-row">
        <span className="language-switcher-row__label">{t("language.label")}</span>
        {switcher}
      </div>
    );
  }

  return switcher;
}
