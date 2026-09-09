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

package compose

import (
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/einobridge"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/superagent-ai/superagent-base/backend/domain/workflow/entity/vo"
)

func TestAddFieldMappingsWithDeduplication(t *testing.T) {
	tests := []struct {
		name              string
		initialCarryOvers map[vo.NodeKey][]*einobridge.FieldMapping
		fromNodeKey       vo.NodeKey
		fieldMappings     []*einobridge.FieldMapping
		expectedCount     int
		description       string
	}{
		{
			name:              "empty_carry_overs",
			initialCarryOvers: make(map[vo.NodeKey][]*einobridge.FieldMapping),
			fromNodeKey:       "node1",
			fieldMappings: []*einobridge.FieldMapping{
				einobridge.MapFieldPaths(einobridge.FieldPath{"input1"}, einobridge.FieldPath{"output1"}),
				einobridge.MapFieldPaths(einobridge.FieldPath{"input2"}, einobridge.FieldPath{"output2"}),
			},
			expectedCount: 2,
			description:   "should add all mappings when carryOvers is empty",
		},
		{
			name: "no_duplicates",
			initialCarryOvers: map[vo.NodeKey][]*einobridge.FieldMapping{
				"node1": {
					einobridge.MapFieldPaths(einobridge.FieldPath{"input1"}, einobridge.FieldPath{"output1"}),
				},
			},
			fromNodeKey: "node1",
			fieldMappings: []*einobridge.FieldMapping{
				einobridge.MapFieldPaths(einobridge.FieldPath{"input2"}, einobridge.FieldPath{"output2"}),
				einobridge.MapFieldPaths(einobridge.FieldPath{"input3"}, einobridge.FieldPath{"output3"}),
			},
			expectedCount: 3,
			description:   "should add new mappings when no duplicates exist",
		},
		{
			name: "with_duplicates",
			initialCarryOvers: map[vo.NodeKey][]*einobridge.FieldMapping{
				"node1": {
					einobridge.MapFieldPaths(einobridge.FieldPath{"input1"}, einobridge.FieldPath{"output1"}),
					einobridge.MapFieldPaths(einobridge.FieldPath{"input2"}, einobridge.FieldPath{"output2"}),
				},
			},
			fromNodeKey: "node1",
			fieldMappings: []*einobridge.FieldMapping{
				einobridge.MapFieldPaths(einobridge.FieldPath{"input1"}, einobridge.FieldPath{"output1"}), // duplicate
				einobridge.MapFieldPaths(einobridge.FieldPath{"input3"}, einobridge.FieldPath{"output3"}), // new
				einobridge.MapFieldPaths(einobridge.FieldPath{"input2"}, einobridge.FieldPath{"output2"}), // duplicate
			},
			expectedCount: 3,
			description:   "should skip duplicates and only add new mappings",
		},
		{
			name: "all_duplicates",
			initialCarryOvers: map[vo.NodeKey][]*einobridge.FieldMapping{
				"node1": {
					einobridge.MapFieldPaths(einobridge.FieldPath{"input1"}, einobridge.FieldPath{"output1"}),
					einobridge.MapFieldPaths(einobridge.FieldPath{"input2"}, einobridge.FieldPath{"output2"}),
				},
			},
			fromNodeKey: "node1",
			fieldMappings: []*einobridge.FieldMapping{
				einobridge.MapFieldPaths(einobridge.FieldPath{"input1"}, einobridge.FieldPath{"output1"}),
				einobridge.MapFieldPaths(einobridge.FieldPath{"input2"}, einobridge.FieldPath{"output2"}),
			},
			expectedCount: 2,
			description:   "should not add any mappings when all are duplicates",
		},
		{
			name: "new_node_key",
			initialCarryOvers: map[vo.NodeKey][]*einobridge.FieldMapping{
				"node1": {
					einobridge.MapFieldPaths(einobridge.FieldPath{"input1"}, einobridge.FieldPath{"output1"}),
				},
			},
			fromNodeKey: "node2",
			fieldMappings: []*einobridge.FieldMapping{
				einobridge.MapFieldPaths(einobridge.FieldPath{"input1"}, einobridge.FieldPath{"output1"}),
				einobridge.MapFieldPaths(einobridge.FieldPath{"input2"}, einobridge.FieldPath{"output2"}),
			},
			expectedCount: 2,
			description:   "should add all mappings for new node key",
		},
		{
			name: "empty_field_mappings",
			initialCarryOvers: map[vo.NodeKey][]*einobridge.FieldMapping{
				"node1": {
					einobridge.MapFieldPaths(einobridge.FieldPath{"input1"}, einobridge.FieldPath{"output1"}),
				},
			},
			fromNodeKey:   "node1",
			fieldMappings: []*einobridge.FieldMapping{},
			expectedCount: 1,
			description:   "should not change carryOvers when fieldMappings is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of initial carryOvers to avoid modifying the test data
			carryOvers := make(map[vo.NodeKey][]*einobridge.FieldMapping)
			for k, v := range tt.initialCarryOvers {
				carryOvers[k] = make([]*einobridge.FieldMapping, len(v))
				copy(carryOvers[k], v)
			}

			// Call the function under test
			addFieldMappingsWithDeduplication(carryOvers, tt.fromNodeKey, tt.fieldMappings)

			// Verify the result
			actualCount := len(carryOvers[tt.fromNodeKey])
			assert.Equal(t, tt.expectedCount, actualCount, tt.description)

			// Verify no duplicates exist in the result
			mappings := carryOvers[tt.fromNodeKey]
			for i := 0; i < len(mappings); i++ {
				for j := i + 1; j < len(mappings); j++ {
					assert.False(t, mappings[i].Equals(mappings[j]),
						"found duplicate mappings at indices %d and %d", i, j)
				}
			}
		})
	}
}

func TestAddFieldMappingsWithDeduplication_NilSafety(t *testing.T) {
	t.Run("nil_field_mappings", func(t *testing.T) {
		carryOvers := make(map[vo.NodeKey][]*einobridge.FieldMapping)
		fromNodeKey := vo.NodeKey("node1")

		// Should not panic with nil fieldMappings
		assert.NotPanics(t, func() {
			addFieldMappingsWithDeduplication(carryOvers, fromNodeKey, nil)
		})

		// Should initialize empty slice for the node
		assert.NotNil(t, carryOvers[fromNodeKey])
		assert.Equal(t, 0, len(carryOvers[fromNodeKey]))
	})

	t.Run("nil_carry_overs", func(t *testing.T) {
		// Should panic with nil carryOvers - this is expected behavior
		assert.Panics(t, func() {
			addFieldMappingsWithDeduplication(nil, "node1", []*einobridge.FieldMapping{})
		})
	})
}
