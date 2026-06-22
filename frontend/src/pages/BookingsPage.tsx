import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

import { approveBooking, cancelBooking, listBookings, rejectBooking } from "../api/bookings";
import { ApiError } from "../api/client";
import { EmptyState } from "../components/EmptyState";
import { ErrorState } from "../components/ErrorState";
import { LoadingState } from "../components/LoadingState";
import { PageHeader } from "../components/PageHeader";
import { StatusBadge } from "../components/StatusBadge";
import type { Booking, BookingStatus } from "../types/bookings";
import { formatUtcDateTime } from "../utils/datetime";

type BookingAction = "approve" | "reject" | "cancel";
type BookingsTab = "pending" | "all";
type StatusFilter = "all" | BookingStatus;

interface BookingFilters {
  search: string;
  status: StatusFilter;
}

const defaultFilters: BookingFilters = {
  search: "",
  status: "all",
};

function canApproveOrReject(status: BookingStatus): boolean {
  return status === "pending";
}

function canCancel(status: BookingStatus, endAt: string): boolean {
  return (status === "pending" || status === "confirmed") && new Date(endAt).getTime() > Date.now();
}

function mapActionError(error: ApiError, t: ReturnType<typeof useTranslation>["t"]): string {
  switch (error.code) {
    case "booking_cancel_not_allowed":
      return t("pages.bookings.actions.errors.invalidTransition");
    case "booking_forbidden":
      return t("pages.bookings.actions.errors.forbidden");
    case "booking_already_ended":
      return t("pages.bookings.actions.errors.alreadyEnded");
    case "not_found":
      return t("pages.bookings.actions.errors.notFound");
    default:
      return error.message;
  }
}

export function BookingsPage(): JSX.Element {
  const { t } = useTranslation();
  const [bookings, setBookings] = useState<Booking[]>([]);
  const [activeTab, setActiveTab] = useState<BookingsTab>("pending");
  const [filters, setFilters] = useState<BookingFilters>(defaultFilters);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [pendingActionId, setPendingActionId] = useState<number | null>(null);
  const [pendingActionType, setPendingActionType] = useState<BookingAction | null>(null);

  useEffect(() => {
    void loadBookings();
  }, []);

  async function loadBookings(): Promise<void> {
    setLoading(true);
    setError(null);

    try {
      const data = await listBookings();
      setBookings(data);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : t("errors.generic"));
    } finally {
      setLoading(false);
    }
  }

  async function handleBookingAction(booking: Booking, action: BookingAction): Promise<void> {
    const confirmationMessage =
      action === "approve"
        ? t("pages.bookings.confirmations.approvePrompt", {
            resourceName: booking.resource_name,
            defaultValue: `Approve the booking for "${booking.resource_name}"?`,
          })
        : action === "reject"
          ? t("pages.bookings.confirmations.rejectPrompt", {
              resourceName: booking.resource_name,
              defaultValue: `Reject the booking for "${booking.resource_name}"?`,
            })
          : t("pages.bookings.confirmations.cancelPrompt", {
              resourceName: booking.resource_name,
              defaultValue: `Cancel the booking for "${booking.resource_name}"?`,
            });

    if (!window.confirm(confirmationMessage)) {
      return;
    }

    setActionError(null);
    setPendingActionId(booking.id);
    setPendingActionType(action);

    try {
      if (action === "approve") {
        await approveBooking(booking.id);
      } else if (action === "reject") {
        await rejectBooking(booking.id);
      } else {
        await cancelBooking(booking.id);
      }

      await loadBookings();
    } catch (requestError) {
      if (requestError instanceof ApiError) {
        setActionError(mapActionError(requestError, t));
      } else if (requestError instanceof Error) {
        setActionError(requestError.message);
      } else {
        setActionError(t("pages.bookings.actions.errors.generic"));
      }
    } finally {
      setPendingActionId(null);
      setPendingActionType(null);
    }
  }

  const visibleBookings = useMemo(() => {
    const normalizedSearch = filters.search.trim().toLowerCase();
    const tabFiltered = activeTab === "pending" ? bookings.filter((booking) => booking.status === "pending") : bookings;

    return [...tabFiltered]
      .filter((booking) => (filters.status === "all" ? true : booking.status === filters.status))
      .filter((booking) => {
        if (!normalizedSearch) {
          return true;
        }

        const searchableFields = [
          booking.resource_name,
          booking.user_full_name ?? "",
          booking.purpose ?? "",
        ];

        return searchableFields.some((value) => value.toLowerCase().includes(normalizedSearch));
      })
      .sort((left, right) => new Date(left.start_at).getTime() - new Date(right.start_at).getTime());
  }, [activeTab, bookings, filters]);

  const hasActiveFilters = filters.search !== defaultFilters.search || filters.status !== defaultFilters.status;

  function resetFilters(): void {
    setFilters(defaultFilters);
  }

  function isActionPending(bookingId: number, action: BookingAction): boolean {
    return pendingActionId === bookingId && pendingActionType === action;
  }

  return (
    <section>
      <PageHeader
        title={t("pages.bookings.title")}
        actions={
          <button type="button" className="btn btn-secondary" onClick={resetFilters} disabled={!hasActiveFilters}>
            {t("pages.bookings.filters.reset")}
          </button>
        }
      />

      {loading ? <LoadingState message={t("pages.bookings.loading")} /> : null}
      {!loading && error ? <ErrorState message={error} onRetry={() => void loadBookings()} /> : null}
      {!loading && !error && bookings.length === 0 ? (
        <EmptyState title={t("pages.bookings.empty.title")} description={t("pages.bookings.empty.description")} />
      ) : null}

      {!loading && !error && bookings.length > 0 ? (
        <div className="bookings-page">
          <div className="bookings-tabs" role="tablist" aria-label={t("pages.bookings.tabs.label")}>
            <button
              type="button"
              role="tab"
              aria-selected={activeTab === "pending"}
              className={`bookings-tab ${activeTab === "pending" ? "active" : ""}`}
              onClick={() => setActiveTab("pending")}
            >
              {t("pages.bookings.tabs.pending")}
            </button>
            <button
              type="button"
              role="tab"
              aria-selected={activeTab === "all"}
              className={`bookings-tab ${activeTab === "all" ? "active" : ""}`}
              onClick={() => setActiveTab("all")}
            >
              {t("pages.bookings.tabs.all")}
            </button>
          </div>

          <div className="bookings-filters">
            <label className="field">
              <span>{t("pages.bookings.filters.search")}</span>
              <input
                type="search"
                value={filters.search}
                onChange={(event) => setFilters((current) => ({ ...current, search: event.target.value }))}
                placeholder={t("pages.bookings.filters.searchPlaceholder")}
              />
            </label>

            <label className="field">
              <span>{t("pages.bookings.filters.status")}</span>
              <select
                value={filters.status}
                onChange={(event) => setFilters((current) => ({ ...current, status: event.target.value as StatusFilter }))}
              >
                <option value="all">{t("pages.bookings.filters.allStatuses")}</option>
                <option value="pending">{t("statuses.pending")}</option>
                <option value="confirmed">{t("statuses.confirmed")}</option>
                <option value="rejected">{t("statuses.rejected")}</option>
                <option value="cancelled">{t("statuses.cancelled")}</option>
                <option value="completed">{t("statuses.completed")}</option>
              </select>
            </label>
          </div>

          {actionError ? <p className="error-text">{actionError}</p> : null}

          <p className="resources-summary muted">{t("pages.bookings.resultsSummary", { count: visibleBookings.length })}</p>

          {visibleBookings.length === 0 ? (
            <EmptyState
              title={t("pages.bookings.noResults.title")}
              description={t("pages.bookings.noResults.description")}
            />
          ) : (
            <div className="bookings-list" role="list">
              {visibleBookings.map((booking) => (
                <article key={booking.id} className="booking-card" role="listitem">
                  <div className="booking-card__header">
                    <StatusBadge status={booking.status} />
                  </div>

                  <div className="booking-card__heading">
                    <h3 className="booking-card__title">{booking.resource_name}</h3>
                  </div>

                  <div className="booking-card__time-block">
                    <div className="booking-card__time-item">
                      <span className="booking-card__time-label">{t("pages.bookings.fields.startAt")}</span>
                      <strong>{formatUtcDateTime(booking.start_at)}</strong>
                    </div>
                    <div className="booking-card__time-item">
                      <span className="booking-card__time-label">{t("pages.bookings.fields.endAt")}</span>
                      <strong>{formatUtcDateTime(booking.end_at)}</strong>
                    </div>
                  </div>

                  <dl className="booking-card__meta">
                    <div>
                      <dt>{t("pages.bookings.fields.user")}</dt>
                      <dd>{booking.user_full_name || t("pages.bookings.unknownUser")}</dd>
                    </div>
                    <div>
                      <dt>{t("pages.bookings.fields.purpose")}</dt>
                      <dd>{booking.purpose || t("pages.bookings.noPurpose")}</dd>
                    </div>
                  </dl>

                  <div className="booking-card__actions">
                    {canApproveOrReject(booking.status) ? (
                      <>
                        <button
                          type="button"
                          className="btn btn-primary"
                          disabled={pendingActionId === booking.id}
                          onClick={() => void handleBookingAction(booking, "approve")}
                        >
                          {isActionPending(booking.id, "approve")
                            ? t("pages.bookings.actions.processing")
                            : t("pages.bookings.actions.approve")}
                        </button>

                        <button
                          type="button"
                          className="btn btn-secondary"
                          disabled={pendingActionId === booking.id}
                          onClick={() => void handleBookingAction(booking, "reject")}
                        >
                          {isActionPending(booking.id, "reject")
                            ? t("pages.bookings.actions.processing")
                            : t("pages.bookings.actions.reject")}
                        </button>
                      </>
                    ) : null}

                    {canCancel(booking.status, booking.end_at) ? (
                      <button
                        type="button"
                        className="btn btn-secondary"
                        disabled={pendingActionId === booking.id}
                        onClick={() => void handleBookingAction(booking, "cancel")}
                      >
                        {isActionPending(booking.id, "cancel")
                          ? t("pages.bookings.actions.processing")
                          : t("pages.bookings.actions.cancel")}
                      </button>
                    ) : null}
                  </div>

                  <p className="booking-card__footer-meta muted">
                    {t("pages.bookings.fields.createdAt")}: {formatUtcDateTime(booking.created_at)}
                  </p>
                </article>
              ))}
            </div>
          )}
        </div>
      ) : null}
    </section>
  );
}
