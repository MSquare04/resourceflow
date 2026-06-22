package service

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

type BookingExpirationWorker struct {
	bookings *BookingService
	interval time.Duration
}

func NewBookingExpirationWorker(bookings *BookingService, interval time.Duration) *BookingExpirationWorker {
	return &BookingExpirationWorker{
		bookings: bookings,
		interval: interval,
	}
}

func (w *BookingExpirationWorker) Start(ctx context.Context) {
	go w.run(ctx)
}

func (w *BookingExpirationWorker) run(ctx context.Context) {
	w.runOnce(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

func (w *BookingExpirationWorker) runOnce(ctx context.Context) {
	_, err := w.bookings.ProcessExpiredBookings(ctx, time.Now().UTC())
	if err == nil || errors.Is(err, context.Canceled) {
		return
	}

	slog.Default().Error("expired bookings processing failed", "error", err)
}
