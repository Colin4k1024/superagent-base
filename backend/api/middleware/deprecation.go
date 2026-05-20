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
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

// DeprecationMW adds Deprecation and Sunset headers to responses for deprecated API paths.
// successorBase is the base path of the successor version (e.g. "/api/v2").
// The middleware rewrites the current path prefix "/api/v1" to successorBase in the Link header.
func DeprecationMW(successorBase string) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		ctx.Next(c)
		path := string(ctx.Request.URI().Path())
		ctx.Response.Header.Set("Deprecation", "true")
		ctx.Response.Header.Set("Sunset", "2026-12-31T00:00:00Z")
		// Construct successor URL by replacing /api/v1 prefix with successorBase.
		successorPath := successorBase + strings.TrimPrefix(path, "/api/v1")
		ctx.Response.Header.Set("Link", "<"+successorPath+">;rel=\"successor-version\"")
	}
}
