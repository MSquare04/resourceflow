package service

import "context"

type requestIDContextKey struct{}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	if requestID == "" {
		return ctx
	}
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

func requestIDFromContext(ctx context.Context) string {
	value := ctx.Value(requestIDContextKey{})
	id, ok := value.(string)
	if !ok {
		return ""
	}
	return id
}
