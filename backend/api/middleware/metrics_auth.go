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

package middleware

import (
	"context"
	"os"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

// MetricsAuthMW protects the /metrics endpoint.
// When METRICS_TOKEN env var is set, the request must include
// "Authorization: Bearer <token>" — otherwise 401 is returned.
// When METRICS_TOKEN is empty the endpoint is open (development default).
func MetricsAuthMW() app.HandlerFunc {
	token := os.Getenv("METRICS_TOKEN")
	return func(_ context.Context, ctx *app.RequestContext) {
		if token == "" {
			ctx.Next(context.Background())
			return
		}
		auth := string(ctx.GetHeader("Authorization"))
		if !strings.HasPrefix(auth, "Bearer ") || strings.TrimPrefix(auth, "Bearer ") != token {
			ctx.JSON(401, map[string]string{"error": "unauthorized: valid METRICS_TOKEN required"})
			ctx.Abort()
			return
		}
		ctx.Next(context.Background())
	}
}
