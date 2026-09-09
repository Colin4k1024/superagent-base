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

// message.go provides identity conversion functions that were needed when
// einobridge.Message = schema.Message but the compose engine expected
// wfcompose.Message. Now that both are wfcompose.Message, these are no-ops.
// They remain for API compatibility with existing callers.
package einobridge

import (
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
)

// WrapMessage is an identity function (was: schema.Message → wfcompose.Message).
func WrapMessage(m *wfcompose.Message) *wfcompose.Message {
	return m
}

// UnwrapMessage is an identity function (was: wfcompose.Message → schema.Message).
func UnwrapMessage(m *wfcompose.Message) *wfcompose.Message {
	return m
}

// WrapMessageSlice is an identity slice converter.
func WrapMessageSlice(in []*wfcompose.Message) []*wfcompose.Message {
	return in
}

// UnwrapMessageSlice is an identity slice converter.
func UnwrapMessageSlice(in []*wfcompose.Message) []*wfcompose.Message {
	return in
}

// WrapDocument is an identity function.
func WrapDocument(d *wfcompose.Document) *wfcompose.Document {
	return d
}

// UnwrapDocument is an identity function.
func UnwrapDocument(d *wfcompose.Document) *wfcompose.Document {
	return d
}

// WrapDocumentSlice is an identity slice converter.
func WrapDocumentSlice(in []*wfcompose.Document) []*wfcompose.Document {
	return in
}

// UnwrapDocumentSlice is an identity slice converter.
func UnwrapDocumentSlice(in []*wfcompose.Document) []*wfcompose.Document {
	return in
}
