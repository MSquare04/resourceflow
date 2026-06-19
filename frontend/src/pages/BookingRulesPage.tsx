import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

import { createBookingRule, listBookingRules, updateBookingRule } from "../api/bookingRules";
import { ApiError } from "../api/client";
import { listResourceTypes } from "../api/resources";
import { EmptyState } from "../components/EmptyState";
import { ErrorState } from "../components/ErrorState";
import { LoadingState } from "../components/LoadingState";
import { PageHeader } from "../components/PageHeader";
import type { BookingRule, BookingRulePayload } from "../types/bookingRules";
import type { ResourceType } from "../types/resources";
import { formatUtcDateTime } from "../utils/datetime";

type ActiveFilter = "all" | "active" | "inactive";
type ApprovalFilter = "all" | "required" | "notRequired";
type FormMode = "create" | "edit";

interface RuleFilters {
  search: string;
  active: ActiveFilter;
  approval: ApprovalFilter;
}

interface RuleFormState {
  resourceTypeId: string;
  minDurationMinutes: string;
  maxDurationMinutes: string;
  maxActiveBookingsPerUser: string;
  bookingHorizonDays: string;
  requiresApproval: boolean;
  isActive: boolean;
}

const defaultFilters: RuleFilters = {
  search: "",
  active: "all",
  approval: "all",
};

const defaultFormState: RuleFormState = {
  resourceTypeId: "",
  minDurationMinutes: "",
  maxDurationMinutes: "",
  maxActiveBookingsPerUser: "",
  bookingHorizonDays: "",
  requiresApproval: false,
  isActive: true,
};

function formatDuration(minutes: number, t: ReturnType<typeof useTranslation>["t"]): string {
  if (minutes % 60 === 0) {
    return t("pages.bookingRules.duration.hours", { count: minutes / 60 });
  }

  return t("pages.bookingRules.duration.minutes", { count: minutes });
}

function mapRuleError(message: string, t: ReturnType<typeof useTranslation>["t"]): string {
  switch (message) {
    case "invalid booking rule payload":
      return t("pages.bookingRules.form.errors.invalidPayload");
    case "booking rule not found":
      return t("pages.bookingRules.form.errors.ruleNotFound");
    default:
      return message;
  }
}

export function BookingRulesPage(): JSX.Element {
  const { t } = useTranslation();
  const [rules, setRules] = useState<BookingRule[]>([]);
  const [resourceTypes, setResourceTypes] = useState<ResourceType[]>([]);
  const [filters, setFilters] = useState<RuleFilters>(defaultFilters);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isFormOpen, setIsFormOpen] = useState(false);
  const [formMode, setFormMode] = useState<FormMode>("create");
  const [editingRuleId, setEditingRuleId] = useState<number | null>(null);
  const [formState, setFormState] = useState<RuleFormState>(defaultFormState);
  const [formError, setFormError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    void loadBookingRulesPage();
  }, []);

  async function loadBookingRulesPage(): Promise<void> {
    setLoading(true);
    setError(null);

    try {
      const [rulesData, resourceTypesData] = await Promise.all([listBookingRules(), listResourceTypes()]);
      setRules(rulesData);
      setResourceTypes(resourceTypesData);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : t("errors.generic"));
    } finally {
      setLoading(false);
    }
  }

  const resourceTypeMap = useMemo(() => new Map(resourceTypes.map((type) => [type.id, type.name])), [resourceTypes]);

  const filteredRules = useMemo(() => {
    const normalizedSearch = filters.search.trim().toLowerCase();

    return [...rules]
      .filter((rule) => {
        if (!normalizedSearch) {
          return true;
        }

        const typeName = resourceTypeMap.get(rule.resource_type_id) ?? "";
        return typeName.toLowerCase().includes(normalizedSearch);
      })
      .filter((rule) => {
        if (filters.active === "all") {
          return true;
        }

        return filters.active === "active" ? rule.is_active : !rule.is_active;
      })
      .filter((rule) => {
        if (filters.approval === "all") {
          return true;
        }

        return filters.approval === "required" ? rule.requires_approval : !rule.requires_approval;
      })
      .sort((left, right) => {
        const leftTypeName = resourceTypeMap.get(left.resource_type_id) ?? "";
        const rightTypeName = resourceTypeMap.get(right.resource_type_id) ?? "";
        return leftTypeName.localeCompare(rightTypeName);
      });
  }, [filters, resourceTypeMap, rules]);

  const hasActiveFilters =
    filters.search !== defaultFilters.search || filters.active !== defaultFilters.active || filters.approval !== defaultFilters.approval;

  function resetFilters(): void {
    setFilters(defaultFilters);
  }

  function closeForm(): void {
    setIsFormOpen(false);
    setFormMode("create");
    setEditingRuleId(null);
    setFormState(defaultFormState);
    setFormError(null);
    setIsSubmitting(false);
  }

  function openCreateForm(): void {
    setFormMode("create");
    setEditingRuleId(null);
    setFormState(defaultFormState);
    setFormError(null);
    setIsFormOpen(true);
  }

  function openEditForm(rule: BookingRule): void {
    setFormMode("edit");
    setEditingRuleId(rule.id);
    setFormState({
      resourceTypeId: String(rule.resource_type_id),
      minDurationMinutes: String(rule.min_duration_minutes),
      maxDurationMinutes: String(rule.max_duration_minutes),
      maxActiveBookingsPerUser: String(rule.max_active_bookings_per_user),
      bookingHorizonDays: String(rule.booking_horizon_days),
      requiresApproval: rule.requires_approval,
      isActive: rule.is_active,
    });
    setFormError(null);
    setIsFormOpen(true);
  }

  function validateForm(): string | null {
    const resourceTypeId = Number(formState.resourceTypeId);
    const minDuration = Number(formState.minDurationMinutes);
    const maxDuration = Number(formState.maxDurationMinutes);
    const maxActiveBookings = Number(formState.maxActiveBookingsPerUser);
    const bookingHorizon = Number(formState.bookingHorizonDays);

    if (!formState.resourceTypeId) {
      return t("pages.bookingRules.form.errors.resourceTypeRequired");
    }

    if (!Number.isInteger(resourceTypeId) || resourceTypeId <= 0) {
      return t("pages.bookingRules.form.errors.resourceTypeRequired");
    }

    if (formState.minDurationMinutes === "" || Number.isNaN(minDuration)) {
      return t("pages.bookingRules.form.errors.minDurationRequired");
    }

    if (formState.maxDurationMinutes === "" || Number.isNaN(maxDuration)) {
      return t("pages.bookingRules.form.errors.maxDurationRequired");
    }

    if (formState.maxActiveBookingsPerUser === "" || Number.isNaN(maxActiveBookings)) {
      return t("pages.bookingRules.form.errors.maxActiveRequired");
    }

    if (formState.bookingHorizonDays === "" || Number.isNaN(bookingHorizon)) {
      return t("pages.bookingRules.form.errors.horizonRequired");
    }

    if (minDuration <= 0) {
      return t("pages.bookingRules.form.errors.minDurationPositive");
    }

    if (maxDuration < 0 || maxActiveBookings < 0 || bookingHorizon < 0) {
      return t("pages.bookingRules.form.errors.nonNegative");
    }

    if (maxDuration < minDuration) {
      return t("pages.bookingRules.form.errors.invalidDurationRange");
    }

    if (maxActiveBookings < 1) {
      return t("pages.bookingRules.form.errors.maxActivePositive");
    }

    return null;
  }

  async function handleSubmit(): Promise<void> {
    const validationError = validateForm();
    if (validationError) {
      setFormError(validationError);
      return;
    }

    setFormError(null);
    setIsSubmitting(true);

    const payload: BookingRulePayload = {
      resource_type_id: Number(formState.resourceTypeId),
      min_duration_minutes: Number(formState.minDurationMinutes),
      max_duration_minutes: Number(formState.maxDurationMinutes),
      max_active_bookings_per_user: Number(formState.maxActiveBookingsPerUser),
      requires_approval: formState.requiresApproval,
      booking_horizon_days: Number(formState.bookingHorizonDays),
      is_active: formState.isActive,
    };

    try {
      if (formMode === "create") {
        await createBookingRule(payload);
      } else if (editingRuleId !== null) {
        await updateBookingRule(editingRuleId, payload);
      }

      await loadBookingRulesPage();
      closeForm();
    } catch (submitError) {
      if (submitError instanceof ApiError) {
        setFormError(mapRuleError(submitError.message, t));
      } else if (submitError instanceof Error) {
        setFormError(submitError.message);
      } else {
        setFormError(t("pages.bookingRules.form.errors.generic"));
      }
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <section>
      <PageHeader
        title={t("pages.bookingRules.title")}
        actions={
          <button type="button" className="btn btn-primary" onClick={openCreateForm}>
            {t("pages.bookingRules.actions.create")}
          </button>
        }
      />

      {isFormOpen ? (
        <div className="rules-form-panel">
          <div className="rules-form-panel__header">
            <div>
              <h3>{formMode === "create" ? t("pages.bookingRules.form.createTitle") : t("pages.bookingRules.form.editTitle")}</h3>
              <p className="muted">
                {formMode === "create" ? t("pages.bookingRules.form.createHint") : t("pages.bookingRules.form.editHint")}
              </p>
            </div>
            <button type="button" className="btn btn-secondary" onClick={closeForm} disabled={isSubmitting}>
              {t("pages.bookingRules.actions.closeForm")}
            </button>
          </div>

          <div className="rules-form-grid">
            <label className="field">
              <span>{t("pages.bookingRules.form.fields.resourceType")}</span>
              <select
                value={formState.resourceTypeId}
                onChange={(event) => setFormState((current) => ({ ...current, resourceTypeId: event.target.value }))}
              >
                <option value="">{t("pages.bookingRules.form.fields.selectResourceType")}</option>
                {[...resourceTypes]
                  .sort((left, right) => left.name.localeCompare(right.name))
                  .map((resourceType) => (
                    <option key={resourceType.id} value={String(resourceType.id)}>
                      {resourceType.name}
                    </option>
                  ))}
              </select>
            </label>

            <label className="field">
              <span>{t("pages.bookingRules.form.fields.minDurationMinutes")}</span>
              <input
                type="number"
                min="1"
                value={formState.minDurationMinutes}
                onChange={(event) => setFormState((current) => ({ ...current, minDurationMinutes: event.target.value }))}
              />
            </label>

            <label className="field">
              <span>{t("pages.bookingRules.form.fields.maxDurationMinutes")}</span>
              <input
                type="number"
                min="0"
                value={formState.maxDurationMinutes}
                onChange={(event) => setFormState((current) => ({ ...current, maxDurationMinutes: event.target.value }))}
              />
            </label>

            <label className="field">
              <span>{t("pages.bookingRules.form.fields.maxActiveBookingsPerUser")}</span>
              <input
                type="number"
                min="1"
                value={formState.maxActiveBookingsPerUser}
                onChange={(event) =>
                  setFormState((current) => ({ ...current, maxActiveBookingsPerUser: event.target.value }))
                }
              />
            </label>

            <label className="field">
              <span>{t("pages.bookingRules.form.fields.bookingHorizonDays")}</span>
              <input
                type="number"
                min="0"
                value={formState.bookingHorizonDays}
                onChange={(event) => setFormState((current) => ({ ...current, bookingHorizonDays: event.target.value }))}
              />
            </label>

            <label className="field field-checkbox">
              <span>{t("pages.bookingRules.form.fields.requiresApproval")}</span>
              <input
                type="checkbox"
                checked={formState.requiresApproval}
                onChange={(event) => setFormState((current) => ({ ...current, requiresApproval: event.target.checked }))}
              />
            </label>

            <label className="field field-checkbox">
              <span>{t("pages.bookingRules.form.fields.isActive")}</span>
              <input
                type="checkbox"
                checked={formState.isActive}
                onChange={(event) => setFormState((current) => ({ ...current, isActive: event.target.checked }))}
              />
            </label>
          </div>

          {formError ? <p className="error-text">{formError}</p> : null}

          <div className="rules-form-actions">
            <button type="button" className="btn btn-primary" onClick={() => void handleSubmit()} disabled={isSubmitting}>
              {isSubmitting ? t("pages.bookingRules.form.submitting") : t("pages.bookingRules.form.submit")}
            </button>
          </div>
        </div>
      ) : null}

      {loading ? <LoadingState message={t("pages.bookingRules.loading")} /> : null}
      {!loading && error ? <ErrorState message={error} onRetry={() => void loadBookingRulesPage()} /> : null}
      {!loading && !error && rules.length === 0 ? (
        <EmptyState
          title={t("pages.bookingRules.empty.title")}
          description={t("pages.bookingRules.empty.description")}
        />
      ) : null}

      {!loading && !error && rules.length > 0 ? (
        <div className="rules-page">
          <div className="rules-filters">
            <label className="field">
              <span>{t("pages.bookingRules.filters.search")}</span>
              <input
                type="search"
                value={filters.search}
                onChange={(event) => setFilters((current) => ({ ...current, search: event.target.value }))}
                placeholder={t("pages.bookingRules.filters.searchPlaceholder")}
              />
            </label>

            <label className="field">
              <span>{t("pages.bookingRules.filters.activity")}</span>
              <select
                value={filters.active}
                onChange={(event) => setFilters((current) => ({ ...current, active: event.target.value as ActiveFilter }))}
              >
                <option value="all">{t("pages.bookingRules.filters.allActivity")}</option>
                <option value="active">{t("pages.bookingRules.filters.active")}</option>
                <option value="inactive">{t("pages.bookingRules.filters.inactive")}</option>
              </select>
            </label>

            <label className="field">
              <span>{t("pages.bookingRules.filters.approval")}</span>
              <select
                value={filters.approval}
                onChange={(event) =>
                  setFilters((current) => ({ ...current, approval: event.target.value as ApprovalFilter }))
                }
              >
                <option value="all">{t("pages.bookingRules.filters.allApproval")}</option>
                <option value="required">{t("pages.bookingRules.filters.approvalRequired")}</option>
                <option value="notRequired">{t("pages.bookingRules.filters.approvalNotRequired")}</option>
              </select>
            </label>

            <button type="button" className="btn btn-secondary" onClick={resetFilters} disabled={!hasActiveFilters}>
              {t("pages.bookingRules.filters.reset")}
            </button>
          </div>

          <p className="resources-summary muted">{t("pages.bookingRules.resultsSummary", { count: filteredRules.length })}</p>

          {filteredRules.length === 0 ? (
            <EmptyState
              title={t("pages.bookingRules.noResults.title")}
              description={t("pages.bookingRules.noResults.description")}
            />
          ) : (
            <div className="rules-list" role="list">
              {filteredRules.map((rule) => (
                <article key={rule.id} className="rule-card" role="listitem">
                  <div className="rule-card__header">
                    <div className="rule-card__heading">
                      <h3 className="rule-card__title">
                        {resourceTypeMap.get(rule.resource_type_id) ?? t("pages.bookingRules.unknownResourceType")}
                      </h3>
                      <p className="rule-card__subtitle">
                        {t("pages.bookingRules.fields.resourceTypeId")}: {rule.resource_type_id}
                      </p>
                    </div>
                    <div className="resource-card__badges">
                      <span className={`badge ${rule.is_active ? "badge-success" : "badge-muted"}`}>
                        {rule.is_active ? t("pages.bookingRules.status.active") : t("pages.bookingRules.status.inactive")}
                      </span>
                      <span className={`badge ${rule.requires_approval ? "badge-warning" : "badge-info"}`}>
                        {rule.requires_approval
                          ? t("pages.bookingRules.status.approvalRequired")
                          : t("pages.bookingRules.status.approvalNotRequired")}
                      </span>
                    </div>
                  </div>

                  <dl className="rule-card__meta">
                    <div>
                      <dt>{t("pages.bookingRules.fields.minDuration")}</dt>
                      <dd>{formatDuration(rule.min_duration_minutes, t)}</dd>
                    </div>
                    <div>
                      <dt>{t("pages.bookingRules.fields.maxDuration")}</dt>
                      <dd>{formatDuration(rule.max_duration_minutes, t)}</dd>
                    </div>
                    <div>
                      <dt>{t("pages.bookingRules.fields.maxActiveBookingsPerUser")}</dt>
                      <dd>{rule.max_active_bookings_per_user}</dd>
                    </div>
                    <div>
                      <dt>{t("pages.bookingRules.fields.bookingHorizonDays")}</dt>
                      <dd>{t("pages.bookingRules.horizon.days", { count: rule.booking_horizon_days })}</dd>
                    </div>
                    <div>
                      <dt>{t("pages.bookingRules.fields.createdAt")}</dt>
                      <dd>{formatUtcDateTime(rule.created_at)}</dd>
                    </div>
                    <div>
                      <dt>{t("pages.bookingRules.fields.updatedAt")}</dt>
                      <dd>{formatUtcDateTime(rule.updated_at)}</dd>
                    </div>
                  </dl>

                  <div className="rule-card__actions">
                    <button type="button" className="btn btn-secondary" onClick={() => openEditForm(rule)}>
                      {t("pages.bookingRules.actions.edit")}
                    </button>
                  </div>
                </article>
              ))}
            </div>
          )}
        </div>
      ) : null}
    </section>
  );
}
