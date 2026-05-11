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
	"os"
	"strings"
)

const (
	envOtelEndpoint   = "OTEL_ENDPOINT"
	envOtelEnabled    = "OTEL_ENABLED"
	envServiceName    = "SERVICE_NAME"
	defaultEndpoint   = "otel-collector:4317"
	defaultSvcName    = "superagent-base"
)

// LoadConfigFromEnv reads tracer configuration from environment variables.
//
// Environment variables:
//   - SERVICE_NAME    — service name reported to the OTel collector (default: "superagent-base")
//   - OTEL_ENDPOINT   — gRPC endpoint of the OTel collector (default: "otel-collector:4317")
//   - OTEL_ENABLED    — set to "true" or "1" to enable tracing (default: disabled)
func LoadConfigFromEnv() TracerConfig {
	return TracerConfig{
		ServiceName: envOr(envServiceName, defaultSvcName),
		Endpoint:    envOr(envOtelEndpoint, defaultEndpoint),
		Enabled:     isTrue(os.Getenv(envOtelEnabled)),
	}
}

// envOr returns the env variable value or the provided default.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// isTrue returns true for "true" or "1" (case-insensitive).
func isTrue(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return s == "true" || s == "1"
}
