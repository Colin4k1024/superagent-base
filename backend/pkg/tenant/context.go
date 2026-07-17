/*
 * Copyright 2025 superagent-ai Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package tenant provides tenant/space isolation primitives.
//
// Every request that touches tenant-scoped data MUST carry a tenant ID
// in the context. This package provides the canonical way to set and
// retrieve it.
//
// Usage:
//
//	ctx = tenant.WithSpaceID(ctx, 42)
//	sid := tenant.MustSpaceID(ctx) // panics if missing
//	sid, ok := tenant.SpaceID(ctx) // safe version
package tenant

import (
	"context"
	"fmt"
)

type contextKey struct{}

// WithSpaceID attaches a space/tenant ID to the context.
func WithSpaceID(ctx context.Context, spaceID int64) context.Context {
	return context.WithValue(ctx, contextKey{}, spaceID)
}

// SpaceID retrieves the space/tenant ID from context.
// Returns (0, false) if not set.
func SpaceID(ctx context.Context) (int64, bool) {
	v, ok := ctx.Value(contextKey{}).(int64)
	if !ok || v == 0 {
		return 0, false
	}
	return v, true
}

// MustSpaceID retrieves the space/tenant ID from context.
// Panics if not set — use only in code paths where a tenant must be present.
func MustSpaceID(ctx context.Context) int64 {
	sid, ok := SpaceID(ctx)
	if !ok {
		panic("tenant: space_id not found in context; ensure middleware sets it")
	}
	return sid
}

// SpaceIDOrDefault retrieves the space/tenant ID from context,
// returning the given default if not set.
func SpaceIDOrDefault(ctx context.Context, defaultID int64) int64 {
	if sid, ok := SpaceID(ctx); ok {
		return sid
	}
	return defaultID
}

// RequireSpaceID is a helper for service methods that need a tenant ID.
// Returns the space ID or an error if not present in context.
func RequireSpaceID(ctx context.Context) (int64, error) {
	sid, ok := SpaceID(ctx)
	if !ok {
		return 0, fmt.Errorf("tenant: space_id required but not found in context")
	}
	return sid, nil
}

// WithUserID attaches a user ID to the context.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

// UserID retrieves the user ID from context.
func UserID(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIDKey{}).(string)
	return v, ok && v != ""
}

type userIDKey struct{}
