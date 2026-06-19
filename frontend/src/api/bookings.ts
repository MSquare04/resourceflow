import type { Booking } from "../types/bookings";
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
