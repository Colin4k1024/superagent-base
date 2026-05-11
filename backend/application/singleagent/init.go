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

package singleagent

import (
	"github.com/cloudwego/eino/compose"
	"gorm.io/gorm"

	"github.com/superagent-ai/superagent-base/backend/domain/agent/singleagent/entity"
	"github.com/superagent-ai/superagent-base/backend/domain/agent/singleagent/repository"
	singleagent "github.com/superagent-ai/superagent-base/backend/domain/agent/singleagent/service"
	connector "github.com/superagent-ai/superagent-base/backend/domain/connector/service"
	knowledge "github.com/superagent-ai/superagent-base/backend/domain/knowledge/service"
	database "github.com/superagent-ai/superagent-base/backend/domain/memory/database/service"
	variables "github.com/superagent-ai/superagent-base/backend/domain/memory/variables/service"
	"github.com/superagent-ai/superagent-base/backend/domain/plugin/service"
	search "github.com/superagent-ai/superagent-base/backend/domain/search/service"
	shortcutCmd "github.com/superagent-ai/superagent-base/backend/domain/shortcutcmd/service"
	user "github.com/superagent-ai/superagent-base/backend/domain/user/service"
	"github.com/superagent-ai/superagent-base/backend/domain/workflow"
	"github.com/superagent-ai/superagent-base/backend/infra/cache"
	"github.com/superagent-ai/superagent-base/backend/infra/idgen"
	"github.com/superagent-ai/superagent-base/backend/infra/imagex"
	"github.com/superagent-ai/superagent-base/backend/infra/storage"
	"github.com/superagent-ai/superagent-base/backend/pkg/kvstore"
)

type (
	SingleAgent = singleagent.SingleAgent
)

var SingleAgentSVC *SingleAgentApplicationService

type ServiceComponents struct {
	IDGen       idgen.IDGenerator
	DB          *gorm.DB
	Cache       cache.Cmdable
	TosClient   storage.Storage
	ImageX      imagex.ImageX
	EventBus    search.ProjectEventBus
	CounterRepo repository.CounterRepository

	KnowledgeDomainSVC   knowledge.Knowledge
	PluginDomainSVC      service.PluginService
	WorkflowDomainSVC    workflow.Service
	UserDomainSVC        user.User
	VariablesDomainSVC   variables.Variables
	ConnectorDomainSVC   connector.Connector
	DatabaseDomainSVC    database.Database
	ShortcutCMDDomainSVC shortcutCmd.ShortcutCmd
	CPStore              compose.CheckPointStore
}

func InitService(c *ServiceComponents) (*SingleAgentApplicationService, error) {
	domainComponents := &singleagent.Components{
		AgentDraftRepo:   repository.NewSingleAgentRepo(c.DB, c.IDGen, c.Cache),
		AgentVersionRepo: repository.NewSingleAgentVersionRepo(c.DB, c.IDGen),
		PublishInfoRepo:  kvstore.New[entity.PublishInfo](c.DB),
		CounterRepo:      repository.NewCounterRepo(c.Cache),
		CPStore:          c.CPStore,
	}

	singleAgentDomainSVC := singleagent.NewService(domainComponents)
	SingleAgentSVC = newApplicationService(c, singleAgentDomainSVC)

	return SingleAgentSVC, nil
}
