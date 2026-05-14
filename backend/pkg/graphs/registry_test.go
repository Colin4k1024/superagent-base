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

package graphs

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// cleanRegistry removes specific names from the global registry to allow test isolation.
// Only used in tests; production code never removes entries.
func cleanRegistry(names ...string) {
	mu.Lock()
	defer mu.Unlock()
	for _, n := range names {
		delete(registry, n)
	}
}

func TestRegister_And_Get(t *testing.T) {
	name := "test-register-and-get"
	defer cleanRegistry(name)

	called := false
	factory := GraphFactory(func(ctx context.Context) (compose.Runnable[[]*schema.Message, *schema.Message], error) {
		called = true
		return nil, nil
	})

	Register(name, factory)

	got, ok := Get(name)
	if !ok {
		t.Fatalf("Get(%q) returned false, want true", name)
	}
	if got == nil {
		t.Fatal("Get returned nil factory")
	}
	// Invoke to confirm it's the same function (side-effect via closure).
	_, _ = got(context.Background())
	if !called {
		t.Error("factory returned by Get was not the registered factory")
	}
}

func TestGet_NotFound(t *testing.T) {
	_, ok := Get("test-get-nonexistent-xyz")
	if ok {
		t.Error("Get for unregistered name returned true, want false")
	}
}

func TestList(t *testing.T) {
	nameA := "test-list-alpha"
	nameB := "test-list-beta"
	defer cleanRegistry(nameA, nameB)

	noop := GraphFactory(func(ctx context.Context) (compose.Runnable[[]*schema.Message, *schema.Message], error) {
		return nil, nil
	})

	Register(nameA, noop)
	Register(nameB, noop)

	names := List()
	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	if !found[nameA] {
		t.Errorf("List() missing %q", nameA)
	}
	if !found[nameB] {
		t.Errorf("List() missing %q", nameB)
	}
}

func TestRegister_Duplicate_Panics(t *testing.T) {
	name := "test-register-duplicate"
	defer cleanRegistry(name)

	noop := GraphFactory(func(ctx context.Context) (compose.Runnable[[]*schema.Message, *schema.Message], error) {
		return nil, nil
	})

	Register(name, noop)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on duplicate registration, but got none")
		}
	}()
	Register(name, noop) // should panic
}

func TestCompileGraph(t *testing.T) {
	buildFn := func(ctx context.Context) (*compose.Graph[[]*schema.Message, *schema.Message], error) {
		g := compose.NewGraph[[]*schema.Message, *schema.Message]()
		return g, nil
	}

	factory := CompileGraph(buildFn)
	if factory == nil {
		t.Fatal("CompileGraph returned nil factory")
	}

	// Calling the factory will attempt to compile the empty graph.
	// An empty graph (no nodes) is expected to fail compilation — that is fine;
	// we only verify the factory wrapper itself is callable without panicking.
	_, _ = factory(context.Background())
}
