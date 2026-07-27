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

import { createBrowserRouter } from 'react-router-dom'
import Layout from './components/Layout'
import { AuthGuard } from './components/AuthGuard'
import LoginPage from './pages/LoginPage'
import DashboardPage from './pages/DashboardPage'
import AgentsPage from './pages/AgentsPage'
import AgentEditPage from './pages/AgentEditPage'
import WorkflowEditorPage from './pages/WorkflowEditorPage'
import ChatPage from './pages/ChatPage'
import MonitorPage from './pages/MonitorPage'
import SkillsPage from './pages/SkillsPage'
import SettingsPage from './pages/SettingsPage'
import EvolutionPage from './pages/EvolutionPage'
import ObservabilityPage from './pages/ObservabilityPage'
import AgentLoopChatPage from './pages/AgentLoopChatPage'

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  {
    element: <AuthGuard />,
    children: [
      {
        path: '/',
        element: <Layout />,
        children: [
          { index: true, element: <DashboardPage /> },
          { path: 'agents', element: <AgentsPage /> },
          { path: 'agents/new', element: <AgentEditPage /> },
          { path: 'agents/:name/edit', element: <AgentEditPage /> },
          { path: 'agents/:name/workflow', element: <WorkflowEditorPage /> },
          { path: 'chat', element: <ChatPage /> },
          { path: 'chat/:sessionId', element: <ChatPage /> },
          { path: 'monitor', element: <MonitorPage /> },
          { path: 'skills', element: <SkillsPage /> },
          { path: 'settings', element: <SettingsPage /> },
          { path: 'evolution', element: <EvolutionPage /> },
          { path: 'observability', element: <ObservabilityPage /> },
          { path: 'agentloop-demo', element: <AgentLoopChatPage /> },
        ],
      },
    ],
  },
])
