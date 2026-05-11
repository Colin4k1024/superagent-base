// TypeScript types matching the backend proto/API contracts.

export interface Agent {
  id: string
  name: string
  description: string
  status: 'active' | 'inactive' | 'error'
  model: string
  createdAt: string
  updatedAt: string
}

export interface Message {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  createdAt: string
}

export interface ChatSession {
  id: string
  agentId: string
  messages: Message[]
  createdAt: string
}

export interface Metric {
  name: string
  value: number
  unit: string
  timestamp: string
}

export interface Skill {
  name: string
  version: string
  author: string
  description: string
  tags: string[]
  status: 'installed' | 'available'
}

export interface ModelConfig {
  provider: string
  model: string
  apiKey?: string
  baseUrl?: string
  maxTokens?: number
  temperature?: number
}

export interface MCPServer {
  id: string
  name: string
  url: string
  status: 'connected' | 'disconnected' | 'error'
}

export interface ApiError {
  code: number
  message: string
}

export type ApiResult<T> = { data: T; error: null } | { data: null; error: ApiError }
