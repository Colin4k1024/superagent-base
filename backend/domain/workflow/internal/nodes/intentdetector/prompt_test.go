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

package intentdetector

import (
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/einobridge"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandleHistoryMessages(t *testing.T) {
	tests := []struct {
		name             string
		historyMessages  []*einobridge.Message
		expectedMessages []*einobridge.Message
	}{
		{
			name:             "Empty history",
			historyMessages:  []*einobridge.Message{},
			expectedMessages: []*einobridge.Message{},
		},
		{
			name:             "Message with only content",
			historyMessages:  []*einobridge.Message{{Content: "hello"}},
			expectedMessages: []*einobridge.Message{{Content: "hello", MultiContent: nil}},
		},
		{
			name: "Message with only single text multi-content",
			historyMessages: []*einobridge.Message{
				{
					MultiContent: []einobridge.ChatMessagePart{
						{Type: einobridge.ChatMessagePartTypeText, Text: "world"},
					},
				},
			},
			expectedMessages: []*einobridge.Message{{Content: "world", MultiContent: nil}},
		},
		{
			name: "Message with content and multi-content",
			historyMessages: []*einobridge.Message{
				{
					Content: "hello",
					MultiContent: []einobridge.ChatMessagePart{
						{Type: einobridge.ChatMessagePartTypeText, Text: "world"},
					},
				},
			},
			expectedMessages: []*einobridge.Message{{Content: "hello\nworld", MultiContent: nil}},
		},
		{
			name: "Message with multiple multi-content parts",
			historyMessages: []*einobridge.Message{
				{
					MultiContent: []einobridge.ChatMessagePart{
						{Type: einobridge.ChatMessagePartTypeText, Text: "part1"},
						{Type: einobridge.ChatMessagePartTypeText, Text: "part2"},
					},
				},
			},
			expectedMessages: []*einobridge.Message{{Content: "part1\npart2", MultiContent: nil}},
		},
		{
			name: "Message with various multi-content part types",
			historyMessages: []*einobridge.Message{
				{
					MultiContent: []einobridge.ChatMessagePart{
						{Type: einobridge.ChatMessagePartTypeText, Text: "text"},
						{Type: einobridge.ChatMessagePartTypeImageURL, ImageURL: &einobridge.ChatMessageImageURL{URL: "image.png"}},
						{Type: einobridge.ChatMessagePartTypeAudioURL, AudioURL: &einobridge.ChatMessageAudioURL{URL: "audio.mp3"}},
						{Type: einobridge.ChatMessagePartTypeVideoURL, VideoURL: &einobridge.ChatMessageVideoURL{URL: "video.mp4"}},
						{Type: einobridge.ChatMessagePartTypeFileURL, FileURL: &einobridge.ChatMessageFileURL{URL: "file.txt"}},
					},
				},
			},
			expectedMessages: []*einobridge.Message{{Content: "text\nimage.png\naudio.mp3\nvideo.mp4\nfile.txt", MultiContent: nil}},
		},
		{
			name: "Multiple messages",
			historyMessages: []*einobridge.Message{
				{Content: "msg1"},
				{MultiContent: []einobridge.ChatMessagePart{{Type: einobridge.ChatMessagePartTypeText, Text: "msg2"}}},
			},
			expectedMessages: []*einobridge.Message{
				{Content: "msg1", MultiContent: nil},
				{Content: "msg2", MultiContent: nil},
			},
		},
		{
			name:             "Empty message",
			historyMessages:  []*einobridge.Message{{}},
			expectedMessages: []*einobridge.Message{{Content: "", MultiContent: nil}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handleHistoryMessages(tt.historyMessages)
			assert.Equal(t, tt.expectedMessages, result)
		})
	}
}
