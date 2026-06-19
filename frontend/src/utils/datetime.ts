import i18n from "../i18n";

function getCurrentLocale(): string {
  const currentLanguage = i18n.resolvedLanguage ?? i18n.language;

  if (currentLanguage === "en") {
    return "en-US";
  }

  return "ru-RU";
}

export function formatUtcDateTime(utcDateTime: string, locale?: string): string {
  const value = new Date(utcDateTime);
  if (Number.isNaN(value.getTime())) {
    return utcDateTime;
  }

  return new Intl.DateTimeFormat(locale ?? getCurrentLocale(), {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(value);
}
