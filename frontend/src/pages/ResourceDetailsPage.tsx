import { FormEvent, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link, useNavigate, useParams } from "react-router-dom";

import { createBooking } from "../api/bookings";
import { ApiError } from "../api/client";
import {
  getResource,
  listResourceAvailability,
  listResourceCategories,
  listResourceTypes,
} from "../api/resources";
import { EmptyState } from "../components/EmptyState";
import { ErrorState } from "../components/ErrorState";
import { LoadingState } from "../components/LoadingState";
import { PageHeader } from "../components/PageHeader";
import type { Resource, ResourceAvailability, ResourceCategory, ResourceType } from "../types/resources";
import { formatUtcDateTime } from "../utils/datetime";

function mapBookingError(message: string, t: ReturnType<typeof useTranslation>["t"]): string {
  switch (message) {
    case "invalid booking payload":
      return t("pages.resourceDetails.booking.errors.invalidPayload");
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

function toDateTimeLocalValue(value: Date): string {
  const year = value.getFullYear();
  const month = String(value.getMonth() + 1).padStart(2, "0");
  const day = String(value.getDate()).padStart(2, "0");
  const hours = String(value.getHours()).padStart(2, "0");
  const minutes = String(value.getMinutes()).padStart(2, "0");

  return `${year}-${month}-${day}T${hours}:${minutes}`;
}

export function ResourceDetailsPage(): JSX.Element {
  const { id } = useParams();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const resourceId = Number(id);

  const [resource, setResource] = useState<Resource | null>(null);
  const [categories, setCategories] = useState<ResourceCategory[]>([]);
  const [types, setTypes] = useState<ResourceType[]>([]);
  const [availability, setAvailability] = useState<ResourceAvailability[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [startAt, setStartAt] = useState("");
  const [endAt, setEndAt] = useState("");
  const [purpose, setPurpose] = useState("");
  const [formError, setFormError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const startAtMin = toDateTimeLocalValue(new Date());

  useEffect(() => {
    void loadResourceDetails();
  }, [resourceId]);

  async function loadResourceDetails(): Promise<void> {
    if (!Number.isInteger(resourceId) || resourceId <= 0) {
      setError(t("pages.resourceDetails.errors.invalidResource"));
      setLoading(false);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const [resourceData, categoriesData, typesData, availabilityData] = await Promise.all([
        getResource(resourceId),
        listResourceCategories(),
        listResourceTypes(),
        listResourceAvailability(resourceId),
      ]);

      setResource(resourceData);
      setCategories(categoriesData);
      setTypes(typesData);
      setAvailability(availabilityData);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : t("errors.generic"));
    } finally {
      setLoading(false);
    }
  }

  const categoryMap = useMemo(() => new Map(categories.map((category) => [category.id, category.name])), [categories]);
  const typeMap = useMemo(() => new Map(types.map((type) => [type.id, type.name])), [types]);

  const futureAvailability = useMemo(() => {
    const now = Date.now();
    const uniqueSlots = new Map<string, ResourceAvailability>();

    for (const slot of availability) {
      const startTime = new Date(slot.start_at).getTime();
      const endTime = new Date(slot.end_at).getTime();

      if (Number.isNaN(startTime) || Number.isNaN(endTime) || endTime <= now) {
        continue;
      }

      const key = `${slot.start_at}__${slot.end_at}`;
      if (!uniqueSlots.has(key)) {
        uniqueSlots.set(key, slot);
      }
    }

    return [...uniqueSlots.values()].sort(
      (left, right) => new Date(left.start_at).getTime() - new Date(right.start_at).getTime(),
    );
  }, [availability]);

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

    if (startValue <= new Date()) {
      return t("pages.resourceDetails.booking.errors.startInPast");
    }

    if (startValue >= endValue) {
      return t("pages.resourceDetails.booking.errors.invalidRange");
    }

    const isInsideAvailability = futureAvailability.some((slot) => {
      const slotStart = new Date(slot.start_at).getTime();
      const slotEnd = new Date(slot.end_at).getTime();

      return startValue.getTime() >= slotStart && endValue.getTime() <= slotEnd;
    });

    if (!isInsideAvailability) {
      return t("pages.resourceDetails.booking.errors.outsideAvailability");
    }

    return null;
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
        setFormError(mapBookingError(submitError.message, t));
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

  const hasFutureAvailability = futureAvailability.length > 0;

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
          </dl>
        </div>

        <div className="resource-details-grid">
          <div className="resource-details-card">
            <h3 className="resource-details-card__title">{t("pages.resourceDetails.availability.title")}</h3>
            <p className="muted resource-details-hint">{t("pages.resourceDetails.availability.hint")}</p>

            {!hasFutureAvailability ? (
              <EmptyState
                title={t("pages.resourceDetails.availability.noFuture.title")}
                description={t("pages.resourceDetails.availability.noFuture.description")}
              />
            ) : (
              <div className="availability-list" role="list">
                {futureAvailability.map((slot) => (
                  <article key={slot.id} className="availability-card" role="listitem">
                    <div>
                      <strong>{t("pages.resourceDetails.availability.from")}</strong>
                      <div>{formatUtcDateTime(slot.start_at)}</div>
                    </div>
                    <div>
                      <strong>{t("pages.resourceDetails.availability.to")}</strong>
                      <div>{formatUtcDateTime(slot.end_at)}</div>
                    </div>
                  </article>
                ))}
              </div>
            )}
          </div>

          <div className="resource-details-card">
            <h3 className="resource-details-card__title">{t("pages.resourceDetails.booking.title")}</h3>
            {!hasFutureAvailability ? (
              <p className="muted resource-details-hint">{t("pages.resourceDetails.booking.disabledNoAvailability")}</p>
            ) : (
              <p className="muted resource-details-hint">{t("pages.resourceDetails.availability.hint")}</p>
            )}
            <form className="form-grid" onSubmit={handleSubmit}>
              <label className="field">
                <span>{t("pages.resourceDetails.booking.fields.startAt")}</span>
                <input
                  type="datetime-local"
                  value={startAt}
                  min={startAtMin}
                  onChange={(event) => {
                    const nextStartAt = event.target.value;
                    setStartAt(nextStartAt);

                    if (endAt && nextStartAt && endAt < nextStartAt) {
                      setEndAt(nextStartAt);
                    }
                  }}
                  required
                />
              </label>

              <label className="field">
                <span>{t("pages.resourceDetails.booking.fields.endAt")}</span>
                <input
                  type="datetime-local"
                  value={endAt}
                  min={endAtMin}
                  onChange={(event) => setEndAt(event.target.value)}
                  required
                />
              </label>

              <label className="field">
                <span>{t("pages.resourceDetails.booking.fields.purpose")}</span>
                <textarea
                  value={purpose}
                  onChange={(event) => setPurpose(event.target.value)}
                  rows={4}
                  placeholder={t("pages.resourceDetails.booking.fields.purposePlaceholder")}
                />
              </label>

              {formError ? <p className="error-text">{formError}</p> : null}

              <button type="submit" className="btn btn-primary" disabled={isSubmitting || !hasFutureAvailability}>
                {isSubmitting ? t("pages.resourceDetails.booking.submitting") : t("pages.resourceDetails.booking.submit")}
              </button>
            </form>
          </div>
        </div>
      </div>
    </section>
  );
}
