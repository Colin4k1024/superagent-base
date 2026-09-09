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

package llm

import (
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/einobridge"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/stretchr/testify/assert"

	"github.com/superagent-ai/superagent-base/backend/api/model/app/developer_api"
	"github.com/superagent-ai/superagent-base/backend/pkg/lang/ptr"
	"github.com/superagent-ai/superagent-base/backend/pkg/urltobase64url"
)

func TestTransformMessagePart(t *testing.T) {
	mockey.PatchConvey("TestTransformMessagePart", t, func() {
		tests := []struct {
			name                 string
			part                 einobridge.ChatMessagePart
			supportedModals      *developer_api.ModelAbility
			enableTransferBase64 bool
			expectedPart         einobridge.ChatMessagePart
			mockB64              bool
		}{
			{
				name: "Image modal not supported",
				part: einobridge.ChatMessagePart{
					Type:     einobridge.ChatMessagePartTypeImageURL,
					ImageURL: &einobridge.ChatMessageImageURL{URL: "http://example.com/image.png"},
				},
				supportedModals: &developer_api.ModelAbility{},
				expectedPart: einobridge.ChatMessagePart{
					Type: einobridge.ChatMessagePartTypeText,
					Text: "http://example.com/image.png",
				},
			},
			{
				name: "Image modal supported, no base64 transfer",
				part: einobridge.ChatMessagePart{
					Type:     einobridge.ChatMessagePartTypeImageURL,
					ImageURL: &einobridge.ChatMessageImageURL{URL: "http://example.com/image.png"},
				},
				supportedModals: &developer_api.ModelAbility{ImageUnderstanding: ptr.Of(true), SupportMultiModal: ptr.Of(true)},
				expectedPart: einobridge.ChatMessagePart{
					Type:     einobridge.ChatMessagePartTypeImageURL,
					ImageURL: &einobridge.ChatMessageImageURL{URL: "http://example.com/image.png"},
				},
			},
			{
				name: "Audio modal not supported",
				part: einobridge.ChatMessagePart{
					Type:     einobridge.ChatMessagePartTypeAudioURL,
					AudioURL: &einobridge.ChatMessageAudioURL{URL: "http://example.com/audio.mp3"},
				},
				supportedModals: &developer_api.ModelAbility{},
				expectedPart: einobridge.ChatMessagePart{
					Type: einobridge.ChatMessagePartTypeText,
					Text: "http://example.com/audio.mp3",
				},
			},
			{
				name: "Video modal not supported",
				part: einobridge.ChatMessagePart{
					Type:     einobridge.ChatMessagePartTypeVideoURL,
					VideoURL: &einobridge.ChatMessageVideoURL{URL: "http://example.com/video.mp4"},
				},
				supportedModals: &developer_api.ModelAbility{},
				expectedPart: einobridge.ChatMessagePart{
					Type: einobridge.ChatMessagePartTypeText,
					Text: "http://example.com/video.mp4",
				},
			},
			{
				name: "File modal not supported",
				part: einobridge.ChatMessagePart{
					Type:    einobridge.ChatMessagePartTypeFileURL,
					FileURL: &einobridge.ChatMessageFileURL{URL: "http://example.com/file.txt"},
				},
				supportedModals: &developer_api.ModelAbility{},
				expectedPart: einobridge.ChatMessagePart{
					Type: einobridge.ChatMessagePartTypeText,
					Text: "http://example.com/file.txt",
				},
			},
			{
				name: "Text part is unchanged",
				part: einobridge.ChatMessagePart{
					Type: einobridge.ChatMessagePartTypeText,
					Text: "hello world",
				},
				supportedModals: &developer_api.ModelAbility{},
				expectedPart: einobridge.ChatMessagePart{
					Type: einobridge.ChatMessagePartTypeText,
					Text: "hello world",
				},
			},
			{
				name: "Image modal supported, with base64 transfer",
				part: einobridge.ChatMessagePart{
					Type:     einobridge.ChatMessagePartTypeImageURL,
					ImageURL: &einobridge.ChatMessageImageURL{URL: "http://example.com/image.png"},
				},
				supportedModals:      &developer_api.ModelAbility{ImageUnderstanding: ptr.Of(true), SupportMultiModal: ptr.Of(true)},
				enableTransferBase64: true,
				expectedPart: einobridge.ChatMessagePart{
					Type: einobridge.ChatMessagePartTypeImageURL,
					ImageURL: &einobridge.ChatMessageImageURL{
						URL:      "data:image/png;base64,base64encodedstring",
						MIMEType: "image/png",
					},
				},
				mockB64: true,
			},
			{
				name: "Audio modal supported, with base64 transfer",
				part: einobridge.ChatMessagePart{
					Type:     einobridge.ChatMessagePartTypeAudioURL,
					AudioURL: &einobridge.ChatMessageAudioURL{URL: "http://example.com/audio.mp3"},
				},
				supportedModals:      &developer_api.ModelAbility{AudioUnderstanding: ptr.Of(true), SupportMultiModal: ptr.Of(true)},
				enableTransferBase64: true,
				expectedPart: einobridge.ChatMessagePart{
					Type: einobridge.ChatMessagePartTypeAudioURL,
					AudioURL: &einobridge.ChatMessageAudioURL{
						URL:      "data:audio/mpeg;base64,base64encodedstring",
						MIMEType: "audio/mpeg",
					},
				},
				mockB64: true,
			},
			{
				name: "Video modal supported, with base64 transfer",
				part: einobridge.ChatMessagePart{
					Type:     einobridge.ChatMessagePartTypeVideoURL,
					VideoURL: &einobridge.ChatMessageVideoURL{URL: "http://example.com/video.mp4"},
				},
				supportedModals:      &developer_api.ModelAbility{VideoUnderstanding: ptr.Of(true), SupportMultiModal: ptr.Of(true)},
				enableTransferBase64: true,
				expectedPart: einobridge.ChatMessagePart{
					Type: einobridge.ChatMessagePartTypeVideoURL,
					VideoURL: &einobridge.ChatMessageVideoURL{
						URL:      "data:video/mp4;base64,base64encodedstring",
						MIMEType: "video/mp4",
					},
				},
				mockB64: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.mockB64 {
					t.Cleanup(mockey.UnPatchAll)
					var u, m string
					switch tt.part.Type {
					case einobridge.ChatMessagePartTypeImageURL:
						u, m = tt.expectedPart.ImageURL.URL, tt.expectedPart.ImageURL.MIMEType
					case einobridge.ChatMessagePartTypeAudioURL:
						u, m = tt.expectedPart.AudioURL.URL, tt.expectedPart.AudioURL.MIMEType
					case einobridge.ChatMessagePartTypeVideoURL:
						u, m = tt.expectedPart.VideoURL.URL, tt.expectedPart.VideoURL.MIMEType
					case einobridge.ChatMessagePartTypeFileURL:
						u, m = tt.expectedPart.FileURL.URL, tt.expectedPart.FileURL.MIMEType
					}
					mockey.Mock(urltobase64url.URLToBase64).Return(&urltobase64url.FileData{
						Base64Url: u,
						MimeType:  m,
					}, nil).Build()
				}
				result := transformMessagePart(tt.part, tt.supportedModals, tt.enableTransferBase64)
				assert.Equal(t, tt.expectedPart, result)
			})
		}
	})
}
