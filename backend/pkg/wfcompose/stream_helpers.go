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

import (
	"sync"
	"io"
)

// StreamReaderFromArray creates a StreamReader that yields the given
// elements one at a time, then returns io.EOF. This mirrors
// schema.StreamReaderFromArray from eino.
func StreamReaderFromArray[T any](arr []T) *StreamReader[T] {
	idx := 0
	return NewStreamReader(func() (T, error) {
		var zero T
		if idx >= len(arr) {
			return zero, io.EOF
		}
		v := arr[idx]
		idx++
		return v, nil
	}, nil)
}

// StreamReaderWithConvert converts a StreamReader of type I into a
// StreamReader of type O by applying the convert function to each
// element. This mirrors schema.StreamReaderWithConvert from eino.
func StreamReaderWithConvert[I any, O any](in *StreamReader[I], convert func(I) (O, error)) *StreamReader[O] {
	if in == nil {
		return nil
	}
	return NewStreamReader(func() (O, error) {
		var zero O
		v, err := in.Recv()
		if err != nil {
			return zero, err
		}
		return convert(v)
	}, func() { in.Close() })
}

// MergeStreamReaders merges multiple StreamReaders into a single reader
// that yields elements from all inputs in order (round-robin). This
// mirrors schema.MergeStreamReaders from eino.
func MergeStreamReaders[T any](readers []*StreamReader[T]) *StreamReader[T] {
	if len(readers) == 0 {
		return nil
	}
	if len(readers) == 1 {
		return readers[0]
	}

	idx := 0
	active := len(readers)
	closed := make([]bool, len(readers))

	return NewStreamReader(func() (T, error) {
		var zero T
		if active == 0 {
			return zero, io.EOF
		}

		for attempts := 0; attempts < len(readers); attempts++ {
			if idx >= len(readers) {
				idx = 0
			}
			if closed[idx] {
				idx++
				continue
			}

			r := readers[idx]
			if r == nil {
				closed[idx] = true
				active--
				idx++
				continue
			}

			v, err := r.Recv()
			if err == io.EOF || err != nil {
				closed[idx] = true
				active--
				idx++
				if err != io.EOF && err != nil {
					return zero, err
				}
				continue
			}
			idx++
			return v, nil
		}

		return zero, io.EOF
	}, func() {
		for _, r := range readers {
			if r != nil {
				r.Close()
			}
		}
	})
}

// Pipe creates a paired StreamReader/StreamWriter backed by a buffered
// channel. Data sent via StreamWriter.Send becomes available on
// StreamReader.Recv. Closing the writer causes the reader to eventually
// return io.EOF. This mirrors schema.Pipe from eino.
func Pipe[T any](cap int) (*StreamReader[T], *StreamWriter[T]) {
	ch := make(chan T, cap)
	errCh := make(chan error, 1)
	closed := false

	mu := new(sync.Mutex)

	reader := &StreamReader[T]{
		recv: func() (T, error) {
			var zero T
			select {
			case err, ok := <-errCh:
				if !ok {
					return zero, io.EOF
				}
				return zero, err
			case v, ok := <-ch:
				if !ok {
					return zero, io.EOF
				}
				return v, nil
			}
		},
		close: func() {
			mu.Lock()
			defer mu.Unlock()
			if closed {
				return
			}
			closed = true
			close(ch)
		},
	}

	writer := &StreamWriter[T]{
		send: func(v T, err error) {
			if err != nil {
				mu.Lock()
				if !closed {
					closed = true
					errCh <- err
					close(ch)
				}
				mu.Unlock()
				return
			}
			ch <- v
		},
		close: func() {
			mu.Lock()
			defer mu.Unlock()
			if closed {
				return
			}
			closed = true
			close(ch)
		},
	}

	return reader, writer
}
