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

package hiagent

// Config holds connection parameters for HiAgent API.
type Config struct {
	BaseURL string
	APIKey  string
	AppKey  string
}

type CreateConversationRequest struct {
	UserID string            `json:"UserID"`
	Inputs map[string]string `json:"Inputs,omitempty"`
}

type CreateConversationResponse struct {
	Conversation struct {
		AppConversationID string `json:"AppConversationID"`
		ConversationName  string `json:"ConversationName"`
	} `json:"Conversation"`
	BaseResp *BaseResp `json:"BaseResp,omitempty"`
}

type BaseResp struct {
	StatusCode    int    `json:"StatusCode"`
	StatusMessage string `json:"StatusMessage"`
}

type ChatQueryV2Request struct {
	AppConversationID string        `json:"AppConversationID"`
	Query             string        `json:"Query"`
	ResponseMode      string        `json:"ResponseMode"`
	UserID            string        `json:"UserID"`
	QueryExtends      *QueryExtends `json:"QueryExtends,omitempty"`
}

type QueryExtends struct {
	Files []FileInfo `json:"Files,omitempty"`
}

type FileInfo struct {
	Name string `json:"Name"`
	Path string `json:"Path"`
	Size int64  `json:"Size"`
	URL  string `json:"Url"`
}

type ChatQueryV2Response struct {
	Event          string   `json:"event"`
	Answer         string   `json:"answer"`
	TaskID         string   `json:"task_id"`
	ID             string   `json:"id"`
	CreatedAt      int64    `json:"created_at"`
	ConversationID string   `json:"conversation_id"`
	ThinkMessages  []string `json:"think_messages,omitempty"`
	ToolMessages   []string `json:"tool_messages,omitempty"`
}
