import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";

import { listBookingRules } from "../api/bookingRules";
import { listBookings, listMyBookings } from "../api/bookings";
import { listResources } from "../api/resources";
import { listUsers } from "../api/users";
import { useRoles } from "../auth/useRoles";
import { EmptyState } from "../components/EmptyState";
import { ErrorState } from "../components/ErrorState";
import { LoadingState } from "../components/LoadingState";
import { PageHeader } from "../components/PageHeader";
import { StatusBadge } from "../components/StatusBadge";
import type { Booking } from "../types/bookings";
import type { BookingRule } from "../types/bookingRules";
import type { Resource } from "../types/resources";
import type { User } from "../types/users";
import { formatUtcDateTime } from "../utils/datetime";

interface DashboardData {
  resources: Resource[];
  myBookings: Booking[];
  bookings: Booking[] | null;
  users: User[] | null;
  bookingRules: BookingRule[] | null;
}

interface OptionalLoadState {
  bookingsError: string | null;
  usersError: string | null;
  bookingRulesError: string | null;
}

interface QuickLinkItem {
  to: string;
  label: string;
}

export function DashboardPage(): JSX.Element {
  const { t } = useTranslation();
  const { hasRole, hasAnyRole } = useRoles();
  const isAdmin = hasRole("admin");
  const canManageBookings = hasAnyRole(["admin", "manager"]);

  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [optionalErrors, setOptionalErrors] = useState<OptionalLoadState>({
    bookingsError: null,
    usersError: null,
    bookingRulesError: null,
  });

  useEffect(() => {
    void loadDashboard();
  }, [canManageBookings, isAdmin]);

  async function loadDashboard(): Promise<void> {
    setLoading(true);
    setError(null);
    setOptionalErrors({
      bookingsError: null,
      usersError: null,
      bookingRulesError: null,
    });

    try {
      const [resources, myBookings] = await Promise.all([listResources(), listMyBookings()]);

      const nextData: DashboardData = {
        resources,
        myBookings,
        bookings: null,
        users: null,
        bookingRules: null,
      };

      const nextOptionalErrors: OptionalLoadState = {
        bookingsError: null,
        usersError: null,
        bookingRulesError: null,
      };

      const optionalRequests: Array<Promise<void>> = [];

      if (canManageBookings) {
        optionalRequests.push(
          listBookings()
            .then((bookings) => {
              nextData.bookings = bookings;
            })
            .catch((requestError) => {
              nextOptionalErrors.bookingsError =
                requestError instanceof Error ? requestError.message : t("errors.generic");
            }),
        );
      }

      if (isAdmin) {
        optionalRequests.push(
          listUsers()
            .then((users) => {
              nextData.users = users;
            })
            .catch((requestError) => {
              nextOptionalErrors.usersError =
                requestError instanceof Error ? requestError.message : t("errors.generic");
            }),
        );

        optionalRequests.push(
          listBookingRules()
            .then((bookingRules) => {
              nextData.bookingRules = bookingRules;
            })
            .catch((requestError) => {
              nextOptionalErrors.bookingRulesError =
                requestError instanceof Error ? requestError.message : t("errors.generic");
            }),
        );
      }

      await Promise.all(optionalRequests);

      setData(nextData);
      setOptionalErrors(nextOptionalErrors);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : t("errors.generic"));
      setData(null);
    } finally {
      setLoading(false);
    }
  }

  const futureMyBookings = useMemo(() => {
    const now = Date.now();
    return (data?.myBookings ?? [])
      .filter((booking) => {
        const startsInFuture = new Date(booking.start_at).getTime() > now;
        const isUpcomingStatus = booking.status === "pending" || booking.status === "confirmed";
        return startsInFuture && isUpcomingStatus;
      })
      .sort((left, right) => new Date(left.start_at).getTime() - new Date(right.start_at).getTime());
  }, [data?.myBookings]);

  const nearestBookings = useMemo(() => futureMyBookings.slice(0, 5), [futureMyBookings]);

  const availableBookableResources = useMemo(
    () => (data?.resources ?? []).filter((resource) => resource.is_active && resource.is_bookable).length,
    [data?.resources],
  );

  const pendingApprovalsCount = useMemo(
    () => (data?.bookings ?? []).filter((booking) => booking.status === "pending").length,
    [data?.bookings],
  );

  const activeUsersCount = useMemo(
    () => (data?.users ?? []).filter((user) => user.is_active).length,
    [data?.users],
  );

  const activeRulesCount = useMemo(
    () => (data?.bookingRules ?? []).filter((rule) => rule.is_active).length,
    [data?.bookingRules],
  );

  const quickLinks = useMemo<QuickLinkItem[]>(() => {
    const items: QuickLinkItem[] = [
      { to: "/resources", label: t("pages.dashboard.links.resources") },
      { to: "/my-bookings", label: t("pages.dashboard.links.myBookings") },
    ];

    if (canManageBookings) {
      items.push({ to: "/bookings", label: t("pages.dashboard.links.bookings") });
    }

    if (isAdmin) {
      items.push({ to: "/users", label: t("pages.dashboard.links.users") });
      items.push({ to: "/booking-rules", label: t("pages.dashboard.links.bookingRules") });
    }

    return items;
  }, [canManageBookings, isAdmin, t]);

  const optionalWarnings = [
    optionalErrors.bookingsError ? t("pages.dashboard.warnings.bookings") : null,
    optionalErrors.usersError ? t("pages.dashboard.warnings.users") : null,
    optionalErrors.bookingRulesError ? t("pages.dashboard.warnings.bookingRules") : null,
  ].filter(Boolean) as string[];

  if (loading) {
    return (
      <section>
        <LoadingState message={t("pages.dashboard.loading")} />
      </section>
    );
  }

  if (error || !data) {
    return (
      <section>
        <PageHeader title={t("pages.dashboard.title")} />
        <ErrorState message={error ?? t("errors.generic")} onRetry={() => void loadDashboard()} />
      </section>
    );
  }

  return (
    <section className="dashboard-page">
      <PageHeader title={t("pages.dashboard.title")} />

      {optionalWarnings.length > 0 ? (
        <div className="feedback-card">
          <h3>{t("pages.dashboard.warnings.title")}</h3>
          <p className="muted">{optionalWarnings.join(" ")}</p>
        </div>
      ) : null}

      <div className="dashboard-stats">
        <article className="dashboard-stat-card">
          <span className="dashboard-stat-card__label">{t("pages.dashboard.stats.bookableResources")}</span>
          <strong className="dashboard-stat-card__value">{availableBookableResources}</strong>
        </article>

        <article className="dashboard-stat-card">
          <span className="dashboard-stat-card__label">{t("pages.dashboard.stats.myFutureBookings")}</span>
          <strong className="dashboard-stat-card__value">{futureMyBookings.length}</strong>
        </article>

        {canManageBookings ? (
          <article className="dashboard-stat-card">
            <span className="dashboard-stat-card__label">{t("pages.dashboard.stats.pendingApprovals")}</span>
            <strong className="dashboard-stat-card__value">{pendingApprovalsCount}</strong>
          </article>
        ) : null}

        {isAdmin ? (
          <>
            <article className="dashboard-stat-card">
              <span className="dashboard-stat-card__label">{t("pages.dashboard.stats.users")}</span>
              <strong className="dashboard-stat-card__value">{activeUsersCount}</strong>
            </article>

            <article className="dashboard-stat-card">
              <span className="dashboard-stat-card__label">{t("pages.dashboard.stats.activeRules")}</span>
              <strong className="dashboard-stat-card__value">{activeRulesCount}</strong>
            </article>
          </>
        ) : null}
      </div>

      <div className="dashboard-grid">
        <div className="dashboard-panel">
          <div className="dashboard-panel__header">
            <h3>{t("pages.dashboard.sections.nearestBookings")}</h3>
          </div>

          {nearestBookings.length === 0 ? (
            <EmptyState
              title={t("pages.dashboard.empty.nearestBookings.title")}
              description={t("pages.dashboard.empty.nearestBookings.description")}
            />
          ) : (
            <div className="dashboard-bookings-list" role="list">
              {nearestBookings.map((booking) => (
                <article key={booking.id} className="dashboard-booking-card" role="listitem">
                  <div className="dashboard-booking-card__header">
                    <div className="dashboard-booking-card__heading">
                      <h4>{booking.resource_name}</h4>
                      <StatusBadge status={booking.status} />
                    </div>
                  </div>

                  <div className="dashboard-booking-card__time">
                    <div>
                      <span className="dashboard-booking-card__label">{t("pages.myBookings.fields.startAt")}</span>
                      <strong>{formatUtcDateTime(booking.start_at)}</strong>
                    </div>
                    <div>
                      <span className="dashboard-booking-card__label">{t("pages.myBookings.fields.endAt")}</span>
                      <strong>{formatUtcDateTime(booking.end_at)}</strong>
                    </div>
                  </div>

                  <p className="dashboard-booking-card__purpose">
                    {booking.purpose || t("pages.myBookings.noPurpose")}
                  </p>
                </article>
              ))}
            </div>
          )}
        </div>

        <div className="dashboard-panel">
          <div className="dashboard-panel__header">
            <h3>{t("pages.dashboard.sections.quickLinks")}</h3>
          </div>

          <div className="dashboard-links" role="list">
            {quickLinks.map((link) => (
              <Link key={link.to} to={link.to} className="dashboard-link-card" role="listitem">
                <span>{link.label}</span>
              </Link>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
