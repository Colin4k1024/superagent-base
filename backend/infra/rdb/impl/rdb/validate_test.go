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

package rdb

import (
	"strings"
	"testing"
)

func TestValidateIdentifier(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "users", false},
		{"valid underscore", "_internal_table", false},
		{"valid with dots", "schema.table", false},
		{"valid with digits", "table_123", false},
		{"valid mixed", "myTable_v2", false},
		{"empty", "", true},
		{"starts with digit", "1table", true},
		{"contains space", "my table", true},
		{"contains semicolon", "table;DROP", true},
		{"contains quote", "table'name", true},
		{"contains dash", "my-table", true},
		{"contains backtick", "table`name", true},
		{"contains backslash", "table\\name", true},
		{"consecutive dots", "schema..table", true},
		{"too long", strings.Repeat("a", 65), true},
		{"exactly 64", strings.Repeat("a", 64), false},
		{"just underscore", "_", false},
		{"just letter", "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIdentifier(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIdentifier(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateFieldList(t *testing.T) {
	tests := []struct {
		name    string
		fields  []string
		wantErr bool
	}{
		{"valid fields", []string{"id", "name", "email"}, false},
		{"empty list", []string{}, true},
		{"nil list", nil, true},
		{"one invalid", []string{"id", "bad;field"}, true},
		{"all valid with dots", []string{"t.id", "t.name"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFieldList(tt.fields)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFieldList(%v) error = %v, wantErr %v", tt.fields, err, tt.wantErr)
			}
		})
	}
}

func TestSafeIdentifier(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"users", "`users`"},
		{"my_table", "`my_table`"},
		{"table`injection", "`table``injection`"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SafeIdentifier(tt.input)
			if got != tt.want {
				t.Errorf("SafeIdentifier(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
