import i18n from "../i18n";

function getCurrentLocale(): string {
  const currentLanguage = i18n.resolvedLanguage ?? i18n.language;

  if (currentLanguage === "en") {
    return "en-US";
  }

  return "ru-RU";
}

export function getCurrentDateTimeLocale(): string {
  return getCurrentLocale();
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

export function formatLocalDate(date: Date, locale?: string): string {
  return new Intl.DateTimeFormat(locale ?? getCurrentLocale(), {
    dateStyle: "medium",
  }).format(date);
}

export function formatDisplayDate(date: Date, locale?: string): string {
  return new Intl.DateTimeFormat(locale ?? getCurrentLocale(), {
    day: "numeric",
    month: "long",
    year: "numeric",
  }).format(date);
}

export function formatLocalTime(date: Date, locale?: string): string {
  return new Intl.DateTimeFormat(locale ?? getCurrentLocale(), {
    timeStyle: "short",
  }).format(date);
}
