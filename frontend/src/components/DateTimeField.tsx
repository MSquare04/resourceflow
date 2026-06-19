import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

interface DateTimeFieldProps {
  label: string;
  value: string;
  minValue?: string;
  required?: boolean;
  disabled?: boolean;
  onApply: (value: string) => void;
}

function splitDateTime(value: string): { date: string; time: string } {
  if (!value || !value.includes("T")) {
    return { date: "", time: "" };
  }

  const [date, timePart] = value.split("T");
  return {
    date,
    time: timePart.slice(0, 5),
  };
}

export function DateTimeField({
  label,
  value,
  minValue,
  required = false,
  disabled = false,
  onApply,
}: DateTimeFieldProps): JSX.Element {
  const { t } = useTranslation();
  const [draftDate, setDraftDate] = useState("");
  const [draftTime, setDraftTime] = useState("");

  useEffect(() => {
    const nextValue = splitDateTime(value);
    setDraftDate(nextValue.date);
    setDraftTime(nextValue.time);
  }, [value]);

  const minParts = useMemo(() => splitDateTime(minValue ?? ""), [minValue]);
  const draftValue = draftDate && draftTime ? `${draftDate}T${draftTime}` : "";
  const canApply =
    !disabled &&
    !!draftDate &&
    !!draftTime &&
    (!minValue || draftValue >= minValue);

  return (
    <fieldset className="field date-time-field" disabled={disabled}>
      <legend>{label}</legend>

      <div className="date-time-field__grid">
        <label className="field">
          <span>{t("common.date")}</span>
          <input
            type="date"
            value={draftDate}
            min={minParts.date || undefined}
            onChange={(event) => setDraftDate(event.target.value)}
            required={required}
          />
        </label>

        <label className="field">
          <span>{t("common.time")}</span>
          <input
            type="time"
            value={draftTime}
            min={draftDate && minParts.date === draftDate ? minParts.time || undefined : undefined}
            onChange={(event) => setDraftTime(event.target.value)}
            required={required}
          />
        </label>
      </div>

      <div className="date-time-field__actions">
        <button type="button" className="btn btn-secondary" onClick={() => onApply(draftValue)} disabled={!canApply}>
          {t("common.apply")}
        </button>
        <span className="date-time-field__value muted">
          {value ? value.replace("T", " ") : t("common.notApplied")}
        </span>
      </div>
    </fieldset>
  );
}
