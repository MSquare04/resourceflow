import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

import { listResourceCategories, listResources, listResourceTypes } from "../api/resources";
import { EmptyState } from "../components/EmptyState";
import { ErrorState } from "../components/ErrorState";
import { LoadingState } from "../components/LoadingState";
import { PageHeader } from "../components/PageHeader";
import type { Resource, ResourceCategory, ResourceType } from "../types/resources";

type BooleanFilter = "all" | "true" | "false";

interface ResourceFilters {
  search: string;
  categoryId: string;
  typeId: string;
  isBookable: BooleanFilter;
  isActive: BooleanFilter;
}

const defaultFilters: ResourceFilters = {
  search: "",
  categoryId: "all",
  typeId: "all",
  isBookable: "all",
  isActive: "all",
};

export function ResourcesPage(): JSX.Element {
  const { t } = useTranslation();
  const [resources, setResources] = useState<Resource[]>([]);
  const [categories, setCategories] = useState<ResourceCategory[]>([]);
  const [types, setTypes] = useState<ResourceType[]>([]);
  const [filters, setFilters] = useState<ResourceFilters>(defaultFilters);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    void loadResourcesCatalog();
  }, []);

  async function loadResourcesCatalog(): Promise<void> {
    setLoading(true);
    setError(null);

    try {
      const [resourcesData, categoriesData, typesData] = await Promise.all([
        listResources(),
        listResourceCategories(),
        listResourceTypes(),
      ]);

      setResources(resourcesData);
      setCategories(categoriesData);
      setTypes(typesData);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : t("errors.generic"));
    } finally {
      setLoading(false);
    }
  }

  const sortedCategories = useMemo(
    () => [...categories].sort((left, right) => left.name.localeCompare(right.name)),
    [categories],
  );
  const sortedTypes = useMemo(() => [...types].sort((left, right) => left.name.localeCompare(right.name)), [types]);

  const categoryMap = useMemo(() => new Map(categories.map((category) => [category.id, category])), [categories]);
  const typeMap = useMemo(() => new Map(types.map((type) => [type.id, type])), [types]);

  const typeOptions = useMemo(() => {
    if (filters.categoryId === "all") {
      return sortedTypes;
    }

    return sortedTypes.filter((type) => String(type.category_id) === filters.categoryId);
  }, [filters.categoryId, sortedTypes]);

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

  function getCategoryName(resource: Resource): string {
    return categoryMap.get(resource.category_id)?.name ?? t("pages.resources.unknownCategory");
  }

  function getTypeName(resource: Resource): string {
    return typeMap.get(resource.type_id)?.name ?? t("pages.resources.unknownType");
  }

  return (
    <section>
      <PageHeader
        title={t("pages.resources.title")}
        actions={
          <button type="button" className="btn btn-secondary" onClick={resetFilters} disabled={!hasActiveFilters}>
            {t("pages.resources.filters.reset")}
          </button>
        }
      />

      {loading ? <LoadingState message={t("pages.resources.loading")} /> : null}
      {!loading && error ? <ErrorState message={error} onRetry={() => void loadResourcesCatalog()} /> : null}
      {!loading && !error && resources.length === 0 ? (
        <EmptyState
          title={t("pages.resources.empty.title")}
          description={t("pages.resources.empty.description")}
        />
      ) : null}

      {!loading && !error && resources.length > 0 ? (
        <div className="resources-page">
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
              <select
                value={filters.categoryId}
                onChange={(event) => handleFilterChange("categoryId", event.target.value)}
              >
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

          {filteredResources.length === 0 ? (
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
                        <h3 className="resource-card__title">{resource.name}</h3>
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
                    </dl>
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
