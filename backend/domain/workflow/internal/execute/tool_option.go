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

package execute

import (
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/einobridge"
	"github.com/cloudwego/eino/compose"

	workflowModel "github.com/superagent-ai/superagent-base/backend/crossdomain/workflow/model"
	"github.com/superagent-ai/superagent-base/backend/domain/workflow/entity"
)

type workflowToolOption struct {
	resumeReq            *entity.ResumeRequest
	streamContainer      *StreamContainer
	exeCfg               workflowModel.ExecuteConfig
	toolCallID2ExecuteID map[string]int64
}

func WithResume(req *entity.ResumeRequest, all map[string]int64) einobridge.ToolOption {
	return einobridge.ToolWrapImplSpecificOptFn(func(opts *workflowToolOption) {
		opts.resumeReq = req
		opts.toolCallID2ExecuteID = all
	})
}

func WithParentStreamContainer(sc *StreamContainer) einobridge.ToolOption {
	return einobridge.ToolWrapImplSpecificOptFn(func(opts *workflowToolOption) {
		opts.streamContainer = sc
	})
}

func WithExecuteConfig(cfg workflowModel.ExecuteConfig) einobridge.ToolOption {
	return einobridge.ToolWrapImplSpecificOptFn(func(opts *workflowToolOption) {
		opts.exeCfg = cfg
	})
}

func GetResumeRequest(opts ...einobridge.ToolOption) (*entity.ResumeRequest, map[string]int64) {
	opt := einobridge.ToolGetImplSpecificOptions(&workflowToolOption{}, opts...)
	return opt.resumeReq, opt.toolCallID2ExecuteID
}

func GetParentStreamContainer(opts ...einobridge.ToolOption) *StreamContainer {
	opt := einobridge.ToolGetImplSpecificOptions(&workflowToolOption{}, opts...)
	return opt.streamContainer
}

func GetExecuteConfig(opts ...einobridge.ToolOption) workflowModel.ExecuteConfig {
	opt := einobridge.ToolGetImplSpecificOptions(&workflowToolOption{}, opts...)
	return opt.exeCfg
}

// WithMessagePipe returns an Option which is meant to be passed to the tool workflow,
// as well as a StreamReader to read the messages from the tool workflow.
// This Option will apply to ALL workflow tools to be executed by eino's ToolsNode.
// The workflow tools will emit messages to this stream.
// The caller can receive from the returned StreamReader to get the messages from the tool workflow.
func WithMessagePipe() (compose.Option, *einobridge.StreamReader[*entity.Message], func()) {
	sr, sw := einobridge.Pipe[*entity.Message](10)
	container := &StreamContainer{
		sw:         sw,
		subStreams: make(chan *einobridge.StreamReader[*entity.Message]),
	}

	go container.PipeAll()

	opt := compose.WithToolsNodeOption(compose.WithToolOption(WithParentStreamContainer(container)))
	return opt, sr, func() {
		container.Done()
	}
}
