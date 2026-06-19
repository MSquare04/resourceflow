export type BookingStatus = "pending" | "confirmed" | "rejected" | "cancelled" | "completed";

export interface Booking {
  id: number;
  resource_id: number;
  resource_name: string;
  user_id: number;
  user_full_name: string | null;
  start_at: string;
  end_at: string;
  purpose: string | null;
  status: BookingStatus;
  approved_by_user_id: number | null;
  approved_at: string | null;
  cancelled_at: string | null;
  completed_at: string | null;
  created_at: string;
  updated_at: string;
}
