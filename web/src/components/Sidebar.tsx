import { useState } from 'react'
import { NavLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { logout } from '@/lib/iam'

export default function Sidebar() {
  const { t, i18n } = useTranslation()
  const [mobileOpen, setMobileOpen] = useState(false)

  function handleLogout() {
    logout()
  }

  function toggleLanguage() {
    const next = i18n.language === 'zh' ? 'en' : 'zh'
    i18n.changeLanguage(next)
    localStorage.setItem('language', next)
  }

  const navItems = [
    { to: '/agents', label: t('nav.agents'), icon: '🤖' },
    { to: '/chat', label: t('nav.chat'), icon: '💬' },
    { to: '/monitor', label: t('nav.monitor'), icon: '📊' },
    { to: '/skills', label: t('nav.skills'), icon: '🧩' },
    { to: '/settings', label: t('nav.settings'), icon: '⚙️' },
    { to: '/evolution', label: t('nav.evolution'), icon: '◈' },
    { to: '/xiaohai-debug', label: '小海调试', icon: '🔌' },
  ]

  // On mobile: collapsed (w-16) unless mobileOpen; on md+: always w-56
  const sidebarWidth = mobileOpen ? 'w-56' : 'w-16 md:w-56'

  return (
    <nav className={`flex flex-col min-h-screen bg-gray-900 text-gray-100 py-4 transition-all duration-200 ${sidebarWidth}`}>
      {/* Header: logo + hamburger toggle */}
      <div className="px-4 pb-6 flex items-center justify-between">
        <span className="text-xl font-bold tracking-tight">
          <span className="block md:hidden">{mobileOpen ? 'SA' : 'SA'}</span>
          <span className="hidden md:block">Superagent</span>
        </span>
        {/* Hamburger: visible only on small screens */}
        <button
          onClick={() => setMobileOpen((prev) => !prev)}
          className="block md:hidden p-1 rounded text-gray-400 hover:text-white hover:bg-gray-800 transition-colors"
          aria-label={mobileOpen ? 'Collapse menu' : 'Expand menu'}
        >
          {mobileOpen ? (
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          ) : (
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M4 6h16M4 12h16M4 18h16" />
            </svg>
          )}
        </button>
      </div>

      <ul className="flex-1 space-y-1 px-2">
        {navItems.map(({ to, label, icon }) => (
          <li key={to}>
            <NavLink
              to={to}
              onClick={() => setMobileOpen(false)}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium transition-colors ${
                  isActive
                    ? 'bg-gray-700 text-white'
                    : 'text-gray-300 hover:bg-gray-800 hover:text-white'
                }`
              }
            >
              <span className="shrink-0">{icon}</span>
              {/* Label: hidden on collapsed mobile, visible when open or on md+ */}
              <span className={`${mobileOpen ? 'block' : 'hidden md:block'} truncate`}>
                {label}
              </span>
            </NavLink>
          </li>
        ))}
      </ul>

      <div className="px-4 pt-4 border-t border-gray-700 space-y-3">
        {/* Language toggle */}
        <button
          onClick={toggleLanguage}
          className="w-full flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium text-gray-400 hover:bg-gray-800 hover:text-white transition-colors"
          title={i18n.language === 'zh' ? 'Switch to English' : '切换为中文'}
        >
          <span className="shrink-0">🌐</span>
          <span className={`${mobileOpen ? 'block' : 'hidden md:block'} truncate`}>
            {i18n.language === 'zh' ? '中文' : 'EN'}
          </span>
        </button>

        <button
          onClick={handleLogout}
          className="w-full flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium text-gray-400 hover:bg-gray-800 hover:text-white transition-colors"
        >
          <span className="shrink-0">🚪</span>
          <span className={`${mobileOpen ? 'block' : 'hidden md:block'} truncate`}>
            {t('nav.logout')}
          </span>
        </button>
        <p className={`text-xs text-gray-500 px-3 ${mobileOpen ? 'block' : 'hidden md:block'}`}>
          {t('app.version')}
        </p>
      </div>
    </nav>
  )
}
