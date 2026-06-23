import { useEffect, useId, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";

interface DatePickerProps {
  value: string;
  onChange: (value: string) => void;
  minValue?: string;
  maxValue?: string;
  disabled?: boolean;
  required?: boolean;
  ariaLabel?: string;
}

interface CalendarDay {
  key: string;
  value: string;
  dayNumber: number;
  outsideMonth: boolean;
  isToday: boolean;
  isSelected: boolean;
  isDisabled: boolean;
}

function CalendarIcon(): JSX.Element {
  return (
    <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden="true">
      <rect x="3" y="4" width="18" height="18" rx="3" />
      <path d="M8 2v4M16 2v4M3 10h18" />
    </svg>
  );
}

function ChevronLeftIcon(): JSX.Element {
  return (
    <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden="true">
      <path d="m15 18-6-6 6-6" />
    </svg>
  );
}

function ChevronRightIcon(): JSX.Element {
  return (
    <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden="true">
      <path d="m9 18 6-6-6-6" />
    </svg>
  );
}

function pad(value: number): string {
  return String(value).padStart(2, "0");
}

function getDateValue(date: Date): string {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

function parseDateValue(value: string): Date | null {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    return null;
  }

  const [year, month, day] = value.split("-").map(Number);
  const date = new Date(year, month - 1, day, 0, 0, 0, 0);
  return Number.isNaN(date.getTime()) ? null : date;
}

function getMonthTitle(date: Date, locale: string): string {
  return new Intl.DateTimeFormat(locale, { month: "long", year: "numeric" }).format(date);
}

function getWeekdayLabels(locale: string): string[] {
  const mondayBasedStart = new Date(2024, 0, 1);
  return Array.from({ length: 7 }, (_, index) =>
    new Intl.DateTimeFormat(locale, { weekday: "short" }).format(new Date(mondayBasedStart.getFullYear(), mondayBasedStart.getMonth(), mondayBasedStart.getDate() + index)),
  );
}

function getCalendarDays(viewMonth: Date, selectedValue: string, minValue?: string, maxValue?: string): CalendarDay[] {
  const monthStart = new Date(viewMonth.getFullYear(), viewMonth.getMonth(), 1);
  const monthStartWeekday = (monthStart.getDay() + 6) % 7;
  const gridStart = new Date(monthStart);
  gridStart.setDate(monthStart.getDate() - monthStartWeekday);
  const todayValue = getDateValue(new Date());

  return Array.from({ length: 42 }, (_, index) => {
    const date = new Date(gridStart);
    date.setDate(gridStart.getDate() + index);
    const value = getDateValue(date);

    return {
      key: value,
      value,
      dayNumber: date.getDate(),
      outsideMonth: date.getMonth() !== viewMonth.getMonth(),
      isToday: value === todayValue,
      isSelected: value === selectedValue,
      isDisabled: (minValue !== undefined && value < minValue) || (maxValue !== undefined && value > maxValue),
    };
  });
}

export function DatePicker({
  value,
  onChange,
  minValue,
  maxValue,
  disabled = false,
  required = false,
  ariaLabel,
}: DatePickerProps): JSX.Element {
  const { i18n, t } = useTranslation();
  const triggerRef = useRef<HTMLButtonElement | null>(null);
  const popoverRef = useRef<HTMLDivElement | null>(null);
  const [isOpen, setIsOpen] = useState(false);
  const [position, setPosition] = useState({ top: 0, left: 0, width: 0 });
  const popoverId = useId();
  const selectedDate = value ? parseDateValue(value) : null;
  const [viewMonth, setViewMonth] = useState<Date>(() => selectedDate ?? parseDateValue(minValue ?? "") ?? new Date());

  useEffect(() => {
    if (selectedDate) {
      setViewMonth(new Date(selectedDate.getFullYear(), selectedDate.getMonth(), 1));
    }
  }, [selectedDate]);

  useEffect(() => {
    if (!isOpen) {
      return;
    }

    const updatePosition = (): void => {
      const rect = triggerRef.current?.getBoundingClientRect();
      if (!rect) {
        return;
      }

      const maxLeft = Math.max(12, window.innerWidth - 296);
      setPosition({
        top: rect.bottom + 8,
        left: Math.min(rect.left, maxLeft),
        width: rect.width,
      });
    };

    updatePosition();
    window.addEventListener("resize", updatePosition);
    window.addEventListener("scroll", updatePosition, true);

    return () => {
      window.removeEventListener("resize", updatePosition);
      window.removeEventListener("scroll", updatePosition, true);
    };
  }, [isOpen]);

  useEffect(() => {
    if (!isOpen) {
      return;
    }

    const handlePointerDown = (event: MouseEvent): void => {
      const target = event.target as Node;
      if (triggerRef.current?.contains(target) || popoverRef.current?.contains(target)) {
        return;
      }

      setIsOpen(false);
    };

    const handleKeyDown = (event: KeyboardEvent): void => {
      if (event.key === "Escape") {
        setIsOpen(false);
        triggerRef.current?.focus();
      }
    };

    document.addEventListener("mousedown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [isOpen]);

  const locale = i18n.language === "en" ? "en-US" : "ru-RU";
  const weekdayLabels = useMemo(() => getWeekdayLabels(locale), [locale]);
  const calendarDays = useMemo(
    () => getCalendarDays(viewMonth, value, minValue, maxValue),
    [maxValue, minValue, value, viewMonth],
  );
  const monthTitle = getMonthTitle(viewMonth, locale);
  const previousMonth = new Date(viewMonth.getFullYear(), viewMonth.getMonth() - 1, 1);
  const nextMonth = new Date(viewMonth.getFullYear(), viewMonth.getMonth() + 1, 1);
  const displayValue = value ? formatDateValue(value, locale) : "";

  function handleSelect(nextValue: string): void {
    onChange(nextValue);
    setIsOpen(false);
    triggerRef.current?.focus();
  }

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        className="picker-field"
        onClick={() => setIsOpen((current) => !current)}
        disabled={disabled}
        aria-haspopup="dialog"
        aria-expanded={isOpen}
        aria-controls={isOpen ? popoverId : undefined}
        aria-label={ariaLabel}
        data-required={required || undefined}
      >
        <span className="picker-field__icon" aria-hidden="true">
          <CalendarIcon />
        </span>
        <span className={`picker-field__value ${displayValue ? "" : "is-placeholder"}`}>
          {displayValue || t("pages.resourceDetails.busy.selectedDate")}
        </span>
      </button>

      {isOpen
        ? createPortal(
            <div
              ref={popoverRef}
              id={popoverId}
              className="picker-popover date-picker-popover"
              role="dialog"
              aria-modal="false"
              style={{ top: `${position.top}px`, left: `${position.left}px`, minWidth: `${Math.max(position.width, 280)}px` }}
            >
              <div className="date-picker__header">
                <button
                  type="button"
                  className="picker-icon-button"
                  onClick={() => setViewMonth(previousMonth)}
                  aria-label={getMonthTitle(previousMonth, locale)}
                  title={getMonthTitle(previousMonth, locale)}
                >
                  <ChevronLeftIcon />
                </button>
                <strong className="date-picker__title">{monthTitle}</strong>
                <button
                  type="button"
                  className="picker-icon-button"
                  onClick={() => setViewMonth(nextMonth)}
                  aria-label={getMonthTitle(nextMonth, locale)}
                  title={getMonthTitle(nextMonth, locale)}
                >
                  <ChevronRightIcon />
                </button>
              </div>

              <div className="date-picker__weekdays" aria-hidden="true">
                {weekdayLabels.map((label) => (
                  <span key={label}>{label}</span>
                ))}
              </div>

              <div className="date-picker__grid" role="grid" aria-label={monthTitle}>
                {calendarDays.map((day) => (
                  <button
                    key={day.key}
                    type="button"
                    role="gridcell"
                    className={[
                      "date-picker__day",
                      day.outsideMonth ? "is-outside" : "",
                      day.isToday ? "is-today" : "",
                      day.isSelected ? "is-selected" : "",
                    ]
                      .filter(Boolean)
                      .join(" ")}
                    onClick={() => handleSelect(day.value)}
                    disabled={day.isDisabled}
                    aria-selected={day.isSelected}
                  >
                    {day.dayNumber}
                  </button>
                ))}
              </div>
            </div>,
            document.body,
          )
        : null}
    </>
  );
}

function formatDateValue(value: string, locale: string): string {
  const date = parseDateValue(value);
  if (!date) {
    return value;
  }

  return new Intl.DateTimeFormat(locale, { day: "2-digit", month: "2-digit", year: "numeric" }).format(date);
}
