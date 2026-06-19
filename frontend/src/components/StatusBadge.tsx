import { useTranslation } from "react-i18next";

import type { BookingStatus } from "../types/bookings";

interface StatusBadgeProps {
  status: BookingStatus;
}

const statusClassNames: Record<BookingStatus, string> = {
  pending: "badge-warning",
  confirmed: "badge-info",
  rejected: "badge-danger",
  cancelled: "badge-muted",
  completed: "badge-success",
};

export function StatusBadge({ status }: StatusBadgeProps): JSX.Element {
  const { t } = useTranslation();

  return <span className={`badge ${statusClassNames[status]}`}>{t(`statuses.${status}`)}</span>;
}
