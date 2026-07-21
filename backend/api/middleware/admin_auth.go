/*
 * Copyright 2025 coze-dev Authors
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

package middleware

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/superagent-ai/superagent-base/backend/pkg/logs"
	"github.com/superagent-ai/superagent-base/backend/pkg/rbac"
)

// adminAuthOnce guards the startup log so it prints exactly once.
var adminAuthOnce sync.Once

// isDevMode returns true when the process is running in a development or debug
// context where an unset ADMIN_API_KEY should be tolerated rather than rejected.
func isDevMode() bool {
	appEnv := strings.ToLower(os.Getenv("APP_ENV"))
	logLevel := strings.ToLower(os.Getenv("LOG_LEVEL"))
	return appEnv == "dev" || appEnv == "development" || appEnv == "local" || logLevel == "debug"
}

// ---------------------------------------------------------------------------
// IP rate limiter — sliding-window, 30 req/min per source IP.
// ---------------------------------------------------------------------------

const (
	adminRateLimit   = 30
	adminRateWindow  = time.Minute
	cleanupInterval  = 5 * time.Minute
)

type ipWindow struct {
	mu         sync.Mutex
	timestamps []time.Time
}

var (
	adminRateMu   sync.Mutex
	adminRateMap  = make(map[string]*ipWindow)
	cleanupOnce   sync.Once
)

func startRateCleanup() {
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			adminRateMu.Lock()
			for ip, w := range adminRateMap {
				w.mu.Lock()
				valid := w.timestamps[:0]
				for _, ts := range w.timestamps {
					if now.Sub(ts) < adminRateWindow {
						valid = append(valid, ts)
					}
				}
				if len(valid) == 0 {
					delete(adminRateMap, ip)
				} else {
					w.timestamps = valid
				}
				w.mu.Unlock()
			}
			adminRateMu.Unlock()
		}
	}()
}

// allowRequest returns true if the IP is within the rate limit, false otherwise.
func allowRequest(ip string) bool {
	cleanupOnce.Do(startRateCleanup)

	adminRateMu.Lock()
	w, ok := adminRateMap[ip]
	if !ok {
		w = &ipWindow{}
		adminRateMap[ip] = w
	}
	adminRateMu.Unlock()

	now := time.Now()
	cutoff := now.Add(-adminRateWindow)

	w.mu.Lock()
	defer w.mu.Unlock()

	// Evict expired timestamps (sliding window).
	valid := w.timestamps[:0]
	for _, ts := range w.timestamps {
		if ts.After(cutoff) {
			valid = append(valid, ts)
		}
	}
	w.timestamps = valid

	if len(w.timestamps) >= adminRateLimit {
		return false
	}
	w.timestamps = append(w.timestamps, now)
	return true
}

// ---------------------------------------------------------------------------
// AdminAuthMW is a Hertz middleware that enforces admin authentication and
// IP-based rate limiting for all /api/v1/admin/* routes.
//
// Auth rules:
//   - ADMIN_API_KEY set → require matching X-Admin-Key or Authorization: Bearer <key>
//   - ADMIN_API_KEY empty + dev/debug mode → allow (log once at startup)
//   - ADMIN_API_KEY empty + non-dev mode → reject with 403
// ---------------------------------------------------------------------------

// APIKeyAdminAuthMW returns a Hertz handler that enforces admin API key
// authentication and IP rate limiting for /api/v1/admin/* routes.
//
// store may be nil — if non-nil, it is consulted after the legacy single-key
// check, enabling per-user API keys with role-based access control.
//
// Auth rules:
//   - Key matches ADMIN_API_KEY → attach synthetic admin user to ctx
//   - Key found in store → attach that user to ctx
//   - Key matches neither → 403
//   - No key configured + dev mode → pass-through with synthetic admin user
//   - No key configured + non-dev mode → 403
func APIKeyAdminAuthMW(store rbac.UserStorer) app.HandlerFunc {
	apiKey := os.Getenv("ADMIN_API_KEY")
	dev := isDevMode()

	// Emit a single startup log describing the security posture.
	adminAuthOnce.Do(func() {
		switch {
		case apiKey != "":
			logs.Infof("admin endpoints: authentication enabled (ADMIN_API_KEY is set)")
		case dev:
			logs.Infof("admin endpoints: dev/debug mode — authentication bypassed (set ADMIN_API_KEY for production)")
		default:
			logs.Warnf("admin endpoints disabled: set ADMIN_API_KEY (or APP_ENV=dev for local development)")
		}
	})

	return func(ctx context.Context, c *app.RequestContext) {
		// Rate limit first — applies regardless of auth result.
		ip := c.ClientIP()
		if !allowRequest(ip) {
			c.JSON(429, map[string]any{"code": 429, "msg": "too many requests"})
			c.Abort()
			return
		}

		// --- Authentication ---
		if apiKey != "" {
			// Extract the presented key from X-Admin-Key or Authorization: Bearer <key>.
			presented := string(c.GetHeader("X-Admin-Key"))
			if presented == "" {
				authHeader := string(c.GetHeader("Authorization"))
				if strings.HasPrefix(authHeader, "Bearer ") {
					presented = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}

			// First: check multi-user store (supports per-user keys with roles).
			if store != nil && presented != "" {
				if u, ok := store.LookupByAPIKey(presented); ok {
					ctx = rbac.WithUser(ctx, u)
					c.Next(ctx)
					return
				}
			}

			// Second: check legacy single admin key (backward compatible).
			if presented == apiKey {
				adminUser := &rbac.User{
					ID:   "admin",
					Name: "admin",
					Role: rbac.RoleAdmin,
				}
				ctx = rbac.WithUser(ctx, adminUser)
				c.Next(ctx)
				return
			}

			c.JSON(403, map[string]any{"code": 403, "msg": "forbidden: invalid or missing admin API key"})
			c.Abort()
			return
		}

		// Key is not configured.
		if dev {
			// Dev/debug pass-through with synthetic admin user.
			adminUser := &rbac.User{
				ID:   "admin",
				Name: "admin",
				Role: rbac.RoleAdmin,
			}
			ctx = rbac.WithUser(ctx, adminUser)
			c.Next(ctx)
			return
		}

		// Non-dev, no key — reject.
		logs.Warnf("admin request rejected from %s: ADMIN_API_KEY not set in non-dev environment", ip)
		c.JSON(403, map[string]any{"code": 403, "msg": "admin endpoints disabled: set ADMIN_API_KEY"})
		c.Abort()
	}
}
