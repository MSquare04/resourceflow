import type {
  BatchBookingCreateResponse,
  BatchBookingPayload,
  BatchBookingPreviewResponse,
  Booking,
  CreateBookingPayload,
} from "../types/bookings";
import { apiRequest } from "./client";

export function listMyBookings(): Promise<Booking[]> {
  return apiRequest<Booking[]>("/my/bookings");
}

export function listBookings(): Promise<Booking[]> {
  return apiRequest<Booking[]>("/bookings");
}

export function cancelBooking(id: number): Promise<Booking> {
  return apiRequest<Booking>(`/bookings/${id}/cancel`, {
    method: "POST",
  });
}

export function approveBooking(id: number): Promise<Booking> {
  return apiRequest<Booking>(`/bookings/${id}/approve`, {
    method: "POST",
  });
}

export function rejectBooking(id: number): Promise<Booking> {
  return apiRequest<Booking>(`/bookings/${id}/reject`, {
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

export function previewBatchBookings(payload: BatchBookingPayload): Promise<BatchBookingPreviewResponse> {
  return apiRequest<BatchBookingPreviewResponse>("/bookings/batch/preview", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function createBatchBookings(payload: BatchBookingPayload): Promise<BatchBookingCreateResponse> {
  return apiRequest<BatchBookingCreateResponse>("/bookings/batch", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}
