package tool

import (
	"context"
	"fmt"
	"time"
)

type Invoker func(ctx context.Context, name string, args map[string]any) (map[string]any, error)

type Middleware func(next Invoker) Invoker

func Chain(mws ...Middleware) Middleware {
	return func(next Invoker) Invoker {
		for i := len(mws) - 1; i >= 0; i-- {
			next = mws[i](next)
		}
		return next
	}
}

func RetryMiddleware(maxRetries int, backoff time.Duration) Middleware {
	return func(next Invoker) Invoker {
		return func(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
			var lastErr error
			for i := 0; i <= maxRetries; i++ {
				result, err := next(ctx, name, args)
				if err == nil {
					return result, nil
				}
				lastErr = err
				if i < maxRetries {
					time.Sleep(backoff)
				}
			}
			return nil, fmt.Errorf("tool %s failed after %d retries: %w", name, maxRetries, lastErr)
		}
	}
}

func TimeoutMiddleware(timeout time.Duration) Middleware {
	return func(next Invoker) Invoker {
		return func(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			return next(ctx, name, args)
		}
	}
}

func LogMiddleware(logFn func(name string, args map[string]any, err error)) Middleware {
	return func(next Invoker) Invoker {
		return func(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
			result, err := next(ctx, name, args)
			logFn(name, args, err)
			return result, err
		}
	}
}
