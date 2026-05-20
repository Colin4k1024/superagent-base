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
	"crypto/md5"  //nolint:gosec // MD5 is intentional for the hash skill
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
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
	invoker.Register("text-transform", TextTransformSkill)
	invoker.Register("json-format", JSONFormatSkill)
	invoker.Register("url-parse", URLParseSkill)
	invoker.Register("hash", HashSkill)
	invoker.Register("base64", Base64Skill)
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

// toTitleCase uppercases the first letter of each word.
func toTitleCase(s string) string {
	upper := true
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			upper = true
			return r
		}
		if upper {
			upper = false
			return unicode.ToUpper(r)
		}
		return r
	}, s)
}

// generateUUID returns a random UUID v4 string using crypto/rand.
func generateUUID() string {
	var b [16]byte
	if _, err := cryptorand.Read(b[:]); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ─── text-transform ───────────────────────────────────────────────────────────

// TextTransformSkill performs common text transformations.
// Input fields:
//   - text      (required) – input text
//   - operation (required) – uppercase | lowercase | titlecase | reverse | trim | word_count | char_count | truncate
//   - max_length (optional) – used when operation=truncate (default 100)
func TextTransformSkill(_ context.Context, input map[string]any) (map[string]any, error) {
	text, ok := input["text"].(string)
	if !ok {
		return nil, fmt.Errorf("text-transform: 'text' field required")
	}
	op, _ := input["operation"].(string)

	switch op {
	case "uppercase":
		return map[string]any{"result": strings.ToUpper(text)}, nil
	case "lowercase":
		return map[string]any{"result": strings.ToLower(text)}, nil
	case "titlecase":
		return map[string]any{"result": toTitleCase(text)}, nil
	case "reverse":
		r := []rune(text)
		for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
			r[i], r[j] = r[j], r[i]
		}
		return map[string]any{"result": string(r)}, nil
	case "trim":
		return map[string]any{"result": strings.TrimSpace(text)}, nil
	case "word_count":
		return map[string]any{"result": text, "count": len(strings.Fields(text))}, nil
	case "char_count":
		return map[string]any{"result": text, "count": len([]rune(text))}, nil
	case "truncate":
		maxLen := 100
		if v, ok := input["max_length"].(float64); ok && v > 0 {
			maxLen = int(v)
		}
		runes := []rune(text)
		if len(runes) <= maxLen {
			return map[string]any{"result": text}, nil
		}
		return map[string]any{"result": string(runes[:maxLen]) + "..."}, nil
	default:
		return nil, fmt.Errorf("text-transform: unknown operation %q", op)
	}
}

// ─── json-format ──────────────────────────────────────────────────────────────

// JSONFormatSkill formats, validates, minifies, or lists keys of a JSON string.
// Input fields:
//   - json      (required) – raw JSON string
//   - operation (required) – pretty | minify | validate | keys
//   - indent    (optional) – spaces per level for pretty (default 2)
func JSONFormatSkill(_ context.Context, input map[string]any) (map[string]any, error) {
	raw, ok := input["json"].(string)
	if !ok {
		return nil, fmt.Errorf("json-format: 'json' field required")
	}
	op, _ := input["operation"].(string)

	var parsed any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		if op == "validate" {
			return map[string]any{"valid": false, "error": err.Error()}, nil
		}
		return nil, fmt.Errorf("json-format: invalid JSON: %w", err)
	}

	switch op {
	case "validate":
		return map[string]any{"valid": true}, nil
	case "minify":
		b, _ := json.Marshal(parsed)
		return map[string]any{"result": string(b)}, nil
	case "pretty":
		indent := "  "
		if v, ok := input["indent"].(float64); ok && v > 0 {
			indent = strings.Repeat(" ", int(v))
		}
		b, _ := json.MarshalIndent(parsed, "", indent)
		return map[string]any{"result": string(b)}, nil
	case "keys":
		m, ok := parsed.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("json-format: keys operation requires a JSON object")
		}
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		return map[string]any{"keys": keys}, nil
	default:
		return nil, fmt.Errorf("json-format: unknown operation %q", op)
	}
}

// ─── url-parse ────────────────────────────────────────────────────────────────

// URLParseSkill parses a URL and returns its components.
// Input fields:
//   - url (required) – URL string to parse
func URLParseSkill(_ context.Context, input map[string]any) (map[string]any, error) {
	rawURL, ok := input["url"].(string)
	if !ok || rawURL == "" {
		return nil, fmt.Errorf("url-parse: 'url' field required")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return map[string]any{"valid": false, "error": err.Error()}, nil
	}

	queryMap := make(map[string]any)
	for k, vals := range u.Query() {
		if len(vals) == 1 {
			queryMap[k] = vals[0]
		} else {
			queryMap[k] = vals
		}
	}

	return map[string]any{
		"scheme":   u.Scheme,
		"host":     u.Host,
		"hostname": u.Hostname(),
		"port":     u.Port(),
		"path":     u.Path,
		"query":    queryMap,
		"fragment": u.Fragment,
		"valid":    true,
	}, nil
}

// ─── hash ─────────────────────────────────────────────────────────────────────

// HashSkill computes MD5, SHA256, or SHA512 of an input string.
// Input fields:
//   - text      (required) – string to hash
//   - algorithm (optional) – md5 | sha256 | sha512 (default sha256)
func HashSkill(_ context.Context, input map[string]any) (map[string]any, error) {
	text, ok := input["text"].(string)
	if !ok {
		return nil, fmt.Errorf("hash: 'text' field required")
	}
	algo := "sha256"
	if a, ok := input["algorithm"].(string); ok && a != "" {
		algo = strings.ToLower(a)
	}

	data := []byte(text)
	var digest string
	switch algo {
	case "md5":
		sum := md5.Sum(data) //nolint:gosec
		digest = hex.EncodeToString(sum[:])
	case "sha256":
		sum := sha256.Sum256(data)
		digest = hex.EncodeToString(sum[:])
	case "sha512":
		sum := sha512.Sum512(data)
		digest = hex.EncodeToString(sum[:])
	default:
		return nil, fmt.Errorf("hash: unknown algorithm %q (use md5, sha256, sha512)", algo)
	}

	return map[string]any{
		"hash":         digest,
		"algorithm":    algo,
		"input_length": len(data),
	}, nil
}

// ─── base64 ───────────────────────────────────────────────────────────────────

// Base64Skill encodes or decodes a string using standard or URL-safe Base64.
// Input fields:
//   - text      (required) – text to encode, or Base64 to decode
//   - operation (required) – encode | decode
//   - url_safe  (optional) – use URL-safe alphabet (default false)
func Base64Skill(_ context.Context, input map[string]any) (map[string]any, error) {
	text, ok := input["text"].(string)
	if !ok {
		return nil, fmt.Errorf("base64: 'text' field required")
	}
	op, _ := input["operation"].(string)
	urlSafe, _ := input["url_safe"].(bool)

	enc := base64.StdEncoding
	if urlSafe {
		enc = base64.URLEncoding
	}

	switch op {
	case "encode":
		return map[string]any{"result": enc.EncodeToString([]byte(text))}, nil
	case "decode":
		b, err := enc.DecodeString(text)
		if err != nil {
			return map[string]any{"valid": false, "result": "", "error": err.Error()}, nil
		}
		return map[string]any{"result": string(b), "valid": true}, nil
	default:
		return nil, fmt.Errorf("base64: operation must be 'encode' or 'decode'")
	}
}
