import { NavLink } from 'react-router-dom'

const navItems = [
  { to: '/agents', label: 'Agents', icon: '🤖' },
  { to: '/chat', label: 'Chat', icon: '💬' },
  { to: '/monitor', label: 'Monitor', icon: '📊' },
  { to: '/skills', label: 'Skills', icon: '🧩' },
  { to: '/settings', label: 'Settings', icon: '⚙️' },
]

export default function Sidebar() {
  return (
    <nav className="flex flex-col w-56 min-h-screen bg-gray-900 text-gray-100 py-4">
      <div className="px-4 pb-6">
        <span className="text-xl font-bold tracking-tight">Superagent</span>
      </div>

      <ul className="flex-1 space-y-1 px-2">
        {navItems.map(({ to, label, icon }) => (
          <li key={to}>
            <NavLink
              to={to}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium transition-colors ${
                  isActive
                    ? 'bg-gray-700 text-white'
                    : 'text-gray-300 hover:bg-gray-800 hover:text-white'
                }`
              }
            >
              <span>{icon}</span>
              {label}
            </NavLink>
          </li>
        ))}
      </ul>

      <div className="px-4 pt-4 border-t border-gray-700 text-xs text-gray-500">
        v0.1.0
      </div>
    </nav>
  )
}
