import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

import { cancelBooking, listMyBookings } from "../api/bookings";
import { ApiError } from "../api/client";
import { EmptyState } from "../components/EmptyState";
import { ErrorState } from "../components/ErrorState";
import { LoadingState } from "../components/LoadingState";
import { PageHeader } from "../components/PageHeader";
import { StatusBadge } from "../components/StatusBadge";
import type { Booking, BookingStatus } from "../types/bookings";
import { formatUtcDateTime } from "../utils/datetime";

type BookingAction = "cancel";
type StatusFilter = "all" | BookingStatus;

interface BookingFilters {
  search: string;
  status: StatusFilter;
}

const defaultFilters: BookingFilters = {
  search: "",
  status: "all",
};

function canCancelBooking(status: BookingStatus, endAt: string): boolean {
  return (status === "pending" || status === "confirmed") && new Date(endAt).getTime() > Date.now();
}

function mapActionError(error: ApiError, t: ReturnType<typeof useTranslation>["t"]): string {
  switch (error.code) {
    case "booking_forbidden":
      return t("pages.myBookings.actions.errors.forbidden");
    case "booking_cancel_not_allowed":
      return t("pages.myBookings.actions.errors.invalidTransition");
    case "booking_already_ended":
      return t("pages.myBookings.actions.errors.alreadyEnded");
    case "not_found":
      return t("pages.myBookings.actions.errors.notFound");
    default:
      return error.message;
  }
}

export function MyBookingsPage(): JSX.Element {
  const { t } = useTranslation();
  const [bookings, setBookings] = useState<Booking[]>([]);
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
      const data = await listMyBookings();
      setBookings(data);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : t("errors.generic"));
    } finally {
      setLoading(false);
    }
  }

  async function handleBookingAction(booking: Booking): Promise<void> {
    const confirmationMessage = t("pages.myBookings.confirmations.cancelPrompt", {
      resourceName: booking.resource_name,
      defaultValue: `Cancel the booking for "${booking.resource_name}"?`,
    });

    if (!window.confirm(confirmationMessage)) {
      return;
    }

    setActionError(null);
    setPendingActionId(booking.id);
    setPendingActionType("cancel");

    try {
      await cancelBooking(booking.id);
      await loadBookings();
    } catch (actionRequestError) {
      if (actionRequestError instanceof ApiError) {
        setActionError(mapActionError(actionRequestError, t));
      } else if (actionRequestError instanceof Error) {
        setActionError(actionRequestError.message);
      } else {
        setActionError(t("pages.myBookings.actions.errors.generic"));
      }
    } finally {
      setPendingActionId(null);
      setPendingActionType(null);
    }
  }

  const filteredBookings = useMemo(() => {
    const normalizedSearch = filters.search.trim().toLowerCase();
    const now = Date.now();

    return [...bookings]
      .filter((booking) => {
        if (!normalizedSearch) {
          return true;
        }

        const searchableFields = [booking.resource_name, booking.purpose ?? ""];
        return searchableFields.some((value) => value.toLowerCase().includes(normalizedSearch));
      })
      .filter((booking) => (filters.status === "all" ? true : booking.status === filters.status))
      .sort((left, right) => {
        const leftStart = new Date(left.start_at).getTime();
        const rightStart = new Date(right.start_at).getTime();
        const leftIsFuture = leftStart >= now;
        const rightIsFuture = rightStart >= now;

        if (leftIsFuture && !rightIsFuture) {
          return -1;
        }

        if (!leftIsFuture && rightIsFuture) {
          return 1;
        }

        return leftIsFuture ? leftStart - rightStart : rightStart - leftStart;
      });
  }, [bookings, filters]);

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
        title={t("pages.myBookings.title")}
        actions={
          <button type="button" className="btn btn-secondary" onClick={resetFilters} disabled={!hasActiveFilters}>
            {t("pages.myBookings.filters.reset")}
          </button>
        }
      />

      {loading ? <LoadingState message={t("pages.myBookings.loading")} /> : null}
      {!loading && error ? <ErrorState message={error} onRetry={() => void loadBookings()} /> : null}
      {!loading && !error && bookings.length === 0 ? (
        <EmptyState
          title={t("pages.myBookings.empty.title")}
          description={t("pages.myBookings.empty.description")}
        />
      ) : null}

      {!loading && !error && bookings.length > 0 ? (
        <div className="bookings-page">
          <div className="bookings-filters">
            <label className="field">
              <span>{t("pages.myBookings.filters.search")}</span>
              <input
                type="search"
                value={filters.search}
                onChange={(event) => setFilters((current) => ({ ...current, search: event.target.value }))}
                placeholder={t("pages.myBookings.filters.searchPlaceholder")}
              />
            </label>

            <label className="field">
              <span>{t("pages.myBookings.filters.status")}</span>
              <select
                value={filters.status}
                onChange={(event) =>
                  setFilters((current) => ({ ...current, status: event.target.value as StatusFilter }))
                }
              >
                <option value="all">{t("pages.myBookings.filters.allStatuses")}</option>
                <option value="pending">{t("statuses.pending")}</option>
                <option value="confirmed">{t("statuses.confirmed")}</option>
                <option value="rejected">{t("statuses.rejected")}</option>
                <option value="cancelled">{t("statuses.cancelled")}</option>
                <option value="completed">{t("statuses.completed")}</option>
              </select>
            </label>
          </div>

          {actionError ? <p className="error-text">{actionError}</p> : null}

          <p className="resources-summary muted">{t("pages.myBookings.resultsSummary", { count: filteredBookings.length })}</p>

          {filteredBookings.length === 0 ? (
            <EmptyState
              title={t("pages.myBookings.noResults.title")}
              description={t("pages.myBookings.noResults.description")}
            />
          ) : (
            <div className="bookings-list" role="list">
              {filteredBookings.map((booking) => (
                <article key={booking.id} className="booking-card" role="listitem">
                  <div className="booking-card__header">
                    <StatusBadge status={booking.status} />
                  </div>

                  <div className="booking-card__heading">
                    <h3 className="booking-card__title">{booking.resource_name}</h3>
                  </div>

                  <div className="booking-card__time-block">
                    <div className="booking-card__time-item">
                      <span className="booking-card__time-label">{t("pages.myBookings.fields.startAt")}</span>
                      <strong>{formatUtcDateTime(booking.start_at)}</strong>
                    </div>
                    <div className="booking-card__time-item">
                      <span className="booking-card__time-label">{t("pages.myBookings.fields.endAt")}</span>
                      <strong>{formatUtcDateTime(booking.end_at)}</strong>
                    </div>
                  </div>

                  <dl className="booking-card__meta">
                    <div>
                      <dt>{t("pages.myBookings.fields.purpose")}</dt>
                      <dd>{booking.purpose || t("pages.myBookings.noPurpose")}</dd>
                    </div>
                  </dl>

                  <div className="booking-card__actions">
                    {canCancelBooking(booking.status, booking.end_at) ? (
                      <button
                        type="button"
                        className="btn btn-secondary"
                        disabled={pendingActionId === booking.id}
                        onClick={() => void handleBookingAction(booking)}
                      >
                        {isActionPending(booking.id, "cancel")
                          ? t("pages.myBookings.actions.processing")
                          : t("pages.myBookings.actions.cancel")}
                      </button>
                    ) : null}
                  </div>

                  <p className="booking-card__footer-meta muted">
                    {t("pages.myBookings.fields.createdAt")}: {booking.created_at ? formatUtcDateTime(booking.created_at) : t("pages.myBookings.notSpecified")}
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
