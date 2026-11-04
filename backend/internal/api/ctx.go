package api

import "context"

func contextWith[T any](ctx context.Context, key ctxKey, val T) context.Context {
	return context.WithValue(ctx, key, val)
}
