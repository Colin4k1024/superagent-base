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

package agentdef

import (
	"context"
	"testing"

	aclagent "github.com/superagent-ai/superagent-base/backend/pkg/agent"
)

func TestRegisterMiddleware_AndGet(t *testing.T) {
	name := "test_mw_register"
	factory := func(_ context.Context, _ map[string]any) (aclagent.Middleware, error) {
		return aclagent.BaseMiddleware{}, nil
	}

	RegisterMiddleware(name, factory)

	got, ok := GetMiddlewareFactory(name)
	if !ok {
		t.Fatalf("expected factory for %q to be registered", name)
	}
	if got == nil {
		t.Fatal("factory should not be nil")
	}

	mw, err := got(context.Background(), nil)
	if err != nil {
		t.Fatalf("factory returned error: %v", err)
	}
	if mw == nil {
		t.Fatal("factory returned nil middleware")
	}
}

func TestGetMiddlewareFactory_NotFound(t *testing.T) {
	_, ok := GetMiddlewareFactory("nonexistent_middleware_xyz")
	if ok {
		t.Error("expected not-found for unregistered middleware")
	}
}

func TestListMiddleware_ContainsRegistered(t *testing.T) {
	name := "test_mw_list"
	RegisterMiddleware(name, func(_ context.Context, _ map[string]any) (aclagent.Middleware, error) {
		return aclagent.BaseMiddleware{}, nil
	})

	names := ListMiddleware()
	found := false
	for _, n := range names {
		if n == name {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ListMiddleware() should contain %q", name)
	}
}

func TestRegisterMiddleware_PanicsOnEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on empty name")
		}
	}()
	RegisterMiddleware("", func(_ context.Context, _ map[string]any) (aclagent.Middleware, error) {
		return nil, nil
	})
}

func TestRegisterMiddleware_PanicsOnNilFactory(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on nil factory")
		}
	}()
	RegisterMiddleware("test_nil_factory", nil)
}
