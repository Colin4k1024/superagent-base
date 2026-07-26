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

// Package crossdomain provides a unified service registry that replaces
// the 16 separate global SetDefaultSVC/DefaultSVC variables.
//
// Migration path:
//  1. application.Init() creates a ServiceRegistry and calls SetDefaultRegistry()
//  2. New code uses crossdomain.Default().Agent() instead of agent.DefaultSVC()
//  3. Old code continues to work via the legacy SetDefaultSVC calls in application.Init()
//  4. Once all call sites are migrated, remove the per-package SetDefaultSVC/DefaultSVC
package crossdomain

import (
	"sync"

	"github.com/superagent-ai/superagent-base/backend/crossdomain/agent"
	agentrun "github.com/superagent-ai/superagent-base/backend/crossdomain/agentrun"
	"github.com/superagent-ai/superagent-base/backend/crossdomain/app"
	"github.com/superagent-ai/superagent-base/backend/crossdomain/connector"
	"github.com/superagent-ai/superagent-base/backend/crossdomain/conversation"
	"github.com/superagent-ai/superagent-base/backend/crossdomain/database"
	"github.com/superagent-ai/superagent-base/backend/crossdomain/datacopy"
	"github.com/superagent-ai/superagent-base/backend/crossdomain/knowledge"
	"github.com/superagent-ai/superagent-base/backend/crossdomain/message"
	"github.com/superagent-ai/superagent-base/backend/crossdomain/permission"
	"github.com/superagent-ai/superagent-base/backend/crossdomain/plugin"
	"github.com/superagent-ai/superagent-base/backend/crossdomain/search"
	"github.com/superagent-ai/superagent-base/backend/crossdomain/upload"
	"github.com/superagent-ai/superagent-base/backend/crossdomain/user"
	"github.com/superagent-ai/superagent-base/backend/crossdomain/variables"
	"github.com/superagent-ai/superagent-base/backend/crossdomain/workflow"
)

// ServiceRegistry holds all crossdomain service interfaces.
// It replaces the 16 separate global variables with a single,
// testable, replaceable registry.
type ServiceRegistry struct {
	agent        agent.SingleAgent
	agentRun     agentrun.AgentRun
	app          app.AppService
	connector    connector.Connector
	conversation conversation.Conversation
	database     database.Database
	dataCopy     datacopy.DataCopy
	knowledge    knowledge.Knowledge
	message      message.Message
	permission   permission.Permission
	plugin       plugin.PluginService
	search       search.Search
	upload       upload.Uploader
	user         user.User
	variables    variables.Variables
	workflow     workflow.Workflow
}

// RegistryOption configures the ServiceRegistry.
type RegistryOption func(*ServiceRegistry)

// NewServiceRegistry creates a new ServiceRegistry with the given options.
func NewServiceRegistry(opts ...RegistryOption) *ServiceRegistry {
	r := &ServiceRegistry{}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// WithAgent sets the agent service.
func WithAgent(s agent.SingleAgent) RegistryOption {
	return func(r *ServiceRegistry) { r.agent = s }
}

// WithAgentRun sets the agent run service.
func WithAgentRun(s agentrun.AgentRun) RegistryOption {
	return func(r *ServiceRegistry) { r.agentRun = s }
}

// WithApp sets the app service.
func WithApp(s app.AppService) RegistryOption {
	return func(r *ServiceRegistry) { r.app = s }
}

// WithConnector sets the connector service.
func WithConnector(s connector.Connector) RegistryOption {
	return func(r *ServiceRegistry) { r.connector = s }
}

// WithConversation sets the conversation service.
func WithConversation(s conversation.Conversation) RegistryOption {
	return func(r *ServiceRegistry) { r.conversation = s }
}

// WithDatabase sets the database service.
func WithDatabase(s database.Database) RegistryOption {
	return func(r *ServiceRegistry) { r.database = s }
}

// WithDataCopy sets the data copy service.
func WithDataCopy(s datacopy.DataCopy) RegistryOption {
	return func(r *ServiceRegistry) { r.dataCopy = s }
}

// WithKnowledge sets the knowledge service.
func WithKnowledge(s knowledge.Knowledge) RegistryOption {
	return func(r *ServiceRegistry) { r.knowledge = s }
}

// WithMessage sets the message service.
func WithMessage(s message.Message) RegistryOption {
	return func(r *ServiceRegistry) { r.message = s }
}

// WithPermission sets the permission service.
func WithPermission(s permission.Permission) RegistryOption {
	return func(r *ServiceRegistry) { r.permission = s }
}

// WithPlugin sets the plugin service.
func WithPlugin(s plugin.PluginService) RegistryOption {
	return func(r *ServiceRegistry) { r.plugin = s }
}

// WithSearch sets the search service.
func WithSearch(s search.Search) RegistryOption {
	return func(r *ServiceRegistry) { r.search = s }
}

// WithUpload sets the upload service.
func WithUpload(s upload.Uploader) RegistryOption {
	return func(r *ServiceRegistry) { r.upload = s }
}

// WithUser sets the user service.
func WithUser(s user.User) RegistryOption {
	return func(r *ServiceRegistry) { r.user = s }
}

// WithVariables sets the variables service.
func WithVariables(s variables.Variables) RegistryOption {
	return func(r *ServiceRegistry) { r.variables = s }
}

// WithWorkflow sets the workflow service.
func WithWorkflow(s workflow.Workflow) RegistryOption {
	return func(r *ServiceRegistry) { r.workflow = s }
}

// ── Accessors ───────────────────────────────────────────────────────────────

func (r *ServiceRegistry) Agent() agent.SingleAgent          { return r.agent }
func (r *ServiceRegistry) AgentRun() agentrun.AgentRun       { return r.agentRun }
func (r *ServiceRegistry) App() app.AppService               { return r.app }
func (r *ServiceRegistry) Connector() connector.Connector    { return r.connector }
func (r *ServiceRegistry) Conversation() conversation.Conversation { return r.conversation }
func (r *ServiceRegistry) Database() database.Database       { return r.database }
func (r *ServiceRegistry) DataCopy() datacopy.DataCopy       { return r.dataCopy }
func (r *ServiceRegistry) Knowledge() knowledge.Knowledge    { return r.knowledge }
func (r *ServiceRegistry) Message() message.Message          { return r.message }
func (r *ServiceRegistry) Permission() permission.Permission { return r.permission }
func (r *ServiceRegistry) Plugin() plugin.PluginService      { return r.plugin }
func (r *ServiceRegistry) Search() search.Search             { return r.search }
func (r *ServiceRegistry) Upload() upload.Uploader           { return r.upload }
func (r *ServiceRegistry) User() user.User                   { return r.user }
func (r *ServiceRegistry) Variables() variables.Variables    { return r.variables }
func (r *ServiceRegistry) Workflow() workflow.Workflow        { return r.workflow }

// ── Global Registry ─────────────────────────────────────────────────────────

var (
	defaultRegistry *ServiceRegistry
	registryMu      sync.RWMutex
)

// SetDefaultRegistry sets the global service registry.
// Must be called once during application startup.
func SetDefaultRegistry(r *ServiceRegistry) {
	registryMu.Lock()
	defer registryMu.Unlock()
	defaultRegistry = r
}

// Default returns the global service registry.
// Returns nil if SetDefaultRegistry has not been called.
func Default() *ServiceRegistry {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return defaultRegistry
}

// MustDefault returns the global service registry, panicking if nil.
// Use only in startup paths where the registry must have been initialized.
func MustDefault() *ServiceRegistry {
	r := Default()
	if r == nil {
		panic("crossdomain: service registry not initialized; call SetDefaultRegistry first")
	}
	return r
}
