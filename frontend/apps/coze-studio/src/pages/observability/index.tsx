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

import { useTranslation } from 'react-i18next';
import { useState } from 'react';

import {
  Layout,
  Header,
  HeaderTitle,
  SubHeader,
  SubHeaderFilters,
  Content,
} from '@coze-studio/workspace-base/develop';
import { Select } from '@coze-arch/coze-design';

import TracesTab from '@/components/observability/TracesTab';
import ToolsTab from '@/components/observability/ToolsTab';
import TokenUsageTab from '@/components/observability/TokenUsageTab';
import OverviewTab from '@/components/observability/OverviewTab';
import ModelRoutingTab from '@/components/observability/ModelRoutingTab';
import AgentAnalyticsTab from '@/components/observability/AgentAnalyticsTab';

type TabKey = 'overview' | 'traces' | 'tokens' | 'tools' | 'agents' | 'models';

export default function ObservabilityPage() {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState<TabKey>('overview');

  const tabOptions = [
    { value: 'overview', label: t('observability.tabs.overview', '概览') },
    { value: 'traces', label: t('observability.tabs.traces', '链路追踪') },
    { value: 'tokens', label: t('observability.tabs.tokens', 'Token 用量') },
    { value: 'tools', label: t('observability.tabs.tools', '工具') },
    { value: 'agents', label: t('observability.tabs.agents', '智能体') },
    { value: 'models', label: t('observability.tabs.models', '模型') },
  ];

  return (
    <Layout>
      <Header>
        <HeaderTitle>
          <span>{t('observability.title', '可观测性')}</span>
        </HeaderTitle>
      </Header>

      <SubHeader>
        <SubHeaderFilters>
          <Select
            className="min-w-[128px]"
            value={activeTab}
            onChange={v => setActiveTab(v as TabKey)}
          >
            {tabOptions.map(opt => (
              <Select.Option key={opt.value} value={opt.value}>
                {opt.label}
              </Select.Option>
            ))}
          </Select>
        </SubHeaderFilters>
      </SubHeader>

      <Content>
        {activeTab === 'overview' && <OverviewTab />}
        {activeTab === 'traces' && <TracesTab />}
        {activeTab === 'tokens' && <TokenUsageTab />}
        {activeTab === 'tools' && <ToolsTab />}
        {activeTab === 'agents' && <AgentAnalyticsTab />}
        {activeTab === 'models' && <ModelRoutingTab />}
      </Content>
    </Layout>
  );
}
