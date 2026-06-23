export interface BookingRule {
  id: number;
  resource_type_id: number;
  min_duration_minutes: number;
  max_duration_minutes: number;
  max_active_bookings_per_user: number;
  requires_approval: boolean;
  booking_horizon_days: number;
  workday_start: string;
  workday_end: string;
  unrestricted_time: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface BookingRulePayload {
  resource_type_id: number;
  min_duration_minutes: number;
  max_duration_minutes: number;
  max_active_bookings_per_user: number;
  requires_approval: boolean;
  booking_horizon_days: number;
  workday_start: string;
  workday_end: string;
  unrestricted_time: boolean;
  is_active: boolean;
}
