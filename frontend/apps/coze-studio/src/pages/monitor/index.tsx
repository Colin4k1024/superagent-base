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
import { useState, useEffect } from 'react';

import {
  Layout,
  Header,
  HeaderTitle,
  HeaderActions,
  SubHeader,
  SubHeaderFilters,
  Content,
} from '@coze-studio/workspace-base/develop';
import { Button, Select, Spin } from '@coze-arch/coze-design';

import { adminApi, type AdminStatus } from '@/lib/superagent-api';
import MetricsPanel from '@/components/monitor/MetricsPanel';
import LogsPanel from '@/components/monitor/LogsPanel';

type TabKey = 'status' | 'metrics' | 'logs' | 'admin';

export default function MonitorPage() {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState<TabKey>('status');
  const [status, setStatus] = useState<AdminStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [reloading, setReloading] = useState(false);

  const loadStatus = () => {
    setLoading(true);
    adminApi
      .getStatus()
      .then(setStatus)
      .catch(() => setStatus(null))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    loadStatus();
  }, []);

  const handleReload = async () => {
    setReloading(true);
    try {
      await adminApi.reload();
      await loadStatus();
    } catch (err) {
      console.warn('reload failed', err);
    }
    setReloading(false);
  };

  const tabOptions = [
    { value: 'status', label: t('monitor.status', '状态') },
    { value: 'metrics', label: t('monitor.metrics', '指标') },
    { value: 'logs', label: t('monitor.logs', '日志') },
    { value: 'admin', label: t('monitor.admin', '管理') },
  ];

  return (
    <Layout>
      <Header>
        <HeaderTitle>
          <span>{t('nav.monitor', '监控')}</span>
        </HeaderTitle>
        <HeaderActions>
          <Button loading={reloading} onClick={handleReload}>
            {reloading
              ? t('monitor.reloading', '加载中...')
              : t('monitor.reload', '重新加载')}
          </Button>
        </HeaderActions>
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
        <Spin spinning={loading}>
          {activeTab === 'status' && (
            <div className="grid grid-cols-3 auto-rows-min gap-[20px] [@media(min-width:1600px)]:grid-cols-4">
              <StatusCard
                label="Backend"
                ok={!!status}
                detail={status ? 'Running' : 'Unreachable'}
              />
              <StatusCard label="MySQL" ok detail="Connected" />
              <StatusCard label="Redis" ok detail="Connected" />
              <StatusCard
                label={t('monitor.status', '状态')}
                ok={status?.health === 'ok'}
                detail={status?.health || '-'}
              />
              {status?.start_time ? (
                <div className="border border-[var(--semi-color-border)] rounded-[8px] p-[16px]">
                  <div className="text-[14px] font-[500] coz-fg-primary mb-[4px]">
                    {t('dashboard.startedAt', '启动于')}
                  </div>
                  <div className="text-[12px] coz-fg-secondary">
                    {new Date(status.start_time).toLocaleString()}
                  </div>
                </div>
              ) : null}
              {status?.uptime_seconds !== undefined && (
                <div className="border border-[var(--semi-color-border)] rounded-[8px] p-[16px]">
                  <div className="text-[14px] font-[500] coz-fg-primary mb-[4px]">
                    {t('dashboard.uptime', '运行时间')}
                  </div>
                  <div className="text-[12px] coz-fg-secondary">
                    {formatUptime(status.uptime_seconds)}
                  </div>
                </div>
              )}
            </div>
          )}

          {activeTab === 'metrics' && <MetricsPanel />}

          {activeTab === 'logs' && <LogsPanel />}

          {activeTab === 'admin' && (
            <div className="grid grid-cols-3 auto-rows-min gap-[20px] [@media(min-width:1600px)]:grid-cols-4">
              <div className="border border-[var(--semi-color-border)] rounded-[8px] p-[16px]">
                <div className="text-[14px] font-[500] coz-fg-primary mb-[4px]">
                  {t('monitor.reload', '重新加载')}
                </div>
                <div className="text-[12px] coz-fg-secondary mb-[12px]">
                  Reload agent configurations from disk
                </div>
                <Button size="small" onClick={handleReload} loading={reloading}>
                  {t('monitor.reload', '重新加载')}
                </Button>
              </div>
            </div>
          )}
        </Spin>
      </Content>
    </Layout>
  );
}

function StatusCard({
  label,
  ok,
  detail,
}: {
  label: string;
  ok: boolean;
  detail: string;
}) {
  return (
    <div className="border border-[var(--semi-color-border)] rounded-[8px] p-[16px]">
      <div className="flex items-center gap-[8px] mb-[8px]">
        <span
          className={`w-[8px] h-[8px] rounded-full ${ok ? 'bg-green-500' : 'bg-red-500'}`}
        />
        <span className="text-[14px] font-[500] coz-fg-primary">{label}</span>
      </div>
      <div className="text-[12px] coz-fg-secondary">{detail}</div>
    </div>
  );
}

const SECONDS_PER_MINUTE = 60;
const SECONDS_PER_HOUR = 3600;
const SECONDS_PER_DAY = 86400;

function formatUptime(seconds: number): string {
  if (seconds < SECONDS_PER_MINUTE) {
    return `${seconds}s`;
  }
  if (seconds < SECONDS_PER_HOUR) {
    return `${Math.floor(seconds / SECONDS_PER_MINUTE)}m`;
  }
  if (seconds < SECONDS_PER_DAY) {
    return `${Math.floor(seconds / SECONDS_PER_HOUR)}h ${Math.floor((seconds % SECONDS_PER_HOUR) / SECONDS_PER_MINUTE)}m`;
  }
  return `${Math.floor(seconds / SECONDS_PER_DAY)}d ${Math.floor((seconds % SECONDS_PER_DAY) / SECONDS_PER_HOUR)}h`;
}
