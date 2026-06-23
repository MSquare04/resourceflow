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

export interface CreateBookingPayload {
  resource_id: number;
  start_at: string;
  end_at: string;
  purpose: string | null;
}

export interface BatchBookingPayload {
  resource_id: number;
  dates: string[];
  start_time: string;
  end_time: string;
  purpose: string | null;
}

export interface BatchBookingPreviewItem {
  date: string;
  valid: boolean;
  error_code?: string;
  status?: BookingStatus;
}

export interface BatchBookingPreviewResponse {
  can_create: boolean;
  items: BatchBookingPreviewItem[];
}

export interface BatchBookingCreateResponse {
  created_count: number;
  items: Booking[];
}
