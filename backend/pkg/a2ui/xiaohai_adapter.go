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

package a2ui

import "fmt"

// XiaohaiEvent represents a single event in the 集团IT智能体输出规范 v1.0.0 format.
type XiaohaiEvent struct {
	Type    string       `json:"type"`
	Data    *XiaohaiData `json:"data,omitempty"`
	Version string       `json:"version"`
}

// XiaohaiData is the data payload of a XiaohaiEvent.
type XiaohaiData struct {
	ContentType string `json:"content_type,omitempty"`
	Content     any    `json:"content,omitempty"`
}

const xiaohaiVersion = "1.0.0"

// ConvertA2UIToXiaohai converts an internal A2UI event into one or more
// XiaohaiEvent(s) conforming to the 集团IT智能体输出规范.
func ConvertA2UIToXiaohai(evt *Event) []*XiaohaiEvent {
	if evt == nil {
		return nil
	}

	switch evt.Type {
	case EventText:
		td, _ := evt.Data.(*TextData)
		if td == nil {
			return nil
		}
		delta := td.Delta
		if delta == "" {
			delta = td.Content
		}
		if delta == "" {
			return nil
		}
		return []*XiaohaiEvent{{
			Type: "answer",
			Data: &XiaohaiData{
				ContentType: "markdown",
				Content:     delta,
			},
			Version: xiaohaiVersion,
		}}

	case EventThinking:
		td, _ := evt.Data.(*ThinkingData)
		if td == nil {
			return nil
		}
		delta := td.Delta
		if delta == "" {
			delta = td.Content
		}
		if delta == "" {
			return nil
		}
		return []*XiaohaiEvent{{
			Type: "think",
			Data: &XiaohaiData{
				ContentType: "markdown",
				Content:     delta,
			},
			Version: xiaohaiVersion,
		}}

	case EventToolCall:
		tcd, _ := evt.Data.(*ToolCallData)
		if tcd == nil {
			return nil
		}
		return []*XiaohaiEvent{{
			Type: "execution_steps",
			Data: &XiaohaiData{
				ContentType: "markdown",
				Content:     fmt.Sprintf("正在调用 %s ...", tcd.Name),
			},
			Version: xiaohaiVersion,
		}}

	case EventToolResult:
		return []*XiaohaiEvent{{
			Type:    "execution_steps_end",
			Version: xiaohaiVersion,
		}}

	case EventDone:
		return []*XiaohaiEvent{{
			Type:    "stream_end",
			Version: xiaohaiVersion,
		}}

	case EventError:
		ed, _ := evt.Data.(*ErrorData)
		msg := "未知错误"
		if ed != nil {
			msg = ed.Message
		}
		return []*XiaohaiEvent{{
			Type: "answer",
			Data: &XiaohaiData{
				ContentType: "text",
				Content:     fmt.Sprintf("错误: %s", msg),
			},
			Version: xiaohaiVersion,
		}}

	case EventProgress:
		pd, _ := evt.Data.(*ProgressData)
		if pd == nil {
			return nil
		}
		return []*XiaohaiEvent{{
			Type: "execution_steps",
			Data: &XiaohaiData{
				ContentType: "text",
				Content:     fmt.Sprintf("[%s] %s (%d/%d)", pd.AgentName, pd.Step, pd.Current, pd.Total),
			},
			Version: xiaohaiVersion,
		}}

	case EventAgentSwitch:
		asd, _ := evt.Data.(*AgentSwitchData)
		if asd == nil {
			return nil
		}
		return []*XiaohaiEvent{{
			Type: "execution_steps",
			Data: &XiaohaiData{
				ContentType: "text",
				Content:     fmt.Sprintf("切换至 %s", asd.ToAgent),
			},
			Version: xiaohaiVersion,
		}}

	case EventInterrupt:
		id, _ := evt.Data.(*InterruptData)
		if id == nil {
			return nil
		}
		return []*XiaohaiEvent{{
			Type: "answer",
			Data: &XiaohaiData{
				ContentType: "text",
				Content:     fmt.Sprintf("需要确认: %s", id.Reason),
			},
			Version: xiaohaiVersion,
		}}

	default:
		return nil
	}
}
