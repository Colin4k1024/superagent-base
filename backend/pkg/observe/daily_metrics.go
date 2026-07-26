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

import (
	"sort"
	"sync"
	"time"
)

// DailyBucket holds aggregated metrics for one calendar day.
type DailyBucket struct {
	Date             string `json:"date"`
	TraceCount       int    `json:"countTraces"`
	ObservationCount int    `json:"countObservations"`
	InputTokens      int    `json:"input_tokens"`
	OutputTokens     int    `json:"output_tokens"`
	ErrorCount       int    `json:"error_count"`
	TotalDurationMs  int64  `json:"-"`
	TotalCost        float64 `json:"totalCost"`
	Usage            []map[string]int `json:"usage,omitempty"`
}

// MetricsBucketer maintains rolling daily counters.
type MetricsBucketer struct {
	mu      sync.Mutex
	buckets map[string]*DailyBucket
	maxDays int
}

// NewMetricsBucketer creates a bucketer retaining maxDays of history.
func NewMetricsBucketer(maxDays int) *MetricsBucketer {
	if maxDays <= 0 {
		maxDays = 30
	}
	return &MetricsBucketer{
		buckets: make(map[string]*DailyBucket),
		maxDays: maxDays,
	}
}

// Record updates daily metrics from a completed trace.
func (mb *MetricsBucketer) Record(tr *TraceRecord) {
	if tr == nil {
		return
	}
	mb.mu.Lock()
	defer mb.mu.Unlock()

	day := tr.StartTime.Format("2006-01-02")
	b, ok := mb.buckets[day]
	if !ok {
		b = &DailyBucket{Date: day}
		mb.buckets[day] = b
	}

	b.TraceCount++
	b.ObservationCount += len(tr.Spans)
	b.InputTokens += tr.TotalInputTokens
	b.OutputTokens += tr.TotalOutputTokens
	b.TotalDurationMs += int64(tr.DurationMs)
	if tr.Status == "error" {
		b.ErrorCount++
	}

	mb.prune()
}

// Query returns daily buckets within [from, to] range, sorted by date ascending.
func (mb *MetricsBucketer) Query(from, to string) []DailyBucket {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	var fromTime, toTime time.Time
	if from != "" {
		fromTime, _ = time.Parse(time.RFC3339, from)
	}
	if to != "" {
		toTime, _ = time.Parse(time.RFC3339, to)
	}

	result := make([]DailyBucket, 0, len(mb.buckets))
	for _, b := range mb.buckets {
		dayTime, _ := time.Parse("2006-01-02", b.Date)
		if !fromTime.IsZero() && dayTime.Before(fromTime.Truncate(24*time.Hour)) {
			continue
		}
		if !toTime.IsZero() && dayTime.After(toTime.Truncate(24*time.Hour)) {
			continue
		}
		// Build usage array for frontend compatibility
		out := *b
		out.Usage = []map[string]int{{
			"inputUsage":  b.InputTokens,
			"outputUsage": b.OutputTokens,
			"totalUsage":  b.InputTokens + b.OutputTokens,
		}}
		result = append(result, out)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Date < result[j].Date
	})
	return result
}

// prune removes entries older than maxDays.
func (mb *MetricsBucketer) prune() {
	if len(mb.buckets) <= mb.maxDays {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -mb.maxDays).Format("2006-01-02")
	for day := range mb.buckets {
		if day < cutoff {
			delete(mb.buckets, day)
		}
	}
}
