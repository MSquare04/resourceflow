import { FormEvent, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";

import { ApiError } from "../api/client";
import { listDepartments } from "../api/departments";
import {
  createResource,
  listResourceCategories,
  listResources,
  listResourceTypes,
  updateResource,
} from "../api/resources";
import { useRoles } from "../auth/useRoles";
import { ToggleSwitch } from "../components/ToggleSwitch";
import { EmptyState } from "../components/EmptyState";
import { ErrorState } from "../components/ErrorState";
import { LoadingState } from "../components/LoadingState";
import { PageHeader } from "../components/PageHeader";
import type { Resource, ResourceCategory, ResourcePayload, ResourceType } from "../types/resources";
import type { Department } from "../types/users";

type BooleanFilter = "all" | "true" | "false";
type ResourceFormMode = "create" | "edit";

interface ResourceFilters {
  search: string;
  categoryId: string;
  typeId: string;
  isBookable: BooleanFilter;
  isActive: BooleanFilter;
}

interface ResourceFormState {
  name: string;
  description: string;
  categoryId: string;
  typeId: string;
  departmentId: string;
  location: string;
  capacity: string;
  isBookable: boolean;
  isActive: boolean;
}

const defaultFilters: ResourceFilters = {
  search: "",
  categoryId: "all",
  typeId: "all",
  isBookable: "all",
  isActive: "all",
};

const defaultFormState: ResourceFormState = {
  name: "",
  description: "",
  categoryId: "",
  typeId: "",
  departmentId: "",
  location: "",
  capacity: "",
  isBookable: true,
  isActive: true,
};

function mapResourceError(message: string, t: ReturnType<typeof useTranslation>["t"]): string {
  switch (message) {
    case "invalid resource payload":
      return t("pages.resources.form.errors.invalidPayload");
    case "resource not found":
      return t("pages.resources.form.errors.resourceNotFound");
    default:
      return message;
  }
}

export function ResourcesPage(): JSX.Element {
  const { t } = useTranslation();
  const { hasRole } = useRoles();
  const isAdmin = hasRole("admin");
  const [resources, setResources] = useState<Resource[]>([]);
  const [categories, setCategories] = useState<ResourceCategory[]>([]);
  const [types, setTypes] = useState<ResourceType[]>([]);
  const [departments, setDepartments] = useState<Department[]>([]);
  const [filters, setFilters] = useState<ResourceFilters>(defaultFilters);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isFormOpen, setIsFormOpen] = useState(false);
  const [formMode, setFormMode] = useState<ResourceFormMode>("create");
  const [editingResourceId, setEditingResourceId] = useState<number | null>(null);
  const [formState, setFormState] = useState<ResourceFormState>(defaultFormState);
  const [formError, setFormError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const formPanelRef = useRef<HTMLDivElement | null>(null);
  const nameInputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    void loadResourcesCatalog();
  }, [isAdmin]);

  useEffect(() => {
    if (!isAdmin || !isFormOpen) {
      return;
    }

    const frameId = window.requestAnimationFrame(() => {
      formPanelRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
      window.setTimeout(() => {
        nameInputRef.current?.focus({ preventScroll: true });
      }, 150);
    });

    return () => {
      window.cancelAnimationFrame(frameId);
    };
  }, [isAdmin, isFormOpen, formMode, editingResourceId]);

  async function loadResourcesCatalog(): Promise<{
    resources: Resource[];
    categories: ResourceCategory[];
    types: ResourceType[];
    departments: Department[];
  } | null> {
    setLoading(true);
    setError(null);

    try {
      const [resourcesData, categoriesData, typesData] = await Promise.all([
        listResources(),
        listResourceCategories(),
        listResourceTypes(),
      ]);
      let departmentsData: Department[] = [];

      if (isAdmin) {
        try {
          departmentsData = await listDepartments();
        } catch {
          departmentsData = [];
        }
      }

      setResources(resourcesData);
      setCategories(categoriesData);
      setTypes(typesData);
      setDepartments(departmentsData);

      return {
        resources: resourcesData,
        categories: categoriesData,
        types: typesData,
        departments: departmentsData,
      };
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : t("errors.generic"));
      return null;
    } finally {
      setLoading(false);
    }
  }

  const sortedCategories = useMemo(
    () => [...categories].sort((left, right) => left.name.localeCompare(right.name)),
    [categories],
  );
  const sortedTypes = useMemo(() => [...types].sort((left, right) => left.name.localeCompare(right.name)), [types]);
  const sortedDepartments = useMemo(
    () => [...departments].sort((left, right) => left.name.localeCompare(right.name)),
    [departments],
  );

  const categoryMap = useMemo(() => new Map(categories.map((category) => [category.id, category])), [categories]);
  const typeMap = useMemo(() => new Map(types.map((type) => [type.id, type])), [types]);
  const departmentMap = useMemo(
    () => new Map(departments.map((department) => [department.id, department])),
    [departments],
  );

  const typeOptions = useMemo(() => {
    if (filters.categoryId === "all") {
      return sortedTypes;
    }

    return sortedTypes.filter((type) => String(type.category_id) === filters.categoryId);
  }, [filters.categoryId, sortedTypes]);

  const formTypeOptions = useMemo(() => {
    if (!formState.categoryId) {
      return sortedTypes;
    }

    return sortedTypes.filter((type) => String(type.category_id) === formState.categoryId);
  }, [formState.categoryId, sortedTypes]);

  const filteredResources = useMemo(() => {
    const normalizedSearch = filters.search.trim().toLowerCase();

    return [...resources]
      .filter((resource) => {
        if (!normalizedSearch) {
          return true;
        }

        const categoryName = categoryMap.get(resource.category_id)?.name ?? "";
        const typeName = typeMap.get(resource.type_id)?.name ?? "";
        const searchableFields = [
          resource.name,
          resource.description,
          resource.location ?? "",
          categoryName,
          typeName,
          resource.capacity !== null ? String(resource.capacity) : "",
        ];

        return searchableFields.some((field) => field.toLowerCase().includes(normalizedSearch));
      })
      .filter((resource) => (filters.categoryId === "all" ? true : String(resource.category_id) === filters.categoryId))
      .filter((resource) => (filters.typeId === "all" ? true : String(resource.type_id) === filters.typeId))
      .filter((resource) => (filters.isBookable === "all" ? true : String(resource.is_bookable) === filters.isBookable))
      .filter((resource) => (filters.isActive === "all" ? true : String(resource.is_active) === filters.isActive))
      .sort((left, right) => left.name.localeCompare(right.name));
  }, [categoryMap, filters, resources, typeMap]);

  const hasActiveFilters = useMemo(
    () =>
      filters.search !== defaultFilters.search ||
      filters.categoryId !== defaultFilters.categoryId ||
      filters.typeId !== defaultFilters.typeId ||
      filters.isBookable !== defaultFilters.isBookable ||
      filters.isActive !== defaultFilters.isActive,
    [filters],
  );

  function handleFilterChange<Key extends keyof ResourceFilters>(key: Key, value: ResourceFilters[Key]): void {
    setFilters((currentFilters) => {
      const nextFilters = { ...currentFilters, [key]: value };

      if (key === "categoryId") {
        const isTypeStillAvailable =
          nextFilters.typeId === "all" ||
          sortedTypes.some(
            (type) =>
              String(type.id) === nextFilters.typeId &&
              (nextFilters.categoryId === "all" || String(type.category_id) === nextFilters.categoryId),
          );

        if (!isTypeStillAvailable) {
          nextFilters.typeId = "all";
        }
      }

      return nextFilters;
    });
  }

  function resetFilters(): void {
    setFilters(defaultFilters);
  }

  function resetFormState(): void {
    setFormState(defaultFormState);
    setFormError(null);
    setEditingResourceId(null);
    setFormMode("create");
  }

  function closeForm(): void {
    setIsFormOpen(false);
    resetFormState();
  }

  function openCreateForm(): void {
    resetFormState();
    setFormMode("create");
    setIsFormOpen(true);
  }

  function openEditForm(resource: Resource): void {
    setFormMode("edit");
    setEditingResourceId(resource.id);
    setFormState({
      name: resource.name,
      description: resource.description,
      categoryId: String(resource.category_id),
      typeId: String(resource.type_id),
      departmentId: resource.department_id !== null ? String(resource.department_id) : "",
      location: resource.location ?? "",
      capacity: resource.capacity !== null ? String(resource.capacity) : "",
      isBookable: resource.is_bookable,
      isActive: resource.is_active,
    });
    setFormError(null);
    setIsFormOpen(true);
  }

  function handleFormChange<Key extends keyof ResourceFormState>(key: Key, value: ResourceFormState[Key]): void {
    setFormState((currentState) => {
      const nextState = { ...currentState, [key]: value };

      if (key === "categoryId") {
        const typeStillValid = sortedTypes.some(
          (type) => String(type.id) === nextState.typeId && String(type.category_id) === nextState.categoryId,
        );

        if (!typeStillValid) {
          nextState.typeId = "";
        }
      }

      return nextState;
    });
  }

  function validateForm(): string | null {
    if (!formState.name.trim()) {
      return t("pages.resources.form.errors.nameRequired");
    }

    if (!formState.categoryId || Number(formState.categoryId) <= 0) {
      return t("pages.resources.form.errors.categoryRequired");
    }

    if (!formState.typeId || Number(formState.typeId) <= 0) {
      return t("pages.resources.form.errors.typeRequired");
    }

    const selectedType = types.find((type) => type.id === Number(formState.typeId));
    if (!selectedType || selectedType.category_id !== Number(formState.categoryId)) {
      return t("pages.resources.form.errors.invalidTypeCategory");
    }

    if (formState.departmentId && !departments.some((department) => department.id === Number(formState.departmentId))) {
      return t("pages.resources.form.errors.invalidDepartment");
    }

    if (formState.capacity) {
      const capacity = Number(formState.capacity);
      if (!Number.isFinite(capacity) || capacity < 0) {
        return t("pages.resources.form.errors.capacityNonNegative");
      }
    }

    return null;
  }

  function buildPayload(): ResourcePayload {
    return {
      name: formState.name.trim(),
      description: formState.description.trim(),
      category_id: Number(formState.categoryId),
      type_id: Number(formState.typeId),
      department_id: formState.departmentId ? Number(formState.departmentId) : null,
      location: formState.location.trim() ? formState.location.trim() : null,
      capacity: formState.capacity ? Number(formState.capacity) : null,
      is_bookable: formState.isBookable,
      is_active: formState.isActive,
    };
  }

  async function handleFormSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();

    const validationError = validateForm();
    if (validationError) {
      setFormError(validationError);
      return;
    }

    setFormError(null);
    setIsSubmitting(true);

    try {
      const payload = buildPayload();

      if (formMode === "create") {
        const createdResource = await createResource(payload);
        const catalog = await loadResourcesCatalog();

        if (!catalog) {
          return;
        }

        const freshResource = catalog.resources.find((resource) => resource.id === createdResource.id) ?? createdResource;
        openEditForm(freshResource);
      } else if (editingResourceId !== null) {
        await updateResource(editingResourceId, payload);
        await loadResourcesCatalog();
        closeForm();
      }
    } catch (submitError) {
      if (submitError instanceof ApiError) {
        setFormError(mapResourceError(submitError.message, t));
      } else if (submitError instanceof Error) {
        setFormError(submitError.message);
      } else {
        setFormError(t("pages.resources.form.errors.generic"));
      }
    } finally {
      setIsSubmitting(false);
    }
  }

  function getCategoryName(resource: Resource): string {
    return categoryMap.get(resource.category_id)?.name ?? t("pages.resources.unknownCategory");
  }

  function getTypeName(resource: Resource): string {
    return typeMap.get(resource.type_id)?.name ?? t("pages.resources.unknownType");
  }

  function getDepartmentName(resource: Resource): string {
    if (resource.department_id === null) {
      return t("pages.resources.noDepartment");
    }

    return departmentMap.get(resource.department_id)?.name ?? t("pages.resources.unknownDepartment");
  }

  return (
    <section>
      <PageHeader
        title={t("pages.resources.title")}
        actions={
          <>
            {isAdmin ? (
              <button type="button" className="btn btn-primary" onClick={openCreateForm}>
                {t("pages.resources.actions.create")}
              </button>
            ) : null}
            <button type="button" className="btn btn-secondary" onClick={resetFilters} disabled={!hasActiveFilters}>
              {t("pages.resources.filters.reset")}
            </button>
          </>
        }
      />

      {loading ? <LoadingState message={t("pages.resources.loading")} /> : null}
      {!loading && error ? <ErrorState message={error} onRetry={() => void loadResourcesCatalog()} /> : null}

      {!loading && !error ? (
        <div className="resources-page">
          {isAdmin && isFormOpen ? (
            <div className="resources-form-panel" ref={formPanelRef}>
              <div className="resources-form-panel__header">
                <div>
                  <h3>
                    {formMode === "create"
                      ? t("pages.resources.form.createTitle")
                      : t("pages.resources.form.editTitle")}
                  </h3>
                  <p className="muted">
                    {formMode === "create"
                      ? t("pages.resources.form.createHint")
                      : t("pages.resources.form.editHint")}
                  </p>
                </div>
                <button type="button" className="btn btn-secondary" onClick={closeForm}>
                  {t("pages.resources.actions.closeForm")}
                </button>
              </div>

              <form className="resources-form-grid" onSubmit={handleFormSubmit}>
                <label className="field">
                  <span>{t("pages.resources.form.fields.name")}</span>
                  <input
                    type="text"
                    ref={nameInputRef}
                    value={formState.name}
                    onChange={(event) => handleFormChange("name", event.target.value)}
                    required
                  />
                </label>

                <label className="field">
                  <span>{t("pages.resources.form.fields.category")}</span>
                  <select
                    value={formState.categoryId}
                    onChange={(event) => handleFormChange("categoryId", event.target.value)}
                    required
                  >
                    <option value="">{t("pages.resources.form.fields.selectCategory")}</option>
                    {sortedCategories.map((category) => (
                      <option key={category.id} value={String(category.id)}>
                        {category.name}
                      </option>
                    ))}
                  </select>
                </label>

                <label className="field">
                  <span>{t("pages.resources.form.fields.type")}</span>
                  <select
                    value={formState.typeId}
                    onChange={(event) => handleFormChange("typeId", event.target.value)}
                    required
                  >
                    <option value="">{t("pages.resources.form.fields.selectType")}</option>
                    {formTypeOptions.map((type) => (
                      <option key={type.id} value={String(type.id)}>
                        {type.name}
                      </option>
                    ))}
                  </select>
                </label>

                <label className="field">
                  <span>{t("pages.resources.form.fields.department")}</span>
                  <select
                    value={formState.departmentId}
                    onChange={(event) => handleFormChange("departmentId", event.target.value)}
                  >
                    <option value="">{t("pages.resources.form.fields.noDepartment")}</option>
                    {sortedDepartments.map((department) => (
                      <option key={department.id} value={String(department.id)}>
                        {department.name}
                      </option>
                    ))}
                  </select>
                </label>

                <label className="field">
                  <span>{t("pages.resources.form.fields.location")}</span>
                  <input
                    type="text"
                    value={formState.location}
                    onChange={(event) => handleFormChange("location", event.target.value)}
                  />
                </label>

                <label className="field">
                  <span>{t("pages.resources.form.fields.capacity")}</span>
                  <input
                    type="number"
                    min="0"
                    value={formState.capacity}
                    onChange={(event) => handleFormChange("capacity", event.target.value)}
                  />
                </label>

                <label className="field resources-form-grid__full">
                  <span>{t("pages.resources.form.fields.description")}</span>
                  <textarea
                    value={formState.description}
                    onChange={(event) => handleFormChange("description", event.target.value)}
                    rows={4}
                  />
                </label>

                <ToggleSwitch
                  checked={formState.isBookable}
                  label={t("pages.resources.form.fields.isBookable")}
                  onChange={(checked) => handleFormChange("isBookable", checked)}
                />

                <ToggleSwitch
                  checked={formState.isActive}
                  label={t("pages.resources.form.fields.isActive")}
                  onChange={(checked) => handleFormChange("isActive", checked)}
                />

                {formError ? <p className="error-text resources-form-grid__full">{formError}</p> : null}

                <div className="resources-form-actions resources-form-grid__full">
                  <button type="submit" className="btn btn-primary" disabled={isSubmitting}>
                    {isSubmitting ? t("pages.resources.form.submitting") : t("pages.resources.form.submit")}
                  </button>
                </div>
              </form>
            </div>
          ) : null}

          <div className="resources-filters">
            <label className="field">
              <span>{t("pages.resources.filters.search")}</span>
              <input
                type="search"
                value={filters.search}
                onChange={(event) => handleFilterChange("search", event.target.value)}
                placeholder={t("pages.resources.filters.searchPlaceholder")}
              />
            </label>

            <label className="field">
              <span>{t("pages.resources.filters.category")}</span>
              <select value={filters.categoryId} onChange={(event) => handleFilterChange("categoryId", event.target.value)}>
                <option value="all">{t("pages.resources.filters.allCategories")}</option>
                {sortedCategories.map((category) => (
                  <option key={category.id} value={String(category.id)}>
                    {category.name}
                  </option>
                ))}
              </select>
            </label>

            <label className="field">
              <span>{t("pages.resources.filters.type")}</span>
              <select value={filters.typeId} onChange={(event) => handleFilterChange("typeId", event.target.value)}>
                <option value="all">{t("pages.resources.filters.allTypes")}</option>
                {typeOptions.map((type) => (
                  <option key={type.id} value={String(type.id)}>
                    {type.name}
                  </option>
                ))}
              </select>
            </label>

            <label className="field">
              <span>{t("pages.resources.filters.bookable")}</span>
              <select
                value={filters.isBookable}
                onChange={(event) => handleFilterChange("isBookable", event.target.value as BooleanFilter)}
              >
                <option value="all">{t("pages.resources.filters.allBookableOptions")}</option>
                <option value="true">{t("pages.resources.bookable.true")}</option>
                <option value="false">{t("pages.resources.bookable.false")}</option>
              </select>
            </label>

            <label className="field">
              <span>{t("pages.resources.filters.active")}</span>
              <select
                value={filters.isActive}
                onChange={(event) => handleFilterChange("isActive", event.target.value as BooleanFilter)}
              >
                <option value="all">{t("pages.resources.filters.allActiveOptions")}</option>
                <option value="true">{t("pages.resources.active.true")}</option>
                <option value="false">{t("pages.resources.active.false")}</option>
              </select>
            </label>
          </div>

          <p className="resources-summary muted">
            {t("pages.resources.resultsSummary", { count: filteredResources.length })}
          </p>

          {resources.length === 0 ? (
            <EmptyState title={t("pages.resources.empty.title")} description={t("pages.resources.empty.description")} />
          ) : filteredResources.length === 0 ? (
            <EmptyState
              title={t("pages.resources.noResults.title")}
              description={t("pages.resources.noResults.description")}
            />
          ) : (
            <div className="resource-catalog" role="list">
              {filteredResources.map((resource) => {
                const activeLabel = resource.is_active
                  ? t("pages.resources.active.true")
                  : t("pages.resources.active.false");
                const activeHint = resource.is_active
                  ? t("pages.resources.active.trueHint")
                  : t("pages.resources.active.falseHint");
                const bookableLabel = resource.is_bookable
                  ? t("pages.resources.bookable.true")
                  : t("pages.resources.bookable.false");
                const bookableHint = resource.is_bookable
                  ? t("pages.resources.bookable.trueHint")
                  : t("pages.resources.bookable.falseHint");

                return (
                  <article key={resource.id} className="resource-card" role="listitem">
                    <div className="resource-card__header">
                      <div>
                        <Link to={`/resources/${resource.id}`} className="resource-card__title-link">
                          <h3 className="resource-card__title">{resource.name}</h3>
                        </Link>
                        <p className="resource-card__description">
                          {resource.description || t("pages.resources.noDescription")}
                        </p>
                      </div>
                      <div className="resource-card__badges">
                        <span
                          className={`badge ${resource.is_active ? "badge-success" : "badge-muted"}`}
                          title={activeHint}
                        >
                          {activeLabel}
                        </span>
                        <span
                          className={`badge ${resource.is_bookable ? "badge-info" : "badge-muted"}`}
                          title={bookableHint}
                        >
                          {bookableLabel}
                        </span>
                      </div>
                    </div>

                    <dl className="resource-card__meta">
                      <div>
                        <dt>{t("pages.resources.fields.category")}</dt>
                        <dd>{getCategoryName(resource)}</dd>
                      </div>
                      <div>
                        <dt>{t("pages.resources.fields.type")}</dt>
                        <dd>{getTypeName(resource)}</dd>
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
                      <div>
                        <dt>{t("pages.resources.fields.department")}</dt>
                        <dd>{getDepartmentName(resource)}</dd>
                      </div>
                    </dl>

                    <div className="resource-card__actions">
                      <Link to={`/resources/${resource.id}`} className="btn btn-secondary">
                        {t("pages.resources.actions.open")}
                      </Link>
                      {isAdmin ? (
                        <button type="button" className="btn btn-secondary" onClick={() => openEditForm(resource)}>
                          {t("pages.resources.actions.edit")}
                        </button>
                      ) : null}
                    </div>
                  </article>
                );
              })}
            </div>
          )}
        </div>
      ) : null}
    </section>
  );
}
