import { FormEvent, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link, useNavigate, useParams } from "react-router-dom";

import { createBooking } from "../api/bookings";
import { listBookingRules } from "../api/bookingRules";
import { ApiError } from "../api/client";
import { listDepartments } from "../api/departments";
import {
  createResourceAvailability,
  deleteResourceAvailability,
  getResource,
  listResourceAvailability,
  listResourceBusyIntervals,
  listResourceCategories,
  listResourceTypes,
  updateResourceAvailability,
} from "../api/resources";
import { useRoles } from "../auth/useRoles";
import { DateTimeField } from "../components/DateTimeField";
import { EmptyState } from "../components/EmptyState";
import { ErrorState } from "../components/ErrorState";
import { LoadingState } from "../components/LoadingState";
import { PageHeader } from "../components/PageHeader";
import type { BookingRule } from "../types/bookingRules";
import type { Resource, ResourceAvailability, ResourceBusyInterval, ResourceCategory, ResourceType } from "../types/resources";
import type { Department } from "../types/users";
import { formatUtcDateTime } from "../utils/datetime";

type AvailabilityFormMode = "create" | "edit";

function mapBookingError(error: ApiError, t: ReturnType<typeof useTranslation>["t"]): string {
  if (error.code === "conflict" || error.status === 409) {
    if (error.message === "resource is inactive or not bookable") {
      return t("pages.resourceDetails.booking.errors.resourceUnavailable");
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
    case "booking interval is outside resource availability":
      return t("pages.resourceDetails.booking.errors.outsideAvailability");
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
    case "invalid availability payload":
      return t("pages.resourceDetails.availability.errors.invalidPayload");
    case "resource not found":
      return t("pages.resourceDetails.availability.errors.resourceNotFound");
    case "resource availability not found":
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
  const [startAt, setStartAt] = useState("");
  const [endAt, setEndAt] = useState("");
  const [purpose, setPurpose] = useState("");
  const [formError, setFormError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isAvailabilityFormOpen, setIsAvailabilityFormOpen] = useState(false);
  const [availabilityFormMode, setAvailabilityFormMode] = useState<AvailabilityFormMode>("create");
  const [editingAvailabilityId, setEditingAvailabilityId] = useState<number | null>(null);
  const [availabilityStartAt, setAvailabilityStartAt] = useState("");
  const [availabilityEndAt, setAvailabilityEndAt] = useState("");
  const [availabilityFormError, setAvailabilityFormError] = useState<string | null>(null);
  const [availabilityActionError, setAvailabilityActionError] = useState<string | null>(null);
  const [isAvailabilitySubmitting, setIsAvailabilitySubmitting] = useState(false);
  const [pendingAvailabilityId, setPendingAvailabilityId] = useState<number | null>(null);
  const availabilityFormRef = useRef<HTMLFormElement | null>(null);
  const bookingRuleActionButtonRef = useRef<HTMLButtonElement | null>(null);
  const startAtMin = toDateTimeLocalValue(getCurrentLocalMinute());

  useEffect(() => {
    void loadResourceDetails();
  }, [isAdmin, resourceId]);

  async function loadResourceDetails(): Promise<void> {
    if (!Number.isInteger(resourceId) || resourceId <= 0) {
      setError(t("pages.resourceDetails.errors.invalidResource"));
      setLoading(false);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const [resourceData, categoriesData, typesData, availabilityData, busyIntervalsData, bookingRulesData] = await Promise.all([
        getResource(resourceId),
        listResourceCategories(),
        listResourceTypes(),
        listResourceAvailability(resourceId),
        listResourceBusyIntervals(resourceId),
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
      setBusyIntervals(busyIntervalsData);
      setBookingRules(bookingRulesData);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : t("errors.generic"));
    } finally {
      setLoading(false);
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
  const hasFutureRestrictedWindow = futureAvailability.length > 0;
  const bookingDisabled = !activeBookingRule || (hasAdditionalRestrictions && !hasFutureRestrictedWindow);

  const endAtMin = useMemo(() => {
    if (!startAt) {
      return startAtMin;
    }

    return startAt > startAtMin ? startAt : startAtMin;
  }, [startAt, startAtMin]);

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

    if (hasAdditionalRestrictions) {
      const isInsideAvailability = futureAvailability.some((slot) => {
        const slotStart = new Date(slot.start_at).getTime();
        const slotEnd = new Date(slot.end_at).getTime();

        return startValue.getTime() >= slotStart && endValue.getTime() <= slotEnd;
      });

      if (!isInsideAvailability) {
        return t("pages.resourceDetails.booking.errors.outsideAvailability");
      }
    }

    const intersectsBusyInterval = visibleBusyIntervals.some((interval) => {
      const intervalStart = new Date(interval.start_at).getTime();
      const intervalEnd = new Date(interval.end_at).getTime();

      return startValue.getTime() < intervalEnd && endValue.getTime() > intervalStart;
    });

    if (intersectsBusyInterval) {
      return t("pages.resourceDetails.booking.errors.busyConflict");
    }

    return null;
  }

  function resetAvailabilityForm(): void {
    setAvailabilityFormMode("create");
    setEditingAvailabilityId(null);
    setAvailabilityStartAt("");
    setAvailabilityEndAt("");
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
                {hasAdditionalRestrictions
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
            <p className="muted resource-details-hint">{t("pages.resourceDetails.busy.hint")}</p>

            {visibleBusyIntervals.length === 0 ? (
              <EmptyState title={t("pages.resourceDetails.busy.empty.title")} />
            ) : (
              <div className="busy-intervals-list" role="list">
                {visibleBusyIntervals.map((interval) => (
                  <article key={`${interval.start_at}-${interval.end_at}`} className="busy-interval-card" role="listitem">
                    <div>
                      <strong>{t("pages.resourceDetails.busy.from")}</strong>
                      <div>{formatUtcDateTime(interval.start_at)}</div>
                    </div>
                    <div>
                      <strong>{t("pages.resourceDetails.busy.to")}</strong>
                      <div>{formatUtcDateTime(interval.end_at)}</div>
                    </div>
                  </article>
                ))}
              </div>
            )}
          </div>

          <div className="resource-details-card">
            <h3 className="resource-details-card__title">{t("pages.resourceDetails.booking.title")}</h3>
            {!activeBookingRule ? (
              <p className="muted resource-details-hint">{t("pages.resourceDetails.booking.disabledNoRule")}</p>
            ) : bookingDisabled ? (
              <p className="muted resource-details-hint">{t("pages.resourceDetails.booking.disabledNoAvailability")}</p>
            ) : (
              <p className="muted resource-details-hint">
                {hasAdditionalRestrictions
                  ? t("pages.resourceDetails.availability.hint")
                  : t("pages.resourceDetails.booking.unrestrictedHint")}
              </p>
            )}
            <form className="form-grid" onSubmit={handleSubmit}>
              <DateTimeField
                label={t("pages.resourceDetails.booking.fields.startAt")}
                value={startAt}
                minValue={startAtMin}
                required
                disabled={bookingDisabled || isSubmitting}
                onApply={(value) => {
                  setStartAt(value);

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
          </div>
        </div>
      </div>
    </section>
  );
}
