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

package observe

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

// metricsFormat is the Prometheus text exposition format.
var metricsFormat = expfmt.NewFormat(expfmt.TypeTextPlain)

// MetricsHandler is a Hertz-native handler that serves Prometheus metrics
// in the standard text exposition format.
//
// Mount at GET /metrics on the Hertz server:
//
//	s.GET("/metrics", observe.MetricsHandler)
func MetricsHandler(_ context.Context, c *app.RequestContext) {
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		c.String(500, "failed to gather metrics: "+err.Error())
		return
	}

	c.Response.Header.Set("Content-Type", string(metricsFormat))

	enc := expfmt.NewEncoder(c, metricsFormat)
	for _, mf := range mfs {
		if err := enc.Encode(mf); err != nil {
			// Best-effort: partial output is acceptable for scraping.
			return
		}
	}
}
