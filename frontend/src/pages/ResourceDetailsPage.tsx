import { FormEvent, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link, useNavigate, useParams } from "react-router-dom";

import { createBatchBookings, createBooking, previewBatchBookings } from "../api/bookings";
import { listBookingRules } from "../api/bookingRules";
import { ApiError } from "../api/client";
import { listDepartments } from "../api/departments";
import {
  createResourceUnavailability as createResourceAvailability,
  deleteResourceUnavailability as deleteResourceAvailability,
  getResource,
  listResourceBusyIntervalsInRange,
  listResourceCategories,
  listResourceUnavailability as listResourceAvailability,
  listResourceTypes,
  updateResourceUnavailability as updateResourceAvailability,
} from "../api/resources";
import { useRoles } from "../auth/useRoles";
import { DatePicker } from "../components/DatePicker";
import { DateTimeField } from "../components/DateTimeField";
import { ErrorState } from "../components/ErrorState";
import { LoadingState } from "../components/LoadingState";
import { MultiDatePicker } from "../components/MultiDatePicker";
import { PageHeader } from "../components/PageHeader";
import { TimePicker } from "../components/TimePicker";
import type { BookingRule } from "../types/bookingRules";
import type { BatchBookingPreviewResponse } from "../types/bookings";
import type {
  Resource,
  ResourceBusyInterval,
  ResourceCategory,
  ResourceType,
  ResourceUnavailability as ResourceAvailability,
} from "../types/resources";
import type { Department } from "../types/users";
import { formatDisplayDate, formatLocalDate, formatLocalTime, formatUtcDateTime } from "../utils/datetime";

type AvailabilityFormMode = "create" | "edit";
type BookingMode = "single" | "multiple";

function mapBookingError(error: ApiError, t: ReturnType<typeof useTranslation>["t"]): string {
  if (error.code === "conflict" || error.status === 409) {
    if (error.message === "resource is inactive or not bookable") {
      return t("pages.resourceDetails.booking.errors.resourceUnavailable");
    }
    if (error.message === "booking interval intersects resource unavailability") {
      return t("pages.resourceDetails.booking.errors.unavailabilityConflict");
    }

    return t("pages.resourceDetails.booking.errors.conflict");
  }

  const message = error.message;
  switch (message) {
    case "invalid booking payload":
      return t("pages.resourceDetails.booking.errors.invalidPayload");
    case "booking start time cannot be earlier than the current minute":
      return t("pages.resourceDetails.booking.errors.startInPast");
    case "resource not found":
      return t("pages.resourceDetails.booking.errors.resourceNotFound");
    case "booking interval is outside booking rule workday":
      return t("pages.resourceDetails.booking.errors.outsideWorkday");
    case "active booking rule is not configured":
      return t("pages.resourceDetails.booking.errors.ruleNotConfigured");
    case "max active bookings per user exceeded":
      return t("pages.resourceDetails.booking.errors.limitExceeded");
    case "booking horizon exceeded":
      return t("pages.resourceDetails.booking.errors.horizonExceeded");
    case "booking conflicts with existing active booking":
      return t("pages.resourceDetails.booking.errors.conflict");
    default:
      return message;
  }
}

function mapAvailabilityError(error: ApiError, t: ReturnType<typeof useTranslation>["t"]): string {
  if (error.code === "conflict" || error.status === 409) {
    return t("pages.resourceDetails.availability.errors.activeBookingConflict");
  }

  const message = error.message;
  switch (message) {
    case "invalid resource unavailability payload":
      return t("pages.resourceDetails.availability.errors.invalidPayload");
    case "resource not found":
      return t("pages.resourceDetails.availability.errors.resourceNotFound");
    case "resource unavailability not found":
      return t("pages.resourceDetails.availability.errors.availabilityNotFound");
    default:
      return message;
  }
}

function toDateTimeLocalValue(value: Date): string {
  const year = value.getFullYear();
  const month = String(value.getMonth() + 1).padStart(2, "0");
  const day = String(value.getDate()).padStart(2, "0");
  const hours = String(value.getHours()).padStart(2, "0");
  const minutes = String(value.getMinutes()).padStart(2, "0");

  return `${year}-${month}-${day}T${hours}:${minutes}`;
}

function toLocalInputValue(isoString: string): string {
  return toDateTimeLocalValue(new Date(isoString));
}

function getCurrentLocalMinute(): Date {
  const current = new Date();
  current.setSeconds(0, 0);
  return current;
}

function getLocalDateKey(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");

  return `${year}-${month}-${day}`;
}

function parseLocalDateKey(value: string): Date {
  const [year, month, day] = value.split("-").map(Number);
  return new Date(year, (month || 1) - 1, day || 1, 0, 0, 0, 0);
}

function addMinutes(date: Date, minutes: number): Date {
  return new Date(date.getTime() + minutes * 60_000);
}

function buildSequentialDateKeys(startDateKey: string, days: number): string[] {
  const result: string[] = [];
  const startDate = parseLocalDateKey(startDateKey);
  for (let index = 0; index < days; index += 1) {
    const nextDate = new Date(startDate);
    nextDate.setDate(startDate.getDate() + index);
    result.push(getLocalDateKey(nextDate));
  }
  return result;
}

function getTimePart(dateTimeValue: string): string {
  if (!dateTimeValue.includes("T")) {
    return "";
  }
  return dateTimeValue.slice(11, 16);
}

function intervalsIntersect(startAt: number, endAt: number, otherStartAt: number, otherEndAt: number): boolean {
  return startAt < otherEndAt && endAt > otherStartAt;
}

function getEffectiveMinDurationMinutes(rule: BookingRule | null): number {
  const minDurationMinutes = rule?.min_duration_minutes ?? 0;
  return minDurationMinutes > 0 ? minDurationMinutes : 30;
}

function getQuickSelectionStepMinutes(rule: BookingRule | null): number {
  const minDurationMinutes = rule?.min_duration_minutes ?? 0;

  if (minDurationMinutes >= 15) {
    return minDurationMinutes;
  }

  if (minDurationMinutes >= 1) {
    return 15;
  }

  return 30;
}

function parseTimeToMinutes(value: string): number | null {
  const match = /^(\d{2}):(\d{2})/.exec(value);
  if (!match) {
    return null;
  }

  const hours = Number(match[1]);
  const minutes = Number(match[2]);
  if (!Number.isInteger(hours) || !Number.isInteger(minutes) || hours < 0 || hours > 23 || minutes < 0 || minutes > 59) {
    return null;
  }

  return hours * 60 + minutes;
}

type DaySlotState = "free" | "busy" | "unavailable" | "past" | "selected";

interface DaySlot {
  key: string;
  startAt: Date;
  endAt: Date;
  label: string;
  state: DaySlotState;
  disabled: boolean;
}

function EditIcon(): JSX.Element {
  return (
    <svg
      viewBox="0 0 24 24"
      width="16"
      height="16"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M12 20h9" />
      <path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4 12.5-12.5Z" />
    </svg>
  );
}

export function ResourceDetailsPage(): JSX.Element {
  const { id } = useParams();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { hasRole } = useRoles();
  const isAdmin = hasRole("admin");
  const resourceId = Number(id);

  const [resource, setResource] = useState<Resource | null>(null);
  const [categories, setCategories] = useState<ResourceCategory[]>([]);
  const [types, setTypes] = useState<ResourceType[]>([]);
  const [departments, setDepartments] = useState<Department[]>([]);
  const [availability, setAvailability] = useState<ResourceAvailability[]>([]);
  const [bookingRules, setBookingRules] = useState<BookingRule[]>([]);
  const [busyIntervals, setBusyIntervals] = useState<ResourceBusyInterval[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedDate, setSelectedDate] = useState(() => getLocalDateKey(new Date()));
  const [startAt, setStartAt] = useState("");
  const [endAt, setEndAt] = useState("");
  const [purpose, setPurpose] = useState("");
  const [bookingMode, setBookingMode] = useState<BookingMode>("single");
  const [batchSelectedDates, setBatchSelectedDates] = useState<string[]>(() => [getLocalDateKey(new Date())]);
  const [selectedBatchPresetDays, setSelectedBatchPresetDays] = useState<number | null>(null);
  const [batchStartTime, setBatchStartTime] = useState("");
  const [batchEndTime, setBatchEndTime] = useState("");
  const [batchPreview, setBatchPreview] = useState<BatchBookingPreviewResponse | null>(null);
  const [batchFormError, setBatchFormError] = useState<string | null>(null);
  const [isBatchPreviewLoading, setIsBatchPreviewLoading] = useState(false);
  const [isBatchSubmitting, setIsBatchSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [busyLoading, setBusyLoading] = useState(true);
  const [busyError, setBusyError] = useState<string | null>(null);
  const [isAvailabilityFormOpen, setIsAvailabilityFormOpen] = useState(false);
  const [availabilityFormMode, setAvailabilityFormMode] = useState<AvailabilityFormMode>("create");
  const [editingAvailabilityId, setEditingAvailabilityId] = useState<number | null>(null);
  const [availabilityStartAt, setAvailabilityStartAt] = useState("");
  const [availabilityEndAt, setAvailabilityEndAt] = useState("");
  const [availabilityReason, setAvailabilityReason] = useState("");
  const [availabilityFormError, setAvailabilityFormError] = useState<string | null>(null);
  const [availabilityActionError, setAvailabilityActionError] = useState<string | null>(null);
  const [isAvailabilitySubmitting, setIsAvailabilitySubmitting] = useState(false);
  const [pendingAvailabilityId, setPendingAvailabilityId] = useState<number | null>(null);
  const availabilityFormRef = useRef<HTMLFormElement | null>(null);
  const bookingRuleActionButtonRef = useRef<HTMLButtonElement | null>(null);
  const batchPreviewRequestIdRef = useRef(0);
  const startAtMin = toDateTimeLocalValue(getCurrentLocalMinute());

  useEffect(() => {
    if (!batchStartTime) {
      const initialStart = getCurrentLocalMinute();
      setBatchStartTime(toDateTimeLocalValue(initialStart).slice(11, 16));
      setBatchEndTime(toDateTimeLocalValue(addMinutes(initialStart, 30)).slice(11, 16));
    }
  }, [batchEndTime, batchStartTime]);

  useEffect(() => {
    if (startAt) {
      setBatchStartTime(getTimePart(startAt));
    }
  }, [startAt]);

  useEffect(() => {
    if (endAt) {
      setBatchEndTime(getTimePart(endAt));
    }
  }, [endAt]);

  useEffect(() => {
    batchPreviewRequestIdRef.current += 1;
    setBatchPreview(null);
    setBatchFormError(null);
  }, [batchStartTime, batchEndTime, bookingMode, purpose, resourceId]);

  useEffect(() => {
    void loadResourceDetails();
  }, [isAdmin, resourceId]);

  useEffect(() => {
    if (!Number.isInteger(resourceId) || resourceId <= 0) {
      return;
    }

    void loadBusyIntervalsForSelectedDate();
  }, [resourceId, selectedDate]);

  async function loadResourceDetails(): Promise<void> {
    if (!Number.isInteger(resourceId) || resourceId <= 0) {
      setError(t("pages.resourceDetails.errors.invalidResource"));
      setLoading(false);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const [resourceData, categoriesData, typesData, availabilityData, bookingRulesData] = await Promise.all([
        getResource(resourceId),
        listResourceCategories(),
        listResourceTypes(),
        listResourceAvailability(resourceId),
        listBookingRules(),
      ]);
      let departmentsData: Department[] = [];

      if (isAdmin) {
        try {
          departmentsData = await listDepartments();
        } catch {
          departmentsData = [];
        }
      }

      setResource(resourceData);
      setCategories(categoriesData);
      setTypes(typesData);
      setDepartments(departmentsData);
      setAvailability(availabilityData);
      setBookingRules(bookingRulesData);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : t("errors.generic"));
    } finally {
      setLoading(false);
    }
  }

  async function loadBusyIntervalsForSelectedDate(): Promise<void> {
    const dayStart = parseLocalDateKey(selectedDate);
    const dayEnd = new Date(dayStart);
    dayEnd.setDate(dayEnd.getDate() + 1);

    setBusyLoading(true);
    setBusyError(null);

    try {
      const data = await listResourceBusyIntervalsInRange(resourceId, {
        from: dayStart.toISOString(),
        to: dayEnd.toISOString(),
      });
      setBusyIntervals(data);
    } catch (loadError) {
      setBusyIntervals([]);
      setBusyError(loadError instanceof Error ? loadError.message : t("errors.generic"));
    } finally {
      setBusyLoading(false);
    }
  }

  const categoryMap = useMemo(() => new Map(categories.map((category) => [category.id, category.name])), [categories]);
  const typeMap = useMemo(() => new Map(types.map((type) => [type.id, type.name])), [types]);
  const departmentMap = useMemo(
    () => new Map(departments.map((department) => [department.id, department.name])),
    [departments],
  );

  const uniqueAvailability = useMemo(() => {
    const uniqueSlots = new Map<string, ResourceAvailability>();

    for (const slot of availability) {
      const key = `${slot.start_at}__${slot.end_at}`;
      if (!uniqueSlots.has(key)) {
        uniqueSlots.set(key, slot);
      }
    }

    return [...uniqueSlots.values()].sort(
      (left, right) => new Date(left.start_at).getTime() - new Date(right.start_at).getTime(),
    );
  }, [availability]);

  const futureAvailability = useMemo(() => {
    const now = Date.now();

    return uniqueAvailability.filter((slot) => {
      const endTime = new Date(slot.end_at).getTime();
      return !Number.isNaN(endTime) && endTime > now;
    });
  }, [uniqueAvailability]);
  const visibleBusyIntervals = useMemo(() => {
    const uniqueIntervals = new Map<string, ResourceBusyInterval>();

    for (const interval of busyIntervals) {
      const key = `${interval.start_at}__${interval.end_at}`;
      if (!uniqueIntervals.has(key)) {
        uniqueIntervals.set(key, interval);
      }
    }

    return [...uniqueIntervals.values()].sort(
      (left, right) => new Date(left.start_at).getTime() - new Date(right.start_at).getTime(),
    );
  }, [busyIntervals]);
  const activeBookingRule = useMemo(() => {
    if (!resource) {
      return null;
    }

    return [...bookingRules]
      .filter((rule) => rule.resource_type_id === resource.type_id && rule.is_active)
      .sort((left, right) => right.id - left.id)[0] ?? null;
  }, [bookingRules, resource]);
  const hasAdditionalRestrictions = uniqueAvailability.length > 0;
  const bookingDisabled = !activeBookingRule || !resource?.is_active || !resource?.is_bookable;
  const bookingFormAvailable = !!activeBookingRule && !!resource?.is_active && !!resource?.is_bookable;
  const canSubmitBatch =
    batchSelectedDates.length > 0 &&
    batchPreview !== null &&
    batchPreview.can_create &&
    batchPreview.items.every((item) => item.valid) &&
    !isBatchPreviewLoading;
  const selectedDayStart = useMemo(() => parseLocalDateKey(selectedDate), [selectedDate]);
  const selectedDayEnd = useMemo(() => {
    const nextDay = new Date(selectedDayStart);
    nextDay.setDate(nextDay.getDate() + 1);
    return nextDay;
  }, [selectedDayStart]);
  const selectedDayLabel = useMemo(() => formatLocalDate(selectedDayStart), [selectedDayStart]);
  const workdayStartMinutes = useMemo(
    () => (activeBookingRule && !activeBookingRule.unrestricted_time ? parseTimeToMinutes(activeBookingRule.workday_start) : 0),
    [activeBookingRule],
  );
  const workdayEndMinutes = useMemo(
    () => (activeBookingRule && !activeBookingRule.unrestricted_time ? parseTimeToMinutes(activeBookingRule.workday_end) : 24 * 60),
    [activeBookingRule],
  );
  const hasValidWorkdayWindow =
    activeBookingRule === null ||
    activeBookingRule.unrestricted_time ||
    (workdayStartMinutes !== null && workdayEndMinutes !== null && workdayStartMinutes < workdayEndMinutes);

  const endAtMin = useMemo(() => {
    if (!startAt) {
      return startAtMin;
    }

    return startAt > startAtMin ? startAt : startAtMin;
  }, [startAt, startAtMin]);

  function isRangeInsideWorkday(startAtMs: number, endAtMs: number): boolean {
    if (!activeBookingRule || activeBookingRule.unrestricted_time) {
      return true;
    }

    if (workdayStartMinutes === null || workdayEndMinutes === null || workdayStartMinutes >= workdayEndMinutes) {
      return false;
    }

    const startValue = new Date(startAtMs);
    const endValue = new Date(endAtMs);
    if (
      startValue.getFullYear() !== endValue.getFullYear() ||
      startValue.getMonth() !== endValue.getMonth() ||
      startValue.getDate() !== endValue.getDate()
    ) {
      return false;
    }

    const startMinutes = startValue.getHours() * 60 + startValue.getMinutes();
    const endMinutes = endValue.getHours() * 60 + endValue.getMinutes();
    return startMinutes >= workdayStartMinutes && endMinutes <= workdayEndMinutes;
  }

  function intersectsAdditionalRestrictions(startAtMs: number, endAtMs: number): boolean {
    return uniqueAvailability.some((slot) => {
      const slotStart = new Date(slot.start_at).getTime();
      const slotEnd = new Date(slot.end_at).getTime();
      return intervalsIntersect(startAtMs, endAtMs, slotStart, slotEnd);
    });
  }

  function intersectsBusyIntervals(startAtMs: number, endAtMs: number): boolean {
    return visibleBusyIntervals.some((interval) => {
      const intervalStart = new Date(interval.start_at).getTime();
      const intervalEnd = new Date(interval.end_at).getTime();

      return intervalsIntersect(startAtMs, endAtMs, intervalStart, intervalEnd);
    });
  }

  const selectedRange = useMemo(() => {
    const startValue = startAt ? new Date(startAt) : null;
    const endValue = endAt ? new Date(endAt) : null;

    if (!startValue || !endValue || Number.isNaN(startValue.getTime()) || Number.isNaN(endValue.getTime()) || startValue >= endValue) {
      return null;
    }

    return {
      startAtMs: startValue.getTime(),
      endAtMs: endValue.getTime(),
    };
  }, [endAt, startAt]);

  const daySlots = useMemo((): DaySlot[] => {
    const slots: DaySlot[] = [];
    const currentMinuteMs = getCurrentLocalMinute().getTime();
    const minDurationMinutes = getEffectiveMinDurationMinutes(activeBookingRule);
    const quickSelectionStepMinutes = getQuickSelectionStepMinutes(activeBookingRule);
    const selectedDayEndMs = selectedDayEnd.getTime();
    const isToday = selectedDate === getLocalDateKey(new Date());
    const slotWindowStartMinutes =
      activeBookingRule && !activeBookingRule.unrestricted_time && workdayStartMinutes !== null ? workdayStartMinutes : 0;
    const slotWindowEndMinutes =
      activeBookingRule && !activeBookingRule.unrestricted_time && workdayEndMinutes !== null ? workdayEndMinutes : 24 * 60;

    for (
      let offsetMinutes = slotWindowStartMinutes;
      offsetMinutes < slotWindowEndMinutes;
      offsetMinutes += quickSelectionStepMinutes
    ) {
      const slotStart = addMinutes(selectedDayStart, offsetMinutes);
      const candidateEnd = addMinutes(slotStart, minDurationMinutes);
      const slotStartMs = slotStart.getTime();
      const candidateEndMs = candidateEnd.getTime();
      const isPast = slotStartMs < currentMinuteMs;

      if (isToday && isPast) {
        continue;
      }

      const isSelected =
        selectedRange !== null &&
        slotStartMs === selectedRange.startAtMs &&
        candidateEndMs === selectedRange.endAtMs;
      const isBusy = intersectsBusyIntervals(slotStartMs, candidateEndMs);
      const canFitRuleDuration =
        hasValidWorkdayWindow &&
        candidateEndMs <= selectedDayEndMs &&
        isRangeInsideWorkday(slotStartMs, candidateEndMs) &&
        slotStartMs >= currentMinuteMs &&
        !intersectsAdditionalRestrictions(slotStartMs, candidateEndMs) &&
        !intersectsBusyIntervals(slotStartMs, candidateEndMs);

      let state: DaySlotState = "free";
      if (isSelected) {
        state = "selected";
      } else if (isPast) {
        state = "past";
      } else if (isBusy) {
        state = "busy";
      } else if (!canFitRuleDuration) {
        state = "unavailable";
      }

      slots.push({
        key: `${selectedDate}-${offsetMinutes}`,
        startAt: slotStart,
        endAt: candidateEnd,
        label: formatLocalTime(slotStart),
        state,
        disabled: state !== "free",
      });
    }

    return slots;
  }, [
    activeBookingRule,
    hasValidWorkdayWindow,
    selectedDate,
    selectedDayEnd,
    selectedDayStart,
    selectedRange,
    visibleBusyIntervals,
    workdayEndMinutes,
    workdayStartMinutes,
  ]);

  function validateDateRange(): string | null {
    if (!startAt || !endAt) {
      return t("pages.resourceDetails.booking.errors.requiredDates");
    }

    const startValue = new Date(startAt);
    const endValue = new Date(endAt);

    if (Number.isNaN(startValue.getTime()) || Number.isNaN(endValue.getTime())) {
      return t("pages.resourceDetails.booking.errors.invalidDates");
    }

    if (startValue.getTime() < getCurrentLocalMinute().getTime()) {
      return t("pages.resourceDetails.booking.errors.startInPast");
    }

    if (startValue >= endValue) {
      return t("pages.resourceDetails.booking.errors.invalidRange");
    }

    if (!isRangeInsideWorkday(startValue.getTime(), endValue.getTime())) {
      return t("pages.resourceDetails.booking.errors.outsideWorkday");
    }

    if (intersectsAdditionalRestrictions(startValue.getTime(), endValue.getTime())) {
      return t("pages.resourceDetails.booking.errors.unavailabilityConflict");
    }

    if (intersectsBusyIntervals(startValue.getTime(), endValue.getTime())) {
      return t("pages.resourceDetails.booking.errors.busyConflict");
    }

    return null;
  }

  function handleQuickDateSelect(offsetDays: number): void {
    const nextDate = new Date();
    nextDate.setDate(nextDate.getDate() + offsetDays);
    nextDate.setHours(0, 0, 0, 0);
    setSelectedDate(getLocalDateKey(nextDate));
  }

  function handleSlotSelect(slot: DaySlot): void {
    if (!activeBookingRule || slot.disabled) {
      return;
    }

    const minDurationMinutes = getEffectiveMinDurationMinutes(activeBookingRule);
    const nextStartAt = toDateTimeLocalValue(slot.startAt);
    const nextEndAt = toDateTimeLocalValue(addMinutes(slot.startAt, minDurationMinutes));

    setSelectedDate(getLocalDateKey(slot.startAt));
    setStartAt(nextStartAt);
    setEndAt(nextEndAt);
    setFormError(null);
  }

  function toggleBatchDate(dateKey: string): void {
    batchPreviewRequestIdRef.current += 1;
    setSelectedBatchPresetDays(null);
    setBatchSelectedDates((current) => {
      const nextDates = current.includes(dateKey) ? current.filter((item) => item !== dateKey) : [...current, dateKey];
      return [...nextDates].sort();
    });
    setBatchPreview(null);
    setBatchFormError(null);
  }

  function handleBatchQuickRange(days: number): void {
    batchPreviewRequestIdRef.current += 1;
    setSelectedBatchPresetDays(days);
    setBatchSelectedDates(buildSequentialDateKeys(selectedDate, days));
    setBatchPreview(null);
    setBatchFormError(null);
  }

  function removeBatchDate(dateKey: string): void {
    batchPreviewRequestIdRef.current += 1;
    setSelectedBatchPresetDays(null);
    setBatchSelectedDates((current) => {
      const nextDates = current.filter((item) => item !== dateKey);

      if (nextDates.length === 0) {
        setBatchPreview({ can_create: false, items: [] });
        setBatchFormError(null);
        return nextDates;
      }

      if (batchPreview) {
        void runBatchPreview(nextDates);
      } else {
        setBatchPreview(null);
        setBatchFormError(null);
      }

      return nextDates;
    });
  }

  function validateBatchBooking(): string | null {
    if (batchSelectedDates.length === 0) {
      return t("pages.resourceDetails.booking.multiple.errors.noDates");
    }

    if (!batchStartTime || !batchEndTime) {
      return t("pages.resourceDetails.booking.errors.requiredDates");
    }

    if (batchStartTime >= batchEndTime) {
      return t("pages.resourceDetails.booking.errors.invalidRange");
    }

    return null;
  }

  function mapBatchPreviewCode(errorCode?: string): string {
    switch (errorCode) {
      case "booking_conflict":
        return t("pages.resourceDetails.booking.multiple.preview.conflict");
      case "booking_outside_workday":
        return t("pages.resourceDetails.booking.multiple.preview.outsideWorkday");
      case "booking_in_unavailability":
        return t("pages.resourceDetails.booking.multiple.preview.technicalRestriction");
      case "booking_horizon_exceeded":
        return t("pages.resourceDetails.booking.multiple.preview.horizonExceeded");
      case "booking_limit_exceeded":
        return t("pages.resourceDetails.booking.multiple.preview.limitExceeded");
      case "booking_start_in_past":
        return t("pages.resourceDetails.booking.multiple.preview.startInPast");
      case "booking_resource_unavailable":
        return t("pages.resourceDetails.booking.multiple.preview.resourceUnavailable");
      case "booking_rule_not_configured":
        return t("pages.resourceDetails.booking.multiple.preview.ruleNotConfigured");
      default:
        return t("pages.resourceDetails.booking.multiple.preview.invalid");
    }
  }

  async function runBatchPreview(dates: string[] = batchSelectedDates): Promise<void> {
    const validationError = validateBatchBooking();
    if (validationError) {
      setBatchFormError(validationError);
      return;
    }

    if (!resource) {
      setBatchFormError(t("pages.resourceDetails.booking.errors.resourceNotFound"));
      return;
    }

    setBatchFormError(null);
    setIsBatchPreviewLoading(true);
    const requestId = batchPreviewRequestIdRef.current + 1;
    batchPreviewRequestIdRef.current = requestId;

    try {
      const preview = await previewBatchBookings({
        resource_id: resource.id,
        dates,
        start_time: batchStartTime,
        end_time: batchEndTime,
        purpose: purpose.trim() ? purpose.trim() : null,
      });
      if (batchPreviewRequestIdRef.current === requestId) {
        setBatchPreview(preview);
      }
    } catch (submitError) {
      if (batchPreviewRequestIdRef.current === requestId) {
        if (submitError instanceof ApiError) {
          setBatchFormError(submitError.message);
        } else if (submitError instanceof Error) {
          setBatchFormError(submitError.message);
        } else {
          setBatchFormError(t("pages.resourceDetails.booking.errors.generic"));
        }
      }
    } finally {
      if (batchPreviewRequestIdRef.current === requestId) {
        setIsBatchPreviewLoading(false);
      }
    }
  }

  async function handleBatchPreview(): Promise<void> {
    await runBatchPreview();
  }

  function resetAvailabilityForm(): void {
    setAvailabilityFormMode("create");
    setEditingAvailabilityId(null);
    setAvailabilityStartAt("");
    setAvailabilityEndAt("");
    setAvailabilityReason("");
    setAvailabilityFormError(null);
  }

  function closeAvailabilityForm(): void {
    setIsAvailabilityFormOpen(false);
    resetAvailabilityForm();
  }

  function openAvailabilityCreateForm(): void {
    resetAvailabilityForm();
    setIsAvailabilityFormOpen(true);
  }

  function openAvailabilityEditForm(slot: ResourceAvailability): void {
    setAvailabilityFormMode("edit");
    setEditingAvailabilityId(slot.id);
    setAvailabilityStartAt(toLocalInputValue(slot.start_at));
    setAvailabilityEndAt(toLocalInputValue(slot.end_at));
    setAvailabilityReason(slot.reason ?? "");
    setAvailabilityFormError(null);
    setIsAvailabilityFormOpen(true);
  }

  useEffect(() => {
    if (!isAdmin || !isAvailabilityFormOpen) {
      return;
    }

    const frameId = window.requestAnimationFrame(() => {
      availabilityFormRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
    });

    return () => {
      window.cancelAnimationFrame(frameId);
    };
  }, [availabilityFormMode, editingAvailabilityId, isAdmin, isAvailabilityFormOpen]);

  function openBookingRuleEditor(): void {
    if (!resource) {
      return;
    }

    if (activeBookingRule) {
      navigate(`/booking-rules?edit=${activeBookingRule.id}`);
      return;
    }

    navigate("/booking-rules", {
      state: {
        openCreate: true,
        resourceTypeId: resource.type_id,
      },
    });
  }

  function validateAvailabilityForm(): string | null {
    if (!availabilityStartAt || !availabilityEndAt) {
      return t("pages.resourceDetails.availability.errors.requiredDates");
    }

    const startValue = new Date(availabilityStartAt);
    const endValue = new Date(availabilityEndAt);

    if (Number.isNaN(startValue.getTime()) || Number.isNaN(endValue.getTime())) {
      return t("pages.resourceDetails.availability.errors.invalidDates");
    }

    if (startValue >= endValue) {
      return t("pages.resourceDetails.availability.errors.invalidRange");
    }

    const duplicateExists = uniqueAvailability.some((slot) => {
      if (availabilityFormMode === "edit" && slot.id === editingAvailabilityId) {
        return false;
      }

      return slot.start_at === startValue.toISOString() && slot.end_at === endValue.toISOString();
    });

    if (duplicateExists) {
      return t("pages.resourceDetails.availability.errors.duplicate");
    }

    return null;
  }

  async function handleAvailabilitySubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();

    const validationError = validateAvailabilityForm();
    if (validationError) {
      setAvailabilityFormError(validationError);
      return;
    }

    setAvailabilityFormError(null);
    setAvailabilityActionError(null);
    setIsAvailabilitySubmitting(true);

    try {
      const payload = {
        start_at: new Date(availabilityStartAt).toISOString(),
        end_at: new Date(availabilityEndAt).toISOString(),
        reason: availabilityReason.trim() || null,
      };

      if (availabilityFormMode === "create") {
        await createResourceAvailability(resourceId, payload);
      } else if (editingAvailabilityId !== null) {
        await updateResourceAvailability(resourceId, editingAvailabilityId, payload);
      }

      await loadResourceDetails();
      closeAvailabilityForm();
    } catch (submitError) {
      if (submitError instanceof ApiError) {
        setAvailabilityFormError(mapAvailabilityError(submitError, t));
      } else if (submitError instanceof Error) {
        setAvailabilityFormError(submitError.message);
      } else {
        setAvailabilityFormError(t("pages.resourceDetails.availability.errors.generic"));
      }
    } finally {
      setIsAvailabilitySubmitting(false);
    }
  }

  async function handleAvailabilityDelete(slot: ResourceAvailability): Promise<void> {
    if (!window.confirm(t("pages.resourceDetails.availability.confirmations.delete", { start: formatUtcDateTime(slot.start_at) }))) {
      return;
    }

    setAvailabilityActionError(null);
    setPendingAvailabilityId(slot.id);

    try {
      await deleteResourceAvailability(resourceId, slot.id);
      await loadResourceDetails();

      if (editingAvailabilityId === slot.id) {
        closeAvailabilityForm();
      }
    } catch (deleteError) {
      if (deleteError instanceof ApiError) {
        setAvailabilityActionError(mapAvailabilityError(deleteError, t));
      } else if (deleteError instanceof Error) {
        setAvailabilityActionError(deleteError.message);
      } else {
        setAvailabilityActionError(t("pages.resourceDetails.availability.errors.generic"));
      }
    } finally {
      setPendingAvailabilityId(null);
    }
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();

    const validationError = validateDateRange();
    if (validationError) {
      setFormError(validationError);
      return;
    }

    if (!resource) {
      setFormError(t("pages.resourceDetails.booking.errors.resourceNotFound"));
      return;
    }

    setFormError(null);
    setIsSubmitting(true);

    try {
      await createBooking({
        resource_id: resource.id,
        start_at: new Date(startAt).toISOString(),
        end_at: new Date(endAt).toISOString(),
        purpose: purpose.trim() ? purpose.trim() : null,
      });

      window.alert(t("pages.resourceDetails.booking.success"));
      navigate("/my-bookings", { replace: true });
    } catch (submitError) {
      if (submitError instanceof ApiError) {
        setFormError(mapBookingError(submitError, t));
      } else if (submitError instanceof Error) {
        setFormError(submitError.message);
      } else {
        setFormError(t("pages.resourceDetails.booking.errors.generic"));
      }
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleBatchCreate(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();

    const validationError = validateBatchBooking();
    if (validationError) {
      setBatchFormError(validationError);
      return;
    }

    if (!resource) {
      setBatchFormError(t("pages.resourceDetails.booking.errors.resourceNotFound"));
      return;
    }

    if (!batchPreview || !batchPreview.can_create || batchPreview.items.some((item) => !item.valid)) {
      setBatchFormError(t("pages.resourceDetails.booking.multiple.errors.previewRequired"));
      return;
    }

    setBatchFormError(null);
    setIsBatchSubmitting(true);

    try {
      const result = await createBatchBookings({
        resource_id: resource.id,
        dates: batchSelectedDates,
        start_time: batchStartTime,
        end_time: batchEndTime,
        purpose: purpose.trim() ? purpose.trim() : null,
      });

      window.alert(t("pages.resourceDetails.booking.multiple.success", { count: result.created_count }));
      setBatchPreview(null);
      setBatchSelectedDates([selectedDate]);
      await Promise.all([loadResourceDetails(), loadBusyIntervalsForSelectedDate()]);
    } catch (submitError) {
      if (submitError instanceof ApiError) {
        setBatchFormError(submitError.message);
      } else if (submitError instanceof Error) {
        setBatchFormError(submitError.message);
      } else {
        setBatchFormError(t("pages.resourceDetails.booking.errors.generic"));
      }
    } finally {
      setIsBatchSubmitting(false);
    }
  }

  if (loading) {
    return (
      <section>
        <LoadingState message={t("pages.resourceDetails.loading")} />
      </section>
    );
  }

  if (error || !resource) {
    return (
      <section>
        <PageHeader
          title={t("pages.resourceDetails.title")}
          actions={
            <Link to="/resources" className="btn btn-secondary">
              {t("pages.resourceDetails.back")}
            </Link>
          }
        />
        <ErrorState message={error ?? t("pages.resourceDetails.errors.notFound")} onRetry={() => void loadResourceDetails()} />
      </section>
    );
  }

  const displayedAvailability = isAdmin ? uniqueAvailability : futureAvailability;
  const departmentName =
    resource.department_id !== null
      ? (departmentMap.get(resource.department_id) ?? t("pages.resources.unknownDepartment"))
      : t("pages.resources.noDepartment");

  return (
    <section>
      <PageHeader
        title={resource.name}
        actions={
          <Link to="/resources" className="btn btn-secondary">
            {t("pages.resourceDetails.back")}
          </Link>
        }
      />

      <div className="resource-details-page">
        <div className="resource-details-card">
          <div className="resource-details-card__header">
            <div className="resource-details-card__heading">
              <h3 className="resource-details-card__title">{t("pages.resourceDetails.overview")}</h3>
              <p className="resource-details-card__description">
                {resource.description || t("pages.resources.noDescription")}
              </p>
            </div>
            <div className="resource-card__badges">
              <span className={`badge ${resource.is_active ? "badge-success" : "badge-muted"}`}>
                {resource.is_active ? t("pages.resources.active.true") : t("pages.resources.active.false")}
              </span>
              <span className={`badge ${resource.is_bookable ? "badge-info" : "badge-muted"}`}>
                {resource.is_bookable ? t("pages.resources.bookable.true") : t("pages.resources.bookable.false")}
              </span>
            </div>
          </div>

          <dl className="resource-details-meta">
            <div>
              <dt>{t("pages.resources.fields.category")}</dt>
              <dd>{categoryMap.get(resource.category_id) ?? t("pages.resources.unknownCategory")}</dd>
            </div>
            <div>
              <dt>{t("pages.resources.fields.type")}</dt>
              <dd>{typeMap.get(resource.type_id) ?? t("pages.resources.unknownType")}</dd>
            </div>
            <div>
              <dt>{t("pages.resources.fields.location")}</dt>
              <dd>{resource.location || t("pages.resources.notSpecified")}</dd>
            </div>
            <div>
              <dt>{t("pages.resources.fields.capacity")}</dt>
              <dd>
                {resource.capacity !== null
                  ? t("pages.resources.capacityValue", { value: resource.capacity })
                  : t("pages.resources.notSpecified")}
              </dd>
            </div>
            {isAdmin ? (
              <div>
                <dt>{t("pages.resources.fields.department")}</dt>
                <dd>{departmentName}</dd>
              </div>
            ) : null}
          </dl>
        </div>

        <div className="resource-details-grid">
          <div className="resource-details-card resource-details-card--rule">
            <div className="resource-details-card__header">
              <div className="resource-details-card__heading">
                <h3 className="resource-details-card__title">{t("pages.resourceDetails.rule.title")}</h3>
              </div>
              {isAdmin ? (
                <button
                  type="button"
                  ref={bookingRuleActionButtonRef}
                  className="btn btn-secondary btn-icon"
                  onClick={openBookingRuleEditor}
                >
                  <EditIcon />
                  <span>
                    {activeBookingRule
                      ? t("pages.resourceDetails.rule.actions.edit")
                      : t("pages.resourceDetails.rule.actions.configure")}
                  </span>
                </button>
              ) : null}
            </div>

            {activeBookingRule ? (
              <dl className="resource-details-rule-meta">
                <div className="resource-details-rule-meta__item">
                  <dt className="resource-details-rule-meta__label">{t("pages.resourceDetails.rule.fields.minDuration")}</dt>
                  <dd className="resource-details-rule-meta__value">
                    {t("pages.resourceDetails.rule.values.minutes", { count: activeBookingRule.min_duration_minutes })}
                  </dd>
                </div>
                <div className="resource-details-rule-meta__item">
                  <dt className="resource-details-rule-meta__label">{t("pages.resourceDetails.rule.fields.maxDuration")}</dt>
                  <dd className="resource-details-rule-meta__value">
                    {t("pages.resourceDetails.rule.values.minutes", { count: activeBookingRule.max_duration_minutes })}
                  </dd>
                </div>
                <div className="resource-details-rule-meta__item">
                  <dt className="resource-details-rule-meta__label">{t("pages.resourceDetails.rule.fields.horizon")}</dt>
                  <dd className="resource-details-rule-meta__value">
                    {t("pages.resourceDetails.rule.values.days", { count: activeBookingRule.booking_horizon_days })}
                  </dd>
                </div>
                <div className="resource-details-rule-meta__item">
                  <dt className="resource-details-rule-meta__label">{t("pages.resourceDetails.rule.fields.limit")}</dt>
                  <dd className="resource-details-rule-meta__value">
                    {t("pages.resourceDetails.rule.values.limit", { count: activeBookingRule.max_active_bookings_per_user })}
                  </dd>
                </div>
                <div className="resource-details-rule-meta__item">
                  <dt className="resource-details-rule-meta__label">{t("pages.resourceDetails.rule.fields.approval")}</dt>
                  <dd className="resource-details-rule-meta__value">
                    {activeBookingRule.requires_approval
                      ? t("pages.resourceDetails.rule.values.approvalRequired")
                      : t("pages.resourceDetails.rule.values.approvalNotRequired")}
                  </dd>
                </div>
                <div className="resource-details-rule-meta__item">
                  <dt className="resource-details-rule-meta__label">{t("pages.resourceDetails.rule.fields.workday")}</dt>
                  <dd className="resource-details-rule-meta__value">
                    {activeBookingRule.unrestricted_time
                      ? t("pages.resourceDetails.rule.values.unrestrictedTime")
                      : t("pages.resourceDetails.rule.values.workdayValue", {
                          start: activeBookingRule.workday_start,
                          end: activeBookingRule.workday_end,
                        })}
                  </dd>
                </div>
              </dl>
            ) : (
              <p className="muted resource-details-hint">{t("pages.resourceDetails.rule.missing")}</p>
            )}
          </div>

          <div className="resource-details-card resource-details-card--availability">
            <div className="resource-details-card__header">
              <div className="resource-details-card__heading">
                <h3 className="resource-details-card__title">{t("pages.resourceDetails.availability.title")}</h3>
              </div>
              {isAdmin ? (
                <button type="button" className="btn btn-secondary" onClick={openAvailabilityCreateForm}>
                  {t("pages.resourceDetails.availability.actions.create")}
                </button>
              ) : null}
            </div>
            <p className="muted resource-details-hint">{t("pages.resourceDetails.availability.hint")}</p>

            {isAdmin && isAvailabilityFormOpen ? (
              <form ref={availabilityFormRef} className="resource-availability-form" onSubmit={handleAvailabilitySubmit}>
                <DateTimeField
                  label={t("pages.resourceDetails.availability.form.startAt")}
                  value={availabilityStartAt}
                  required
                  onApply={(value) => {
                    setAvailabilityStartAt(value);

                    if (availabilityEndAt && value && availabilityEndAt < value) {
                      setAvailabilityEndAt(value);
                    }
                  }}
                />

                <DateTimeField
                  label={t("pages.resourceDetails.availability.form.endAt")}
                  value={availabilityEndAt}
                  minValue={availabilityStartAt || undefined}
                  required
                  onApply={setAvailabilityEndAt}
                />

                <label className="field resource-availability-form__full">
                  <span>{t("pages.resourceDetails.availability.form.reason")}</span>
                  <textarea
                    value={availabilityReason}
                    onChange={(event) => setAvailabilityReason(event.target.value)}
                    rows={3}
                    placeholder={t("pages.resourceDetails.availability.form.reasonPlaceholder")}
                  />
                </label>

                {availabilityFormError ? <p className="error-text resource-availability-form__full">{availabilityFormError}</p> : null}

                <div className="resource-availability-form__actions resource-availability-form__full">
                  <button type="submit" className="btn btn-primary" disabled={isAvailabilitySubmitting}>
                    {isAvailabilitySubmitting
                      ? t("pages.resourceDetails.availability.form.submitting")
                      : availabilityFormMode === "create"
                        ? t("pages.resourceDetails.availability.form.submitCreate")
                        : t("pages.resourceDetails.availability.form.submitEdit")}
                  </button>
                  <button type="button" className="btn btn-secondary" onClick={closeAvailabilityForm} disabled={isAvailabilitySubmitting}>
                    {t("pages.resourceDetails.availability.actions.cancel")}
                  </button>
                </div>
              </form>
            ) : null}

            {availabilityActionError ? <p className="error-text">{availabilityActionError}</p> : null}

            {displayedAvailability.length === 0 ? (
              <p className="muted resource-details-hint">
                {isAdmin && hasAdditionalRestrictions
                  ? t("pages.resourceDetails.availability.noFuture.description")
                  : t("pages.resourceDetails.availability.unrestricted")}
              </p>
            ) : (
              <div className="availability-list" role="list">
                {displayedAvailability.map((slot) => (
                  <article key={slot.id} className="availability-card" role="listitem">
                    <div className="availability-card__time">
                      <div>
                        <strong>{t("pages.resourceDetails.availability.from")}</strong>
                        <div>{formatUtcDateTime(slot.start_at)}</div>
                      </div>
                      <div>
                        <strong>{t("pages.resourceDetails.availability.to")}</strong>
                        <div>{formatUtcDateTime(slot.end_at)}</div>
                      </div>
                    </div>
                    {slot.reason ? (
                      <div className="availability-card__meta">
                        <strong>{t("pages.resourceDetails.availability.reason")}</strong>
                        <div>{slot.reason}</div>
                      </div>
                    ) : null}
                    {isAdmin ? (
                      <>
                        <div className="availability-card__meta">
                          <strong>{t("pages.resourceDetails.availability.createdAt")}</strong>
                          <div>{formatUtcDateTime(slot.created_at)}</div>
                        </div>
                        <div className="availability-card__meta">
                          <strong>{t("pages.resourceDetails.availability.updatedAt")}</strong>
                          <div>{formatUtcDateTime(slot.updated_at)}</div>
                        </div>
                        <div className="availability-card__actions">
                          <button
                            type="button"
                            className="btn btn-secondary"
                            onClick={() => openAvailabilityEditForm(slot)}
                            disabled={pendingAvailabilityId === slot.id}
                          >
                            {t("pages.resourceDetails.availability.actions.edit")}
                          </button>
                          <button
                            type="button"
                            className="btn btn-secondary"
                            onClick={() => void handleAvailabilityDelete(slot)}
                            disabled={pendingAvailabilityId === slot.id}
                          >
                            {pendingAvailabilityId === slot.id
                              ? t("pages.resourceDetails.availability.actions.deleting")
                              : t("pages.resourceDetails.availability.actions.delete")}
                          </button>
                        </div>
                      </>
                    ) : null}
                  </article>
                ))}
              </div>
            )}
          </div>

          <div className="resource-details-card">
            <div className="resource-details-card__header">
              <div className="resource-details-card__heading">
                <h3 className="resource-details-card__title">{t("pages.resourceDetails.busy.title")}</h3>
              </div>
            </div>
            <div className="resource-day-calendar__controls">
              <div className="resource-day-calendar__quick-actions" role="group" aria-label={t("pages.resourceDetails.busy.quickActions")}>
                <button
                  type="button"
                  className={`bookings-tab ${selectedDate === getLocalDateKey(new Date()) ? "active" : ""}`}
                  onClick={() => handleQuickDateSelect(0)}
                >
                  {t("pages.resourceDetails.busy.today")}
                </button>
                <button
                  type="button"
                  className={`bookings-tab ${selectedDate === getLocalDateKey(addMinutes(new Date(), 24 * 60)) ? "active" : ""}`}
                  onClick={() => handleQuickDateSelect(1)}
                >
                  {t("pages.resourceDetails.busy.tomorrow")}
                </button>
                <button
                  type="button"
                  className={`bookings-tab ${selectedDate === getLocalDateKey(addMinutes(new Date(), 48 * 60)) ? "active" : ""}`}
                  onClick={() => handleQuickDateSelect(2)}
                >
                  {t("pages.resourceDetails.busy.dayAfterTomorrow")}
                </button>
              </div>

              <label className="field resource-day-calendar__date-field">
                <span>{t("pages.resourceDetails.busy.selectedDate")}</span>
                <DatePicker
                  value={selectedDate}
                  onChange={(value) => {
                    if (value) {
                      setSelectedDate(value);
                    }
                  }}
                  ariaLabel={t("pages.resourceDetails.busy.selectedDate")}
                />
              </label>
            </div>

            <p className="muted resource-details-hint">
              {t("pages.resourceDetails.busy.hint", { date: selectedDayLabel })}
            </p>

            <div className="resource-day-calendar__legend" aria-label={t("pages.resourceDetails.busy.legend")}>
              <span className="resource-day-calendar__legend-item">
                <span className="resource-day-calendar__legend-dot is-free" aria-hidden="true" />
                {t("pages.resourceDetails.busy.states.free")}
              </span>
              <span className="resource-day-calendar__legend-item">
                <span className="resource-day-calendar__legend-dot is-selected" aria-hidden="true" />
                {t("pages.resourceDetails.busy.states.selected")}
              </span>
              <span className="resource-day-calendar__legend-item">
                <span className="resource-day-calendar__legend-dot is-busy" aria-hidden="true" />
                {t("pages.resourceDetails.busy.states.busy")}
              </span>
              <span className="resource-day-calendar__legend-item">
                <span className="resource-day-calendar__legend-dot is-unavailable" aria-hidden="true" />
                {t("pages.resourceDetails.busy.states.unavailable")}
              </span>
              <span className="resource-day-calendar__legend-item">
                <span className="resource-day-calendar__legend-dot is-past" aria-hidden="true" />
                {t("pages.resourceDetails.busy.states.past")}
              </span>
            </div>

            {busyLoading ? (
              <LoadingState message={t("pages.resourceDetails.busy.loading")} />
            ) : busyError ? (
              <ErrorState message={busyError} onRetry={() => void loadBusyIntervalsForSelectedDate()} />
            ) : (
              <div className="resource-day-calendar" role="grid" aria-label={t("pages.resourceDetails.busy.calendarLabel", { date: selectedDayLabel })}>
                {daySlots.map((slot) => (
                  <button
                    key={slot.key}
                    type="button"
                    role="gridcell"
                    className={`resource-day-calendar__slot is-${slot.state}`}
                    onClick={() => handleSlotSelect(slot)}
                    disabled={slot.disabled || bookingDisabled || isSubmitting}
                    aria-label={t("pages.resourceDetails.busy.slotLabel", { time: slot.label, state: t(`pages.resourceDetails.busy.states.${slot.state}`) })}
                  >
                    {slot.label}
                  </button>
                ))}
              </div>
            )}

          </div>

          <div className="resource-details-card">
            <h3 className="resource-details-card__title">{t("pages.resourceDetails.booking.title")}</h3>
            {!bookingFormAvailable ? (
              !activeBookingRule ? (
                <p className="muted resource-details-hint">{t("pages.resourceDetails.booking.disabledNoRule")}</p>
              ) : !resource.is_active || !resource.is_bookable ? (
                <p className="muted resource-details-hint">{t("pages.resourceDetails.booking.errors.resourceUnavailable")}</p>
              ) : null
            ) : (
              <>
                <p className="muted resource-details-hint">
                  {activeBookingRule.unrestricted_time
                    ? t("pages.resourceDetails.booking.unrestrictedHint")
                    : t("pages.resourceDetails.rule.values.workdayValue", {
                        start: activeBookingRule.workday_start,
                        end: activeBookingRule.workday_end,
                      })}
                </p>
                <div className="bookings-tabs" role="tablist" aria-label={t("pages.resourceDetails.booking.mode.label")}>
                  <button
                    type="button"
                    className={`bookings-tab ${bookingMode === "single" ? "active" : ""}`}
                    onClick={() => setBookingMode("single")}
                  >
                    {t("pages.resourceDetails.booking.mode.single")}
                  </button>
                  <button
                    type="button"
                    className={`bookings-tab ${bookingMode === "multiple" ? "active" : ""}`}
                    onClick={() => setBookingMode("multiple")}
                  >
                    {t("pages.resourceDetails.booking.mode.multiple")}
                  </button>
                </div>

                {bookingMode === "single" ? (
                  <form className="form-grid" onSubmit={handleSubmit}>
                    <DateTimeField
                      label={t("pages.resourceDetails.booking.fields.startAt")}
                      value={startAt}
                      minValue={startAtMin}
                      required
                      disabled={bookingDisabled || isSubmitting}
                      onApply={(value) => {
                        setStartAt(value);
                        if (value) {
                          setSelectedDate(value.slice(0, 10));
                        }

                        if (endAt && value && endAt < value) {
                          setEndAt(value);
                        }
                      }}
                    />

                    <DateTimeField
                      label={t("pages.resourceDetails.booking.fields.endAt")}
                      value={endAt}
                      minValue={endAtMin}
                      required
                      disabled={bookingDisabled || isSubmitting}
                      onApply={setEndAt}
                    />

                    <label className="field">
                      <span>{t("pages.resourceDetails.booking.fields.purpose")}</span>
                      <textarea
                        value={purpose}
                        onChange={(event) => setPurpose(event.target.value)}
                        rows={4}
                        disabled={bookingDisabled || isSubmitting}
                        placeholder={t("pages.resourceDetails.booking.fields.purposePlaceholder")}
                      />
                    </label>

                    {formError ? <p className="error-text">{formError}</p> : null}

                    <button type="submit" className="btn btn-primary" disabled={isSubmitting || bookingDisabled}>
                      {isSubmitting ? t("pages.resourceDetails.booking.submitting") : t("pages.resourceDetails.booking.submit")}
                    </button>
                  </form>
                ) : (
                  <form className="form-grid resource-batch-form" onSubmit={handleBatchCreate}>
                    <div className="resource-batch-form__full">
                      <div className="resource-batch-form__section">
                        <span className="resource-batch-form__label">{t("pages.resourceDetails.booking.multiple.quickRanges")}</span>
                        <div className="resource-day-calendar__quick-actions">
                          {[3, 5, 7, 14].map((days) => (
                            <button
                              key={days}
                              type="button"
                              className={`bookings-tab ${selectedBatchPresetDays === days ? "active" : ""}`}
                              aria-pressed={selectedBatchPresetDays === days}
                              onClick={() => handleBatchQuickRange(days)}
                            >
                              {t("pages.resourceDetails.booking.multiple.rangeDays", { count: days })}
                            </button>
                          ))}
                          <div className="resource-batch-form__picker">
                            <MultiDatePicker
                              values={batchSelectedDates}
                              onToggleDate={toggleBatchDate}
                              minValue={getLocalDateKey(getCurrentLocalMinute())}
                              disabled={bookingDisabled || isBatchPreviewLoading || isBatchSubmitting}
                              ariaLabel={t("pages.resourceDetails.booking.multiple.pickDates")}
                              triggerLabel={t("pages.resourceDetails.booking.multiple.pickDates")}
                            />
                          </div>
                        </div>
                      </div>
                    </div>

                    <div className="resource-batch-form__full">
                      <div className="resource-batch-selected-list">
                        {batchSelectedDates.map((dateKey) => (
                          <button
                            key={dateKey}
                            type="button"
                            className="resource-batch-chip"
                            onClick={() => removeBatchDate(dateKey)}
                          >
                            <span>{formatDisplayDate(parseLocalDateKey(dateKey))}</span>
                            <span aria-hidden="true">×</span>
                          </button>
                        ))}
                      </div>
                    </div>

                    <label className="field">
                      <span>{t("pages.resourceDetails.booking.fields.startAt")}</span>
                      <TimePicker
                        value={batchStartTime}
                        onChange={setBatchStartTime}
                        disabled={bookingDisabled || isBatchPreviewLoading || isBatchSubmitting}
                        minuteStep={5}
                        ariaLabel={t("pages.resourceDetails.booking.fields.startAt")}
                      />
                    </label>

                    <label className="field">
                      <span>{t("pages.resourceDetails.booking.fields.endAt")}</span>
                      <TimePicker
                        value={batchEndTime}
                        onChange={setBatchEndTime}
                        disabled={bookingDisabled || isBatchPreviewLoading || isBatchSubmitting}
                        minuteStep={5}
                        ariaLabel={t("pages.resourceDetails.booking.fields.endAt")}
                      />
                    </label>

                    <label className="field resource-batch-form__full">
                      <span>{t("pages.resourceDetails.booking.fields.purpose")}</span>
                      <textarea
                        value={purpose}
                        onChange={(event) => setPurpose(event.target.value)}
                        rows={4}
                        disabled={bookingDisabled || isBatchPreviewLoading || isBatchSubmitting}
                        placeholder={t("pages.resourceDetails.booking.fields.purposePlaceholder")}
                      />
                    </label>

                    <div className="resource-batch-form__actions resource-batch-form__full">
                      <button
                        type="button"
                        className="btn btn-secondary"
                        onClick={() => void handleBatchPreview()}
                        disabled={bookingDisabled || isBatchPreviewLoading || isBatchSubmitting}
                      >
                        {isBatchPreviewLoading
                          ? t("pages.resourceDetails.booking.multiple.preview.loading")
                          : t("pages.resourceDetails.booking.multiple.preview.action")}
                      </button>
                      <button type="submit" className="btn btn-primary" disabled={!canSubmitBatch || isBatchSubmitting || bookingDisabled}>
                        {isBatchSubmitting
                          ? t("pages.resourceDetails.booking.multiple.submitLoading")
                          : t("pages.resourceDetails.booking.multiple.submit")}
                      </button>
                    </div>

                    {batchFormError ? <p className="error-text resource-batch-form__full">{batchFormError}</p> : null}

                    {batchPreview ? (
                      <div className="resource-batch-preview resource-batch-form__full">
                        <h4 className="resource-batch-preview__title">{t("pages.resourceDetails.booking.multiple.preview.title")}</h4>
                        {isBatchPreviewLoading ? (
                          <LoadingState message={t("pages.resourceDetails.booking.multiple.preview.loading")} />
                        ) : batchPreview.items.length === 0 ? (
                          <p className="muted resource-details-hint">{t("pages.resourceDetails.booking.multiple.preview.empty")}</p>
                        ) : (
                          <div className="resource-batch-preview__list" role="list">
                            {batchPreview.items.map((item) => (
                              <article key={item.date} className="resource-batch-preview__item" role="listitem">
                                <div>
                                  <strong>{formatDisplayDate(parseLocalDateKey(item.date))}</strong>
                                  <div className="muted">{`${batchStartTime}-${batchEndTime}`}</div>
                                </div>
                                <div className={`badge ${item.valid ? "badge-success" : "badge-warning"}`}>
                                  {item.valid
                                    ? t("pages.resourceDetails.booking.multiple.preview.available")
                                    : mapBatchPreviewCode(item.error_code)}
                                </div>
                                <button type="button" className="btn btn-secondary" onClick={() => removeBatchDate(item.date)}>
                                  {t("pages.resourceDetails.booking.multiple.preview.remove")}
                                </button>
                              </article>
                            ))}
                          </div>
                        )}
                      </div>
                    ) : null}
                  </form>
                )}
              </>
            )}
          </div>
        </div>
      </div>
    </section>
  );
}

