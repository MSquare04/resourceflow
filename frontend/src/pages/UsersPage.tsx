import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

import { listDepartments } from "../api/departments";
import { ApiError } from "../api/client";
import { createUser, listUsers, replaceUserRoles, updateUser } from "../api/users";
import { EmptyState } from "../components/EmptyState";
import { ErrorState } from "../components/ErrorState";
import { LoadingState } from "../components/LoadingState";
import { PageHeader } from "../components/PageHeader";
import type { RoleCode } from "../types/auth";
import type { CreateUserPayload, Department, UpdateUserPayload, User } from "../types/users";

type ActiveFilter = "all" | "active" | "inactive";
type FormMode = "create" | "edit";

interface UserFilters {
  search: string;
  role: "all" | RoleCode;
  active: ActiveFilter;
}

interface UserFormState {
  fullName: string;
  email: string;
  password: string;
  departmentId: string;
  isActive: boolean;
  roles: RoleCode[];
}

const roleOptions: RoleCode[] = ["admin", "manager", "employee", "hr", "interviewer"];

const defaultFilters: UserFilters = {
  search: "",
  role: "all",
  active: "all",
};

const defaultFormState: UserFormState = {
  fullName: "",
  email: "",
  password: "",
  departmentId: "",
  isActive: true,
  roles: [],
};

function isValidEmail(value: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);
}

function mapUserError(message: string, t: ReturnType<typeof useTranslation>["t"]): string {
  switch (message) {
    case "invalid user payload":
      return t("pages.users.form.errors.invalidPayload");
    case "user email already exists":
      return t("pages.users.form.errors.emailExists");
    case "one or more role codes are invalid":
      return t("pages.users.form.errors.invalidRoles");
    case "user not found":
      return t("pages.users.form.errors.userNotFound");
    default:
      return message;
  }
}

export function UsersPage(): JSX.Element {
  const { t } = useTranslation();
  const [users, setUsers] = useState<User[]>([]);
  const [departments, setDepartments] = useState<Department[]>([]);
  const [filters, setFilters] = useState<UserFilters>(defaultFilters);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isFormOpen, setIsFormOpen] = useState(false);
  const [formMode, setFormMode] = useState<FormMode>("create");
  const [editingUserId, setEditingUserId] = useState<number | null>(null);
  const [formState, setFormState] = useState<UserFormState>(defaultFormState);
  const [formError, setFormError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    void loadUsersPage();
  }, []);

  async function loadUsersPage(): Promise<void> {
    setLoading(true);
    setError(null);

    try {
      const [usersData, departmentsData] = await Promise.all([listUsers(), listDepartments()]);
      setUsers(usersData);
      setDepartments(departmentsData);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : t("errors.generic"));
    } finally {
      setLoading(false);
    }
  }

  const departmentMap = useMemo(() => new Map(departments.map((department) => [department.id, department.name])), [departments]);

  const filteredUsers = useMemo(() => {
    const normalizedSearch = filters.search.trim().toLowerCase();

    return [...users]
      .filter((user) => {
        if (!normalizedSearch) {
          return true;
        }

        return [user.full_name, user.email].some((value) => value.toLowerCase().includes(normalizedSearch));
      })
      .filter((user) => (filters.role === "all" ? true : user.roles.includes(filters.role)))
      .filter((user) => {
        if (filters.active === "all") {
          return true;
        }

        return filters.active === "active" ? user.is_active : !user.is_active;
      })
      .sort((left, right) => left.full_name.localeCompare(right.full_name));
  }, [filters, users]);

  const hasActiveFilters =
    filters.search !== defaultFilters.search || filters.role !== defaultFilters.role || filters.active !== defaultFilters.active;

  function resetFilters(): void {
    setFilters(defaultFilters);
  }

  function closeForm(): void {
    setIsFormOpen(false);
    setFormMode("create");
    setEditingUserId(null);
    setFormState(defaultFormState);
    setFormError(null);
    setIsSubmitting(false);
  }

  function openCreateForm(): void {
    setFormMode("create");
    setEditingUserId(null);
    setFormState(defaultFormState);
    setFormError(null);
    setIsFormOpen(true);
  }

  function openEditForm(user: User): void {
    setFormMode("edit");
    setEditingUserId(user.id);
    setFormState({
      fullName: user.full_name,
      email: user.email,
      password: "",
      departmentId: user.department_id !== null ? String(user.department_id) : "",
      isActive: user.is_active,
      roles: user.roles,
    });
    setFormError(null);
    setIsFormOpen(true);
  }

  function toggleRole(role: RoleCode): void {
    setFormState((current) => ({
      ...current,
      roles: current.roles.includes(role) ? current.roles.filter((item) => item !== role) : [...current.roles, role],
    }));
  }

  function validateForm(): string | null {
    if (!formState.fullName.trim()) {
      return t("pages.users.form.errors.fullNameRequired");
    }

    if (!formState.email.trim()) {
      return t("pages.users.form.errors.emailRequired");
    }

    if (!isValidEmail(formState.email.trim())) {
      return t("pages.users.form.errors.emailInvalid");
    }

    if (formMode === "create" && !formState.password.trim()) {
      return t("pages.users.form.errors.passwordRequired");
    }

    if (formState.roles.length === 0) {
      return t("pages.users.form.errors.rolesRequired");
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

    try {
      if (formMode === "create") {
        const payload: CreateUserPayload = {
          full_name: formState.fullName.trim(),
          email: formState.email.trim().toLowerCase(),
          password: formState.password.trim(),
          department_id: formState.departmentId ? Number(formState.departmentId) : null,
          is_active: formState.isActive,
          roles: formState.roles,
        };

        await createUser(payload);
      } else if (editingUserId !== null) {
        const payload: UpdateUserPayload = {
          full_name: formState.fullName.trim(),
          email: formState.email.trim().toLowerCase(),
          password: formState.password.trim(),
          department_id: formState.departmentId ? Number(formState.departmentId) : null,
          is_active: formState.isActive,
        };

        await updateUser(editingUserId, payload);
        await replaceUserRoles(editingUserId, formState.roles);
      }

      await loadUsersPage();
      closeForm();
    } catch (submitError) {
      if (submitError instanceof ApiError) {
        setFormError(mapUserError(submitError.message, t));
      } else if (submitError instanceof Error) {
        setFormError(submitError.message);
      } else {
        setFormError(t("pages.users.form.errors.generic"));
      }
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <section>
      <PageHeader
        title={t("pages.users.title")}
        actions={
          <button type="button" className="btn btn-primary" onClick={openCreateForm}>
            {t("pages.users.actions.create")}
          </button>
        }
      />

      {isFormOpen ? (
        <div className="users-form-panel">
          <div className="users-form-panel__header">
            <div>
              <h3>{formMode === "create" ? t("pages.users.form.createTitle") : t("pages.users.form.editTitle")}</h3>
              <p className="muted">
                {formMode === "create" ? t("pages.users.form.createHint") : t("pages.users.form.editHint")}
              </p>
            </div>
            <button type="button" className="btn btn-secondary" onClick={closeForm} disabled={isSubmitting}>
              {t("pages.users.actions.closeForm")}
            </button>
          </div>

          <div className="users-form-grid">
            <label className="field">
              <span>{t("pages.users.form.fields.fullName")}</span>
              <input
                type="text"
                value={formState.fullName}
                onChange={(event) => setFormState((current) => ({ ...current, fullName: event.target.value }))}
                required
              />
            </label>

            <label className="field">
              <span>{t("pages.users.form.fields.email")}</span>
              <input
                type="email"
                value={formState.email}
                onChange={(event) => setFormState((current) => ({ ...current, email: event.target.value }))}
                required
              />
            </label>

            <label className="field">
              <span>
                {formMode === "create"
                  ? t("pages.users.form.fields.password")
                  : t("pages.users.form.fields.passwordOptional")}
              </span>
              <input
                type="password"
                value={formState.password}
                onChange={(event) => setFormState((current) => ({ ...current, password: event.target.value }))}
                required={formMode === "create"}
              />
            </label>

            <label className="field">
              <span>{t("pages.users.form.fields.department")}</span>
              <select
                value={formState.departmentId}
                onChange={(event) => setFormState((current) => ({ ...current, departmentId: event.target.value }))}
              >
                <option value="">{t("pages.users.form.fields.noDepartment")}</option>
                {departments
                  .filter((department) => department.is_active)
                  .sort((left, right) => left.name.localeCompare(right.name))
                  .map((department) => (
                    <option key={department.id} value={String(department.id)}>
                      {department.name}
                    </option>
                  ))}
              </select>
            </label>

            <label className="field field-checkbox">
              <span>{t("pages.users.form.fields.isActive")}</span>
              <input
                type="checkbox"
                checked={formState.isActive}
                onChange={(event) => setFormState((current) => ({ ...current, isActive: event.target.checked }))}
              />
            </label>
          </div>

          <div className="users-roles-field">
            <span className="users-roles-field__label">{t("pages.users.form.fields.roles")}</span>
            <div className="users-roles-options">
              {roleOptions.map((role) => (
                <label key={role} className="users-role-option">
                  <input
                    type="checkbox"
                    checked={formState.roles.includes(role)}
                    onChange={() => toggleRole(role)}
                  />
                  <span>{t(`roles.${role}`)}</span>
                </label>
              ))}
            </div>
          </div>

          {formError ? <p className="error-text">{formError}</p> : null}

          <div className="users-form-actions">
            <button type="button" className="btn btn-primary" onClick={() => void handleSubmit()} disabled={isSubmitting}>
              {isSubmitting ? t("pages.users.form.submitting") : t("pages.users.form.submit")}
            </button>
          </div>
        </div>
      ) : null}

      {loading ? <LoadingState message={t("pages.users.loading")} /> : null}
      {!loading && error ? <ErrorState message={error} onRetry={() => void loadUsersPage()} /> : null}
      {!loading && !error && users.length === 0 ? (
        <EmptyState title={t("pages.users.empty.title")} description={t("pages.users.empty.description")} />
      ) : null}

      {!loading && !error && users.length > 0 ? (
        <div className="users-page">
          <div className="users-filters">
            <label className="field">
              <span>{t("pages.users.filters.search")}</span>
              <input
                type="search"
                value={filters.search}
                onChange={(event) => setFilters((current) => ({ ...current, search: event.target.value }))}
                placeholder={t("pages.users.filters.searchPlaceholder")}
              />
            </label>

            <label className="field">
              <span>{t("pages.users.filters.role")}</span>
              <select
                value={filters.role}
                onChange={(event) => setFilters((current) => ({ ...current, role: event.target.value as UserFilters["role"] }))}
              >
                <option value="all">{t("pages.users.filters.allRoles")}</option>
                {roleOptions.map((role) => (
                  <option key={role} value={role}>
                    {t(`roles.${role}`)}
                  </option>
                ))}
              </select>
            </label>

            <label className="field">
              <span>{t("pages.users.filters.activity")}</span>
              <select
                value={filters.active}
                onChange={(event) =>
                  setFilters((current) => ({ ...current, active: event.target.value as ActiveFilter }))
                }
              >
                <option value="all">{t("pages.users.filters.allActivity")}</option>
                <option value="active">{t("pages.users.filters.active")}</option>
                <option value="inactive">{t("pages.users.filters.inactive")}</option>
              </select>
            </label>

            <button type="button" className="btn btn-secondary" onClick={resetFilters} disabled={!hasActiveFilters}>
              {t("pages.users.filters.reset")}
            </button>
          </div>

          <p className="resources-summary muted">{t("pages.users.resultsSummary", { count: filteredUsers.length })}</p>

          {filteredUsers.length === 0 ? (
            <EmptyState
              title={t("pages.users.noResults.title")}
              description={t("pages.users.noResults.description")}
            />
          ) : (
            <div className="users-list" role="list">
              {filteredUsers.map((user) => (
                <article key={user.id} className="user-card" role="listitem">
                  <div className="user-card__header">
                    <div className="user-card__heading">
                      <h3 className="user-card__title">{user.full_name}</h3>
                      <p className="user-card__email">{user.email}</p>
                    </div>
                    <span className={`badge ${user.is_active ? "badge-success" : "badge-muted"}`}>
                      {user.is_active ? t("pages.users.status.active") : t("pages.users.status.inactive")}
                    </span>
                  </div>

                  <dl className="user-card__meta">
                    <div>
                      <dt>{t("pages.users.fields.department")}</dt>
                      <dd>{user.department_id !== null ? departmentMap.get(user.department_id) ?? t("pages.users.fields.unknownDepartment") : t("pages.users.fields.noDepartment")}</dd>
                    </div>
                    <div>
                      <dt>{t("pages.users.fields.roles")}</dt>
                      <dd className="user-card__roles">
                        {user.roles.length > 0
                          ? user.roles.map((role) => (
                              <span key={role} className="role-chip">
                                {t(`roles.${role}`)}
                              </span>
                            ))
                          : t("pages.users.fields.noRoles")}
                      </dd>
                    </div>
                  </dl>

                  <div className="user-card__actions">
                    <button type="button" className="btn btn-secondary" onClick={() => openEditForm(user)}>
                      {t("pages.users.actions.edit")}
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
