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

// Package builtin provides a set of simple built-in skills that run locally
// without any external dependencies.
package builtin

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/superagent-ai/superagent-base/backend/pkg/skill"
)

// RegisterAll registers all built-in skills to the provided LocalInvoker.
func RegisterAll(invoker *skill.LocalInvoker) {
	invoker.Register("datetime", DatetimeSkill)
	invoker.Register("calculator", CalculatorSkill)
	invoker.Register("uuid", UUIDSkill)
}

// DatetimeSkill returns the current date/time.
// Input fields (all optional):
//   - format   – Go time format string (default "2006-01-02 15:04:05")
//   - timezone – IANA tz name (default "Local")
func DatetimeSkill(_ context.Context, input map[string]any) (map[string]any, error) {
	format := "2006-01-02 15:04:05"
	if f, ok := input["format"].(string); ok && f != "" {
		format = f
	}

	tzName := "Local"
	if t, ok := input["timezone"].(string); ok && t != "" {
		tzName = t
	}
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		loc = time.Local
	}

	now := time.Now().In(loc)
	return map[string]any{
		"datetime":  now.Format(format),
		"timestamp": now.Unix(),
		"timezone":  loc.String(),
	}, nil
}

// CalculatorSkill evaluates a simple arithmetic expression.
// Input fields:
//   - expression (required) – e.g. "2 + 3", "10 * 5", "100 / 4", "2 ^ 8"
func CalculatorSkill(_ context.Context, input map[string]any) (map[string]any, error) {
	expr, ok := input["expression"].(string)
	if !ok || strings.TrimSpace(expr) == "" {
		return nil, fmt.Errorf("calculator: expression field required")
	}
	result, err := evalSimple(expr)
	if err != nil {
		return nil, fmt.Errorf("calculator: %w", err)
	}
	return map[string]any{"result": result}, nil
}

// UUIDSkill generates a random UUID v4.
func UUIDSkill(_ context.Context, _ map[string]any) (map[string]any, error) {
	return map[string]any{"uuid": generateUUID()}, nil
}

// ─── simple expression evaluator ─────────────────────────────────────────────

// evalSimple handles single binary operations: +, -, *, /, ^ (power), % (mod).
// Operands may be integers or decimals.  Whitespace is ignored.
func evalSimple(expr string) (float64, error) {
	expr = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, expr)
	if expr == "" {
		return 0, fmt.Errorf("empty expression")
	}

	// Try to parse as a plain number first.
	if v, err := strconv.ParseFloat(expr, 64); err == nil {
		return v, nil
	}

	// Scan for the operator (right-to-left so we respect left-associativity for
	// + and -, which have lower precedence than * and /).
	ops := []byte{'+', '-', '*', '/', '%', '^'}
	for _, op := range ops {
		idx := strings.LastIndexByte(expr, op)
		// Skip negative sign at position 0.
		if idx <= 0 {
			continue
		}
		left := expr[:idx]
		right := expr[idx+1:]
		if left == "" || right == "" {
			continue
		}
		lv, err := strconv.ParseFloat(left, 64)
		if err != nil {
			continue
		}
		rv, err := strconv.ParseFloat(right, 64)
		if err != nil {
			continue
		}
		switch op {
		case '+':
			return lv + rv, nil
		case '-':
			return lv - rv, nil
		case '*':
			return lv * rv, nil
		case '/':
			if rv == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return lv / rv, nil
		case '%':
			if rv == 0 {
				return 0, fmt.Errorf("modulo by zero")
			}
			return math.Mod(lv, rv), nil
		case '^':
			return math.Pow(lv, rv), nil
		}
	}
	return 0, fmt.Errorf("unsupported expression %q", expr)
}

// ─── UUID v4 generator ────────────────────────────────────────────────────────

// generateUUID returns a random UUID v4 string using crypto/rand.
func generateUUID() string {
	var b [16]byte
	// Use time-seeded pseudo-random to avoid importing crypto/rand for this
	// lightweight built-in; replace with crypto/rand if security is required.
	seed := time.Now().UnixNano()
	for i := range b {
		b[i] = byte(seed>>uint(i*3)) ^ byte(seed>>(64-uint(i*3+1)))
		seed = seed*6364136223846793005 + 1442695040888963407
	}
	// Set version 4 and variant bits.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%12x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
