import { useEffect, useId, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";

interface TimePickerProps {
  value: string;
  onChange: (value: string) => void;
  minValue?: string;
  maxValue?: string;
  disabled?: boolean;
  required?: boolean;
  minuteStep?: number;
  ariaLabel?: string;
}

function ClockIcon(): JSX.Element {
  return (
    <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden="true">
      <circle cx="12" cy="12" r="9" />
      <path d="M12 7v5l3 3" />
    </svg>
  );
}

function pad(value: number): string {
  return String(value).padStart(2, "0");
}

function parseTime(value: string): { hours: number; minutes: number } | null {
  const match = /^(\d{2}):(\d{2})$/.exec(value);
  if (!match) {
    return null;
  }

  const hours = Number(match[1]);
  const minutes = Number(match[2]);
  if (!Number.isInteger(hours) || !Number.isInteger(minutes) || hours < 0 || hours > 23 || minutes < 0 || minutes > 59) {
    return null;
  }

  return { hours, minutes };
}

function toTimeValue(hours: number, minutes: number): string {
  return `${pad(hours)}:${pad(minutes)}`;
}

function clampNumber(value: string, min: number, max: number): number {
  const parsed = Number(value);
  if (Number.isNaN(parsed)) {
    return min;
  }

  return Math.min(Math.max(Math.trunc(parsed), min), max);
}

export function TimePicker({
  value,
  onChange,
  minValue,
  maxValue,
  disabled = false,
  required = false,
  minuteStep = 5,
  ariaLabel,
}: TimePickerProps): JSX.Element {
  const { t } = useTranslation();
  const triggerRef = useRef<HTMLButtonElement | null>(null);
  const popoverRef = useRef<HTMLDivElement | null>(null);
  const [isOpen, setIsOpen] = useState(false);
  const [position, setPosition] = useState({ top: 0, left: 0, width: 0 });
  const [draftHours, setDraftHours] = useState("00");
  const [draftMinutes, setDraftMinutes] = useState("00");
  const popoverId = useId();

  useEffect(() => {
    const parsed = parseTime(value);
    setDraftHours(parsed ? pad(parsed.hours) : "00");
    setDraftMinutes(parsed ? pad(parsed.minutes) : "00");
  }, [value]);

  useEffect(() => {
    if (!isOpen) {
      return;
    }

    const updatePosition = (): void => {
      const rect = triggerRef.current?.getBoundingClientRect();
      if (!rect) {
        return;
      }

      const maxLeft = Math.max(12, window.innerWidth - 320);
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

  const minTime = minValue ? parseTime(minValue) : null;
  const maxTime = maxValue ? parseTime(maxValue) : null;
  const hours = useMemo(() => Array.from({ length: 24 }, (_, index) => pad(index)), []);
  const minutes = useMemo(() => Array.from({ length: 60 }, (_, index) => pad(index)), []);
  const currentValue = toTimeValue(clampNumber(draftHours, 0, 23), clampNumber(draftMinutes, 0, 59));

  function isDisabledTime(nextValue: string): boolean {
    return (minValue !== undefined && nextValue < minValue) || (maxValue !== undefined && nextValue > maxValue);
  }

  function handleQuickUpdate(next: Partial<{ hours: string; minutes: string }>): void {
    const nextHours = next.hours ?? draftHours;
    const nextMinutes = next.minutes ?? draftMinutes;
    const nextValue = toTimeValue(clampNumber(nextHours, 0, 23), clampNumber(nextMinutes, 0, 59));

    setDraftHours(nextHours);
    setDraftMinutes(nextMinutes);

    if (!isDisabledTime(nextValue)) {
      onChange(nextValue);
    }
  }

  function handleManualApply(): void {
    const nextValue = toTimeValue(clampNumber(draftHours, 0, 23), clampNumber(draftMinutes, 0, 59));
    setDraftHours(nextValue.slice(0, 2));
    setDraftMinutes(nextValue.slice(3, 5));

    if (!isDisabledTime(nextValue)) {
      onChange(nextValue);
      setIsOpen(false);
      triggerRef.current?.focus();
    }
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
          <ClockIcon />
        </span>
        <span className={`picker-field__value ${value ? "" : "is-placeholder"}`}>{value || t("common.time")}</span>
      </button>

      {isOpen
        ? createPortal(
            <div
              ref={popoverRef}
              id={popoverId}
              className="picker-popover time-picker-popover"
              role="dialog"
              aria-modal="false"
              style={{ top: `${position.top}px`, left: `${position.left}px`, minWidth: `${Math.max(position.width, 304)}px` }}
            >
              <div className="time-picker__columns">
                <div className="time-picker__column" aria-label={t("common.time")}>
                  {hours.map((hour) => {
                    const nextValue = `${hour}:${draftMinutes}`;
                    return (
                      <button
                        key={hour}
                        type="button"
                        className={`time-picker__option ${draftHours === hour ? "is-selected" : ""}`}
                        onClick={() => handleQuickUpdate({ hours: hour })}
                        disabled={isDisabledTime(nextValue)}
                      >
                        {hour}
                      </button>
                    );
                  })}
                </div>

                <div className="time-picker__column" aria-label={t("common.time")}>
                  {minutes.map((minute) => {
                    const nextValue = `${draftHours}:${minute}`;
                    return (
                      <button
                        key={minute}
                        type="button"
                        className={[
                          "time-picker__option",
                          draftMinutes === minute ? "is-selected" : "",
                          minuteStep > 1 && Number(minute) % minuteStep === 0 ? "is-suggested" : "",
                        ]
                          .filter(Boolean)
                          .join(" ")}
                        onClick={() => handleQuickUpdate({ minutes: minute })}
                        disabled={isDisabledTime(nextValue)}
                      >
                        {minute}
                      </button>
                    );
                  })}
                </div>
              </div>

              <div className="time-picker__manual">
                <input
                  type="number"
                  min={minTime?.hours ?? 0}
                  max={maxTime?.hours ?? 23}
                  value={draftHours}
                  onChange={(event) => setDraftHours(pad(clampNumber(event.target.value, 0, 23)))}
                  aria-label={t("common.time")}
                />
                <span className="time-picker__separator">:</span>
                <input
                  type="number"
                  min={0}
                  max={59}
                  value={draftMinutes}
                  onChange={(event) => setDraftMinutes(pad(clampNumber(event.target.value, 0, 59)))}
                  aria-label={t("common.time")}
                />
                <button type="button" className="btn btn-secondary" onClick={handleManualApply} disabled={isDisabledTime(currentValue)}>
                  {t("common.apply")}
                </button>
              </div>
            </div>,
            document.body,
          )
        : null}
    </>
  );
}
