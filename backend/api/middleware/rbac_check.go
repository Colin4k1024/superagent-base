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

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/superagent-ai/superagent-base/backend/pkg/rbac"
)

// RequirePermission returns a Hertz middleware that checks RBAC permission.
// Use after APIKeyAdminAuthMW which sets the user in context.
func RequirePermission(resource, action string) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if !rbac.CheckPermission(ctx, resource, action) {
			c.JSON(403, map[string]any{"code": 403, "msg": "insufficient permissions"})
			c.Abort()
			return
		}
		c.Next(ctx)
	}
}
