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
	cryptorand "crypto/rand"
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
	invoker.Register("find-skills", FindSkillsSkill)
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

// evalSimple evaluates arithmetic expressions with correct operator precedence.
// Supports: +, -, *, /, % (mod), ^ (power). Operands are integers or decimals.
// Precedence (low to high): +- , */% , ^
// Associativity: left-to-right. Does NOT support parentheses.
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

	// Try to parse as a plain number first (handles negatives like "-3.14").
	if v, err := strconv.ParseFloat(expr, 64); err == nil {
		return v, nil
	}

	// Precedence level 1 (lowest): + and -
	// Scan from right to left for left-associativity.
	if idx := findLastOp(expr, '+', '-'); idx > 0 {
		lv, lerr := evalSimple(expr[:idx])
		rv, rerr := evalSimple(expr[idx+1:])
		if lerr == nil && rerr == nil {
			if expr[idx] == '+' {
				return lv + rv, nil
			}
			return lv - rv, nil
		}
	}

	// Precedence level 2: *, /, %
	if idx := findLastOp(expr, '*', '/', '%'); idx > 0 {
		lv, lerr := evalSimple(expr[:idx])
		rv, rerr := evalSimple(expr[idx+1:])
		if lerr == nil && rerr == nil {
			switch expr[idx] {
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
			}
		}
	}

	// Precedence level 3 (highest): ^
	if idx := findLastOp(expr, '^'); idx > 0 {
		lv, lerr := evalSimple(expr[:idx])
		rv, rerr := evalSimple(expr[idx+1:])
		if lerr == nil && rerr == nil {
			return math.Pow(lv, rv), nil
		}
	}

	return 0, fmt.Errorf("unsupported expression %q", expr)
}

// findLastOp finds the rightmost occurrence of any of the given operators in expr,
// skipping positions where the character could be a unary sign (position 0 or after
// another operator character).
func findLastOp(expr string, ops ...byte) int {
	for i := len(expr) - 1; i > 0; i-- {
		ch := expr[i]
		for _, op := range ops {
			if ch == op {
				// Ensure it's not a unary sign: previous char must be a digit or '.'
				prev := expr[i-1]
				if prev >= '0' && prev <= '9' || prev == '.' {
					return i
				}
			}
		}
	}
	return -1
}

// ─── UUID v4 generator ────────────────────────────────────────────────────────

// generateUUID returns a random UUID v4 string using crypto/rand.
func generateUUID() string {
	var b [16]byte
	if _, err := cryptorand.Read(b[:]); err != nil {
		// Fallback should never happen in practice; panic is appropriate for
		// a broken random source since it compromises session security.
		panic("crypto/rand unavailable: " + err.Error())
	}
	// Set version 4 and variant bits.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
