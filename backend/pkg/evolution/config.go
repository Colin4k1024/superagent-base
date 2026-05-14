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

package evolution

import (
	"encoding/base64"
	"os"
	"strconv"
)

// Config holds all configuration for the evolution engine.
type Config struct {
	// Enabled is the master switch; if false, Init returns nil without error.
	Enabled bool
	// ExperienceURL is the base URL of the Oris Experience Repo (e.g. http://localhost:8090).
	ExperienceURL string
	// HubURL is the optional Hub service URL for federated multi-node sharing.
	HubURL string
	// NodeEndpoint is the public HTTP endpoint of this superagent-base node
	// reported to the Hub so peer nodes can reach it.
	NodeEndpoint string
	// APIKey is the API key for authenticating with the Experience Repo.
	APIKey string
	// Seed is the 32-byte Ed25519 seed for signing messages. Loaded from base64.
	Seed [32]byte
	// SenderID is the unique identifier for this superagent node.
	SenderID string
	// MinConfidence is the minimum confidence threshold for Advisor recommendations.
	MinConfidence float64
	// MaxSuggestions caps the number of Gene recommendations returned by Advisor.
	MaxSuggestions int
}

// LoadConfigFromEnv reads evolution configuration from environment variables.
//
//	EVOLUTION_ENABLED       (default: false)
//	ORIS_EXPERIENCE_URL     (required when enabled)
//	ORIS_HUB_URL            (optional)
//	ORIS_API_KEY
//	ORIS_SEED               (base64-encoded 32-byte Ed25519 seed)
//	ORIS_SENDER_ID          (default: superagent-node-1)
//	EVOLUTION_MIN_CONFIDENCE (default: 0.5)
//	EVOLUTION_MAX_SUGGESTIONS (default: 3)
func LoadConfigFromEnv() Config {
	c := Config{
		Enabled:        parseBool(os.Getenv("EVOLUTION_ENABLED"), false),
		ExperienceURL:  getenv("ORIS_EXPERIENCE_URL", "http://localhost:8090"),
		HubURL:         os.Getenv("ORIS_HUB_URL"),
		NodeEndpoint:   os.Getenv("ORIS_NODE_ENDPOINT"),
		APIKey:         os.Getenv("ORIS_API_KEY"),
		SenderID:       getenv("ORIS_SENDER_ID", "superagent-node-1"),
		MinConfidence:  parseFloat(os.Getenv("EVOLUTION_MIN_CONFIDENCE"), 0.5),
		MaxSuggestions: parseInt(os.Getenv("EVOLUTION_MAX_SUGGESTIONS"), 3),
	}

	if raw := os.Getenv("ORIS_SEED"); raw != "" {
		b, err := base64.StdEncoding.DecodeString(raw)
		if err == nil && len(b) == 32 {
			copy(c.Seed[:], b)
		}
	}

	return c
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseBool(s string, def bool) bool {
	if s == "" {
		return def
	}
	b, err := strconv.ParseBool(s)
	if err != nil {
		return def
	}
	return b
}

func parseFloat(s string, def float64) float64 {
	if s == "" {
		return def
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return f
}

func parseInt(s string, def int) int {
	if s == "" {
		return def
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return i
}
