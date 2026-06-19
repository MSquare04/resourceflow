import i18n from "i18next";
import { initReactI18next } from "react-i18next";

import { en } from "./locales/en";
import { ru } from "./locales/ru";

export const LANGUAGE_STORAGE_KEY = "resourceflow_language";

export const supportedLanguages = ["ru", "en"] as const;
export type SupportedLanguage = (typeof supportedLanguages)[number];

function getInitialLanguage(): SupportedLanguage {
  if (typeof window === "undefined") {
    return "ru";
  }

  const storedLanguage = window.localStorage.getItem(LANGUAGE_STORAGE_KEY);
  if (storedLanguage === "ru" || storedLanguage === "en") {
    return storedLanguage;
  }

  return "ru";
}

void i18n.use(initReactI18next).init({
  resources: {
    ru: { translation: ru },
    en: { translation: en },
  },
  lng: getInitialLanguage(),
  fallbackLng: "ru",
  interpolation: {
    escapeValue: false,
  },
});

export async function changeAppLanguage(language: SupportedLanguage): Promise<void> {
  await i18n.changeLanguage(language);
  if (typeof window !== "undefined") {
    window.localStorage.setItem(LANGUAGE_STORAGE_KEY, language);
  }
}

export default i18n;
