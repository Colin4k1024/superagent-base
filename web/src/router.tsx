import { createBrowserRouter, Navigate } from 'react-router-dom'
import Layout from './components/Layout'
import { AuthGuard } from './components/AuthGuard'
import LoginPage from './pages/LoginPage'
import AgentsPage from './pages/AgentsPage'
import AgentEditPage from './pages/AgentEditPage'
import WorkflowEditorPage from './pages/WorkflowEditorPage'
import ChatPage from './pages/ChatPage'
import MonitorPage from './pages/MonitorPage'
import SkillsPage from './pages/SkillsPage'
import SettingsPage from './pages/SettingsPage'
import EvolutionPage from './pages/EvolutionPage'
import XiaohaiDebugPage from './pages/XiaohaiDebugPage'

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  {
    element: <AuthGuard />,
    children: [
      {
        path: '/',
        element: <Layout />,
        children: [
          { index: true, element: <Navigate to="/agents" replace /> },
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
          { path: 'xiaohai-debug', element: <XiaohaiDebugPage /> },
        ],
      },
    ],
  },
])
