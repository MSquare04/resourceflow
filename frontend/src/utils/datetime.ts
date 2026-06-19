export function formatUtcDateTime(utcDateTime: string, locale?: string): string {
  const value = new Date(utcDateTime);
  if (Number.isNaN(value.getTime())) {
    return utcDateTime;
  }

  return new Intl.DateTimeFormat(locale ?? undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(value);
}