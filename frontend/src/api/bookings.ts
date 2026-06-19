import type { Booking, CreateBookingPayload } from "../types/bookings";
import { apiRequest } from "./client";

export function listMyBookings(): Promise<Booking[]> {
  return apiRequest<Booking[]>("/my/bookings");
}

export function cancelBooking(id: number): Promise<Booking> {
  return apiRequest<Booking>(`/bookings/${id}/cancel`, {
    method: "POST",
  });
}

export function completeBooking(id: number): Promise<Booking> {
  return apiRequest<Booking>(`/bookings/${id}/complete`, {
    method: "POST",
  });
}

export function createBooking(payload: CreateBookingPayload): Promise<Booking> {
  return apiRequest<Booking>("/bookings", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}
