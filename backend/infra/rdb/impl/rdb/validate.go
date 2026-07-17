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

package rdb

import (
	"fmt"
	"regexp"
	"strings"
)

// sqlIdentifierPattern matches valid SQL identifiers:
// - Starts with a letter or underscore
// - Contains only letters, digits, underscores, or dots (for qualified names)
// - Max length 64 characters (MySQL default)
var sqlIdentifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_.]*$`)

const maxIdentifierLength = 64

// ValidateIdentifier validates a single SQL identifier (table name, column name, index name).
// Returns an error if the identifier contains characters that could enable SQL injection.
// Identifiers must match [a-zA-Z_][a-zA-Z0-9_.]* and be <= 64 characters.
func ValidateIdentifier(name string) error {
	if name == "" {
		return fmt.Errorf("sql: empty identifier")
	}
	if len(name) > maxIdentifierLength {
		return fmt.Errorf("sql: identifier %q exceeds max length %d", name, maxIdentifierLength)
	}
	if !sqlIdentifierPattern.MatchString(name) {
		return fmt.Errorf("sql: invalid identifier %q: must match [a-zA-Z_][a-zA-Z0-9_.]*", name)
	}
	// Reject identifiers that are pure dots or contain consecutive dots
	if strings.Contains(name, "..") {
		return fmt.Errorf("sql: invalid identifier %q: contains consecutive dots", name)
	}
	return nil
}

// ValidateIdentifiers validates multiple SQL identifiers at once.
// Returns the first error encountered, or nil if all are valid.
func ValidateIdentifiers(names ...string) error {
	for _, name := range names {
		if err := ValidateIdentifier(name); err != nil {
			return err
		}
	}
	return nil
}

// ValidateFieldList validates a list of field/column names.
// Returns an error if any field name is invalid.
func ValidateFieldList(fields []string) error {
	if len(fields) == 0 {
		return fmt.Errorf("sql: empty field list")
	}
	return ValidateIdentifiers(fields...)
}

// SafeIdentifier wraps a validated identifier in backticks for SQL.
// Callers MUST validate with ValidateIdentifier first.
func SafeIdentifier(name string) string {
	// Double any embedded backticks (defense in depth)
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

// SafeIdentifierList joins validated identifiers with backtick wrapping.
// Callers MUST validate with ValidateFieldList first.
func SafeIdentifierList(fields []string) string {
	safe := make([]string, len(fields))
	for i, f := range fields {
		safe[i] = SafeIdentifier(f)
	}
	return strings.Join(safe, ",")
}
