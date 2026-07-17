import { useState } from 'react'
import { NavLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { handleLogout } from '@/lib/auth'
import {
  Bot, MessageSquare, Zap, Activity, BarChart3,
  Puzzle, Settings, Dna, Search, ChevronLeft,
  ChevronRight, LogOut, Globe, type LucideIcon,
} from 'lucide-react'

interface NavItem {
  to: string
  labelKey: string
  fallback: string
  icon: LucideIcon
}

interface NavGroup {
  titleKey: string
  fallback: string
  items: NavItem[]
}

const navGroups: NavGroup[] = [
  {
    titleKey: 'nav.group.core',
    fallback: 'Core',
    items: [
      { to: '/agents', labelKey: 'nav.agents', fallback: 'Agents', icon: Bot },
      { to: '/chat', labelKey: 'nav.chat', fallback: 'Chat', icon: MessageSquare },
      { to: '/agentloop-demo', labelKey: 'nav.agentloop', fallback: 'AgentLoop', icon: Zap },
    ],
  },
  {
    titleKey: 'nav.group.operations',
    fallback: 'Operations',
    items: [
      { to: '/monitor', labelKey: 'nav.monitor', fallback: 'Monitor', icon: Activity },
      { to: '/observability', labelKey: 'nav.observability', fallback: 'Observability', icon: BarChart3 },
    ],
  },
  {
    titleKey: 'nav.group.platform',
    fallback: 'Platform',
    items: [
      { to: '/skills', labelKey: 'nav.skills', fallback: 'Skills', icon: Puzzle },
      { to: '/settings', labelKey: 'nav.settings', fallback: 'Settings', icon: Settings },
      { to: '/evolution', labelKey: 'nav.evolution', fallback: 'Evolution', icon: Dna },
    ],
  },
]

export default function Sidebar() {
  const { t, i18n } = useTranslation()
  const [collapsed, setCollapsed] = useState(false)
  const [mobileOpen, setMobileOpen] = useState(false)
  const [showLogoutConfirm, setShowLogoutConfirm] = useState(false)

  async function doLogout() {
    await handleLogout()
    window.location.href = '/login'
  }

  function toggleLanguage() {
    const next = i18n.language === 'zh' ? 'en' : 'zh'
    i18n.changeLanguage(next)
    localStorage.setItem('language', next)
  }

  const w = collapsed ? 'w-[64px]' : 'w-[260px]'
  const mobileW = mobileOpen ? 'w-[260px]' : 'w-0 md:w-[260px]'

  return (
    <>
      {/* Mobile overlay */}
      {mobileOpen && (
        <div
          className="fixed inset-0 bg-black/40 z-30 md:hidden"
          onClick={() => setMobileOpen(false)}
        />
      )}

      <nav
        className={`
          flex flex-col h-screen bg-surface-sidebar text-slate-300
          transition-all duration-200 ease-in-out z-40
          ${w} ${mobileW}
          fixed md:relative
          border-r border-white/5
        `}
      >
        {/* Logo */}
        <div className="flex items-center gap-3 px-4 h-16 shrink-0 border-b border-white/5">
          <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-blue-500 to-violet-500 flex items-center justify-center shrink-0">
            <Bot className="w-5 h-5 text-white" />
          </div>
          {!collapsed && (
            <div className="flex flex-col min-w-0">
              <span className="text-sm font-semibold text-white truncate">Superagent</span>
              <span className="text-[10px] text-slate-500">Base</span>
            </div>
          )}
        </div>

        {/* Search (placeholder) */}
        {!collapsed && (
          <div className="px-3 py-3">
            <div className="flex items-center gap-2 px-3 py-2 rounded-lg bg-white/5 text-slate-500 text-sm cursor-pointer hover:bg-white/10 transition-colors">
              <Search className="w-4 h-4" />
              <span>Search...</span>
              <kbd className="ml-auto text-[10px] bg-white/10 px-1.5 py-0.5 rounded font-mono">⌘K</kbd>
            </div>
          </div>
        )}

        {/* Navigation groups */}
        <div className="flex-1 overflow-y-auto px-3 py-2 space-y-4">
          {navGroups.map((group) => (
            <div key={group.titleKey}>
              {!collapsed && (
                <p className="px-3 mb-1.5 text-[10px] font-semibold uppercase tracking-wider text-slate-600">
                  {t(group.titleKey, group.fallback)}
                </p>
              )}
              <ul className="space-y-0.5">
                {group.items.map(({ to, labelKey, fallback, icon: Icon }) => (
                  <li key={to}>
                    <NavLink
                      to={to}
                      onClick={() => setMobileOpen(false)}
                      className={({ isActive }) =>
                        `group flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-all duration-150 relative
                        ${isActive
                          ? 'bg-white/10 text-white'
                          : 'text-slate-400 hover:bg-white/5 hover:text-slate-200'
                        }`
                      }
                    >
                      {({ isActive }) => (
                        <>
                          {isActive && (
                            <div className="absolute left-0 top-1/2 -translate-y-1/2 w-[3px] h-5 bg-blue-500 rounded-r-full" />
                          )}
                          <Icon className={`w-[18px] h-[18px] shrink-0 ${isActive ? 'text-blue-400' : ''}`} />
                          {!collapsed && (
                            <span className="truncate">{t(labelKey, fallback)}</span>
                          )}
                        </>
                      )}
                    </NavLink>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        {/* Footer */}
        <div className="shrink-0 px-3 py-3 border-t border-white/5 space-y-1">
          <button
            onClick={toggleLanguage}
            className="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-slate-500 hover:bg-white/5 hover:text-slate-300 transition-colors"
          >
            <Globe className="w-[18px] h-[18px] shrink-0" />
            {!collapsed && <span>{i18n.language === 'zh' ? '中文' : 'English'}</span>}
          </button>

          <button
            onClick={() => setShowLogoutConfirm(true)}
            className="w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-slate-500 hover:bg-white/5 hover:text-red-400 transition-colors"
          >
            <LogOut className="w-[18px] h-[18px] shrink-0" />
            {!collapsed && <span>{t('nav.logout', 'Logout')}</span>}
          </button>

          <button
            onClick={() => setCollapsed((c) => !c)}
            className="hidden md:flex w-full items-center justify-center py-2 rounded-lg text-slate-600 hover:bg-white/5 hover:text-slate-400 transition-colors"
          >
            {collapsed ? <ChevronRight className="w-4 h-4" /> : <ChevronLeft className="w-4 h-4" />}
          </button>
        </div>
      </nav>

      {/* Mobile hamburger */}
      <button
        onClick={() => setMobileOpen((o) => !o)}
        className="fixed top-4 left-4 z-50 md:hidden p-2 rounded-lg bg-surface-sidebar text-white shadow-elevated"
        aria-label="Toggle menu"
      >
        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M4 6h16M4 12h16M4 18h16" />
        </svg>
      </button>

      {/* Logout confirmation */}
      {showLogoutConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 animate-fade-in">
          <div className="bg-white rounded-card-lg shadow-elevated p-6 w-80 mx-4 animate-scale-in">
            <h3 className="text-lg font-semibold text-gray-900 mb-2">{t('nav.logoutConfirmTitle', 'Confirm Logout')}</h3>
            <p className="text-sm text-gray-600 mb-6">{t('nav.logoutConfirmMessage', 'Are you sure you want to log out?')}</p>
            <div className="flex gap-3 justify-end">
              <button
                onClick={() => setShowLogoutConfirm(false)}
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 transition-colors"
              >
                {t('common.cancel', 'Cancel')}
              </button>
              <button
                onClick={doLogout}
                className="px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors"
              >
                {t('common.confirm', 'Confirm')}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
