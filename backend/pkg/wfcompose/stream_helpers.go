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

package wfcompose

import "io"

type streamElement[T any] struct {
	val T
	err  error
}

// Pipe creates a connected pair of stream reader/writer with the given
// buffer capacity. It mirrors eino's schema.Pipe.
func Pipe[T any](cap int) (*StreamReader[T], *StreamWriter[T]) {
	ch := make(chan streamElement[T], cap)
	closed := false

	writer := &StreamWriter[T]{
		send: func(v T, err error) {
			if closed {
				return
			}
			ch <- streamElement[T]{val: v, err: err}
		},
		close: func() {
			if !closed {
				closed = true
				close(ch)
			}
		},
	}

	reader := &StreamReader[T]{
		recv: func() (T, error) {
			var zero T
			elem, ok := <-ch
			if !ok {
				return zero, io.EOF
			}
			return elem.val, elem.err
		},
		close: writer.close,
	}

	return reader, writer
}

// StreamReaderWithConvert creates a new StreamReader that applies a convert
// function to each element from the source reader. If convert returns
// ErrNoValue, the element is silently dropped and the next is read.
func StreamReaderWithConvert[T, D any](sr *StreamReader[T], convert func(T) (D, error), opts ...ConvertOption) *StreamReader[D] {
	if sr == nil {
		return nil
	}
	return &StreamReader[D]{
		recv: func() (D, error) {
			for {
				val, err := sr.Recv()
				if err != nil {
					var zero D
					return zero, err
				}
				converted, convErr := convert(val)
				if convErr != nil {
					if convErr == ErrNoValue {
						continue
					}
					var zero D
					return zero, convErr
				}
				return converted, nil
			}
		},
		close: sr.Close,
	}
}

// StreamReaderFromArray creates a StreamReader that yields the given slice
// elements one at a time, then returns io.EOF.
func StreamReaderFromArray[T any](arr []T) *StreamReader[T] {
	idx := 0
	return &StreamReader[T]{
		recv: func() (T, error) {
			if idx >= len(arr) {
				var zero T
				return zero, io.EOF
			}
			val := arr[idx]
			idx++
			return val, nil
		},
		close: func() {},
	}
}

// MergeStreamReaders merges multiple stream readers into one. Readers are
// consumed sequentially: when one finishes (io.EOF), the next is started.
func MergeStreamReaders[T any](srs []*StreamReader[T]) *StreamReader[T] {
	if len(srs) == 0 {
		return nil
	}
	idx := 0
	return &StreamReader[T]{
		recv: func() (T, error) {
			for idx < len(srs) {
				if srs[idx] == nil {
					idx++
					continue
				}
				val, err := srs[idx].Recv()
				if err != nil {
					if IsEOF(err) {
						srs[idx].Close()
						idx++
						continue
					}
					return val, err
				}
				return val, nil
			}
			var zero T
			return zero, io.EOF
		},
		close: func() {
			for _, sr := range srs {
				if sr != nil {
					sr.Close()
				}
			}
		},
	}
}
