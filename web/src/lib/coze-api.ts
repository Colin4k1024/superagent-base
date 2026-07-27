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

/**
 * Coze Studio API client.
 * Wraps the original Coze Studio backend endpoints inherited by Superagent Base.
 * All requests go to /api which the Vite dev proxy forwards to localhost:8888.
 */

import { getToken, clearAuth } from './auth'

function cozeHeaders(): Record<string, string> {
  const token = getToken()
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (token) {
    headers['Access-Token'] = token
  }
  return headers
}

function handleAuthError(res: Response): void {
  if (res.status === 401 || res.status === 403) {
    clearAuth()
    window.location.href = '/login'
    throw new Error('Session expired')
  }
}

async function cozePost<T>(path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method: 'POST',
    headers: cozeHeaders(),
    body: body ? JSON.stringify(body) : undefined,
  })
  handleAuthError(res)
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  const data = await res.json()
  if (data.code !== undefined && data.code !== 0) {
    throw new Error(data.msg || data.message || `API error code ${data.code}`)
  }
  return data.data ?? data
}

async function cozeGet<T>(path: string): Promise<T> {
  const res = await fetch(path, { headers: cozeHeaders() })
  handleAuthError(res)
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  const data = await res.json()
  if (data.code !== undefined && data.code !== 0) {
    throw new Error(data.msg || data.message || `API error code ${data.code}`)
  }
  return data.data ?? data
}

// ─────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────

export interface CozeDataset {
  dataset_id: string
  name: string
  description?: string
  icon_url?: string
  document_count?: number
  create_time?: number
  update_time?: number
}

export interface CozeDocument {
  document_id: string
  dataset_id: string
  name: string
  type?: string
  status?: number
  slice_count?: number
  create_time?: number
  update_time?: number
}

export interface CozeSlice {
  slice_id: string
  document_id: string
  content: string
  status?: number
  create_time?: number
  update_time?: number
}

export interface CozePlugin {
  plugin_id: string
  name: string
  description?: string
  icon_url?: string
  status?: number
  api_count?: number
  create_time?: number
}

export interface CozePluginAPI {
  api_id: string
  plugin_id: string
  name: string
  description?: string
  method?: string
  url?: string
  parameters?: Record<string, unknown>
}

export interface CozeWorkflow {
  workflow_id: string
  name: string
  description?: string
  status?: number
  create_time?: number
  update_time?: number
}

export interface CozeBot {
  bot_id: string
  name: string
  description?: string
  icon_url?: string
  status?: number
  create_time?: number
  update_time?: number
}

export interface CozeProduct {
  product_id: string
  name: string
  description?: string
  icon_url?: string
  category?: string
  use_count?: number
}

export interface CozeConversation {
  id: string
  name?: string
  created_at?: number
}

export interface CozeMessage {
  id: string
  role: string
  content: string
  content_type?: string
  create_time?: number
}

export interface CozeDatabase {
  id: string
  name: string
  description?: string
  table_num?: number
  create_time?: number
}

// ─────────────────────────────────────────────────────────────
// Knowledge API  (/api/knowledge/*)
// ─────────────────────────────────────────────────────────────

export const knowledgeApi = {
  create: (params: { name: string; description?: string; icon?: string }) =>
    cozePost<{ dataset_id: string }>('/api/knowledge/create', params),

  list: (params?: { page?: number; size?: number }) =>
    cozePost<{ datasets: CozeDataset[]; total: number }>('/api/knowledge/list', params ?? {}),

  detail: (datasetId: string) =>
    cozePost<CozeDataset>('/api/knowledge/detail', { dataset_id: datasetId }),

  update: (params: { dataset_id: string; name?: string; description?: string }) =>
    cozePost<void>('/api/knowledge/update', params),

  delete: (datasetId: string) =>
    cozePost<void>('/api/knowledge/delete', { dataset_id: datasetId }),
}

export const knowledgeDocumentApi = {
  create: (params: { dataset_id: string; name: string; content?: string; file?: File }) =>
    cozePost<{ document_id: string }>('/api/knowledge/document/create', params),

  list: (params: { dataset_id: string; page?: number; size?: number }) =>
    cozePost<{ documents: CozeDocument[]; total: number }>('/api/knowledge/document/list', params),

  update: (params: { document_id: string; name?: string }) =>
    cozePost<void>('/api/knowledge/document/update', params),

  delete: (documentId: string) =>
    cozePost<void>('/api/knowledge/document/delete', { document_id: documentId }),

  resegment: (documentId: string) =>
    cozePost<void>('/api/knowledge/document/resegment', { document_id: documentId }),

  progress: (documentId: string) =>
    cozePost<{ progress: number; status: number }>('/api/knowledge/document/progress/get', { document_id: documentId }),
}

export const knowledgeSliceApi = {
  list: (params: { document_id: string; page?: number; size?: number }) =>
    cozePost<{ slices: CozeSlice[]; total: number }>('/api/knowledge/slice/list', params),

  create: (params: { document_id: string; content: string }) =>
    cozePost<{ slice_id: string }>('/api/knowledge/slice/create', params),

  update: (params: { slice_id: string; content: string }) =>
    cozePost<void>('/api/knowledge/slice/update', params),

  delete: (sliceId: string) =>
    cozePost<void>('/api/knowledge/slice/delete', { slice_id: sliceId }),
}

// ─────────────────────────────────────────────────────────────
// DraftBot / Intelligence API  (/api/draftbot/*, /api/intelligence_api/*)
// ─────────────────────────────────────────────────────────────

export const draftbotApi = {
  create: (params: { name: string; description?: string; icon?: string }) =>
    cozePost<{ bot_id: string }>('/api/draftbot/create', params),

  delete: (botId: string) =>
    cozePost<void>('/api/draftbot/delete', { bot_id: botId }),

  duplicate: (botId: string) =>
    cozePost<{ bot_id: string }>('/api/draftbot/duplicate', { bot_id: botId }),

  getDisplayInfo: (botId: string) =>
    cozePost<CozeBot>('/api/draftbot/get_display_info', { bot_id: botId }),

  updateDisplayInfo: (params: { bot_id: string; name?: string; description?: string; icon?: string }) =>
    cozePost<void>('/api/draftbot/update_display_info', params),

  publish: (params: { bot_id: string; connector_ids?: string[] }) =>
    cozePost<void>('/api/draftbot/publish', params),
}

export const intelligenceApi = {
  list: (params?: { page?: number; size?: number }) =>
    cozePost<{ intelligences: CozeBot[]; total: number }>(
      '/api/intelligence_api/search/get_draft_intelligence_list',
      params ?? {},
    ),

  getInfo: (botId: string) =>
    cozePost<CozeBot>('/api/intelligence_api/search/get_draft_intelligence_info', { bot_id: botId }),

  create: (params: { name: string; description?: string }) =>
    cozePost<{ project_id: string }>('/api/intelligence_api/draft_project/create', params),

  delete: (projectId: string) =>
    cozePost<void>('/api/intelligence_api/draft_project/delete', { project_id: projectId }),

  copy: (projectId: string) =>
    cozePost<{ project_id: string }>('/api/intelligence_api/draft_project/copy', { project_id: projectId }),

  publish: (params: { project_id: string; connector_ids?: string[] }) =>
    cozePost<void>('/api/intelligence_api/publish/publish_project', params),
}

// ─────────────────────────────────────────────────────────────
// Plugin API  (/api/plugin_api/*)
// ─────────────────────────────────────────────────────────────

export const pluginApi = {
  list: (params?: { page?: number; size?: number }) =>
    cozePost<{ plugins: CozePlugin[]; total: number }>(
      '/api/plugin_api/get_dev_plugin_list',
      params ?? {},
    ),

  getInfo: (pluginId: string) =>
    cozePost<CozePlugin>('/api/plugin_api/get_plugin_info', { plugin_id: pluginId }),

  getAPIs: (pluginId: string) =>
    cozePost<{ apis: CozePluginAPI[] }>('/api/plugin_api/get_plugin_apis', { plugin_id: pluginId }),

  createAPI: (params: { plugin_id: string; name: string; description?: string; method: string; url: string }) =>
    cozePost<{ api_id: string }>('/api/plugin_api/create_api', params),

  updateAPI: (params: { api_id: string; name?: string; description?: string; method?: string; url?: string }) =>
    cozePost<void>('/api/plugin_api/update_api', params),

  deleteAPI: (apiId: string) =>
    cozePost<void>('/api/plugin_api/delete_api', { api_id: apiId }),

  debugAPI: (params: { api_id: string; parameters?: Record<string, unknown> }) =>
    cozePost<{ response: unknown }>('/api/plugin_api/debug_api', params),

  register: (params: { name: string; description?: string; icon?: string }) =>
    cozePost<{ plugin_id: string }>('/api/plugin_api/register', params),

  delete: (pluginId: string) =>
    cozePost<void>('/api/plugin_api/del_plugin', { plugin_id: pluginId }),

  publish: (pluginId: string) =>
    cozePost<void>('/api/plugin_api/publish_plugin', { plugin_id: pluginId }),
}

// ─────────────────────────────────────────────────────────────
// Workflow API  (/api/workflow_api/*)
// ─────────────────────────────────────────────────────────────

export const workflowApi = {
  list: (params?: { page?: number; size?: number }) =>
    cozePost<{ workflows: CozeWorkflow[]; total: number }>(
      '/api/workflow_api/workflow_list',
      params ?? {},
    ),

  create: (params: { name: string; description?: string }) =>
    cozePost<{ workflow_id: string }>('/api/workflow_api/create', params),

  detail: (workflowId: string) =>
    cozePost<CozeWorkflow>('/api/workflow_api/workflow_detail', { workflow_id: workflowId }),

  save: (params: { workflow_id: string; schema: unknown }) =>
    cozePost<void>('/api/workflow_api/save', params),

  delete: (workflowId: string) =>
    cozePost<void>('/api/workflow_api/delete', { workflow_id: workflowId }),

  testRun: (params: { workflow_id: string; input?: Record<string, unknown> }) =>
    cozePost<{ execution_id: string; output?: unknown }>('/api/workflow_api/test_run', params),

  publish: (workflowId: string) =>
    cozePost<void>('/api/workflow_api/publish', { workflow_id: workflowId }),

  nodeTypes: () =>
    cozePost<{ node_types: unknown[] }>('/api/workflow_api/node_type', {}),
}

// ─────────────────────────────────────────────────────────────
// Marketplace API  (/api/marketplace/product/*)
// ─────────────────────────────────────────────────────────────

export const marketplaceApi = {
  list: (params?: { page?: number; size?: number; category?: string }) =>
    cozeGet<{ products: CozeProduct[]; total: number }>(
      `/api/marketplace/product/list${params ? '?' + new URLSearchParams(params as Record<string, string>) : ''}`,
    ),

  detail: (productId: string) =>
    cozeGet<CozeProduct>(`/api/marketplace/product/detail?product_id=${productId}`),

  search: (query: string) =>
    cozeGet<CozeProduct[]>(`/api/marketplace/product/search?q=${encodeURIComponent(query)}`),

  categories: () =>
    cozeGet<{ categories: { id: string; name: string }[] }>('/api/marketplace/product/category/list'),

  duplicate: (productId: string) =>
    cozePost<void>('/api/marketplace/product/duplicate', { product_id: productId }),
}

// ─────────────────────────────────────────────────────────────
// Conversation API  (/api/conversation/*)
// ─────────────────────────────────────────────────────────────

export const conversationApi = {
  create: (params?: { bot_id?: string; name?: string }) =>
    cozePost<{ id: string }>('/api/conversation/create', params ?? {}),

  list: () =>
    cozeGet<{ conversations: CozeConversation[] }>('/v1/conversations'),

  delete: (conversationId: string) =>
    fetch(`/v1/conversations/${conversationId}`, {
      method: 'DELETE',
      headers: cozeHeaders(),
    }).then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`) }),

  getMessageList: (params: { conversation_id: string; page?: number; size?: number }) =>
    cozePost<{ messages: CozeMessage[] }>('/api/conversation/get_message_list', params),

  clearHistory: (conversationId: string) =>
    cozePost<void>('/api/conversation/clear_message', { conversation_id: conversationId }),
}

// ─────────────────────────────────────────────────────────────
// Database API  (/api/memory/database/*)
// ─────────────────────────────────────────────────────────────

export const databaseApi = {
  list: (params?: { page?: number; size?: number }) =>
    cozePost<{ databases: CozeDatabase[]; total: number }>(
      '/api/memory/database/list',
      params ?? {},
    ),

  add: (params: { name: string; description?: string; table_id?: string }) =>
    cozePost<{ database_id: string }>('/api/memory/database/add', params),

  delete: (databaseId: string) =>
    cozePost<void>('/api/memory/database/delete', { database_id: databaseId }),

  getById: (databaseId: string) =>
    cozePost<CozeDatabase>('/api/memory/database/get_by_id', { database_id: databaseId }),

  update: (params: { database_id: string; name?: string; description?: string }) =>
    cozePost<void>('/api/memory/database/update', params),
}

// ─────────────────────────────────────────────────────────────
// Playground API  (/api/playground_api/*)
// ─────────────────────────────────────────────────────────────

export const playgroundApi = {
  getDraftBotInfo: (botId: string) =>
    cozePost<CozeBot>('/api/playground_api/draftbot/get_draft_bot_info', { bot_id: botId }),

  updateDraftBotInfo: (params: { bot_id: string; [key: string]: unknown }) =>
    cozePost<void>('/api/playground_api/draftbot/update_draft_bot_info', params),

  getSpaceList: () =>
    cozePost<{ spaces: { id: string; name: string }[] }>('/api/playground_api/space/list', {}),
}

// ─────────────────────────────────────────────────────────────
// User API  (/api/user/*, /api/passport/account/*)
// ─────────────────────────────────────────────────────────────

export const userApi = {
  getAccountInfo: () =>
    cozePost<{ user_id: string; name: string; email: string; avatar_url?: string }>(
      '/api/passport/account/info/v2/',
      {},
    ),

  updateProfile: (params: { name?: string; avatar_url?: string }) =>
    cozePost<void>('/api/user/update_profile', params),
}
