import type { BookingRule, BookingRulePayload } from "../types/bookingRules";
import { apiRequest } from "./client";

export function listBookingRules(): Promise<BookingRule[]> {
  return apiRequest<BookingRule[]>("/booking-rules");
}

export function createBookingRule(payload: BookingRulePayload): Promise<BookingRule> {
  return apiRequest<BookingRule>("/booking-rules", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function updateBookingRule(id: number, payload: BookingRulePayload): Promise<BookingRule> {
  return apiRequest<BookingRule>(`/booking-rules/${id}`, {
    method: "PUT",
    body: JSON.stringify(payload),
  });
}
