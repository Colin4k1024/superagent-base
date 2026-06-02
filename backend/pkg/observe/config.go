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
	"encoding/base64"
	"os"
	"strconv"
	"strings"
)

const (
	envOtelEndpoint   = "OTEL_ENDPOINT"
	envOtelEnabled    = "OTEL_ENABLED"
	envServiceName    = "SERVICE_NAME"
	defaultEndpoint   = "otel-collector:4317"
	defaultSvcName    = "superagent-base"

	envLangfuseEnabled   = "LANGFUSE_ENABLED"
	envLangfusePublicKey = "LANGFUSE_PUBLIC_KEY"
	envLangfuseSecretKey = "LANGFUSE_SECRET_KEY"
	envLangfuseHost      = "LANGFUSE_HOST"
	envLangfuseDebug     = "LANGFUSE_DEBUG"
	envLangfuseSampleRate = "LANGFUSE_SAMPLE_RATE"

	defaultLangfuseHost = "https://cloud.langfuse.com"
)

// LangfuseConfig holds Langfuse-specific OTLP export settings.
type LangfuseConfig struct {
	Enabled    bool
	PublicKey  string
	SecretKey  string
	Host       string // Base URL, e.g. "https://cloud.langfuse.com" or self-hosted
	Debug      bool
	SampleRate float64 // 0.0-1.0, default 1.0 (100%)
}

// OTLPEndpoint returns the Langfuse OTLP endpoint path.
func (c LangfuseConfig) OTLPEndpoint() string {
	host := strings.TrimRight(c.Host, "/")
	return host + "/api/public/otel"
}

// AuthHeader returns the Basic Auth header value for Langfuse.
func (c LangfuseConfig) AuthHeader() string {
	creds := c.PublicKey + ":" + c.SecretKey
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(creds))
}

// LoadConfigFromEnv reads tracer configuration from environment variables.
//
// Environment variables:
//   - SERVICE_NAME       — service name reported to the OTel collector (default: "superagent-base")
//   - OTEL_ENDPOINT      — gRPC endpoint of the OTel collector (default: "otel-collector:4317")
//   - OTEL_ENABLED       — set to "true" or "1" to enable tracing (default: disabled)
func LoadConfigFromEnv() TracerConfig {
	return TracerConfig{
		ServiceName: envOr(envServiceName, defaultSvcName),
		Endpoint:    envOr(envOtelEndpoint, defaultEndpoint),
		Enabled:     isTrue(os.Getenv(envOtelEnabled)),
		Langfuse:    loadLangfuseConfig(),
	}
}

// loadLangfuseConfig reads Langfuse configuration from environment variables.
//
// Environment variables:
//   - LANGFUSE_ENABLED     — set to "true" or "1" to enable Langfuse export
//   - LANGFUSE_PUBLIC_KEY  — Langfuse project public key (pk-lf-...)
//   - LANGFUSE_SECRET_KEY  — Langfuse project secret key (sk-lf-...)
//   - LANGFUSE_HOST        — Langfuse base URL (default: "https://cloud.langfuse.com")
//   - LANGFUSE_DEBUG       — set to "true" for debug logging
//   - LANGFUSE_SAMPLE_RATE — trace sampling rate 0.0-1.0 (default: 1.0)
func loadLangfuseConfig() LangfuseConfig {
	rate := 1.0
	if v := os.Getenv(envLangfuseSampleRate); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed >= 0 && parsed <= 1 {
			rate = parsed
		}
	}

	return LangfuseConfig{
		Enabled:    isTrue(os.Getenv(envLangfuseEnabled)),
		PublicKey:  os.Getenv(envLangfusePublicKey),
		SecretKey:  os.Getenv(envLangfuseSecretKey),
		Host:       envOr(envLangfuseHost, defaultLangfuseHost),
		Debug:      isTrue(os.Getenv(envLangfuseDebug)),
		SampleRate: rate,
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
