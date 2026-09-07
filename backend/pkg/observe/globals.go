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

package observe

// defaultTraceStore and defaultBucketer are set during startup for local trace collection.
var (
	defaultTraceStore *TraceStore
	defaultBucketer   *MetricsBucketer
)

// SetTraceStore sets the global trace store for span collection.
func SetTraceStore(ts *TraceStore) { defaultTraceStore = ts }

// SetMetricsBucketer sets the global daily metrics bucketer.
func SetMetricsBucketer(mb *MetricsBucketer) { defaultBucketer = mb }

// GetTraceStore returns the global trace store.
func GetTraceStore() *TraceStore { return defaultTraceStore }

// GetMetricsBucketer returns the global metrics bucketer.
func GetMetricsBucketer() *MetricsBucketer { return defaultBucketer }
