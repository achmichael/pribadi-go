import { NavLink } from 'react-router-dom';
import { Settings, Database, Clock, TrendingUp, LogOut, LayoutDashboard } from 'lucide-react';
import { useAuthStore } from '../../lib/auth';

const Sidebar = () => {
  const logout = useAuthStore((state) => state.logout);

  const links = [
    { to: '/', icon: LayoutDashboard, label: 'Overview' },
    { to: '/config', icon: Settings, label: 'Agent Config' },
    { to: '/entities', icon: Database, label: 'Custom Entities' },
    { to: '/cron', icon: Clock, label: 'Cron Jobs' },
    { to: '/stocks', icon: TrendingUp, label: 'Stock Watchlist' },
  ];

  return (
    <div className="w-64 bg-white border-r h-full flex flex-col">
      <div className="p-6 border-b">
        <h1 className="text-xl font-bold text-slate-800">AI Control Panel</h1>
      </div>
      <nav className="flex-1 p-4 space-y-1">
        {links.map((link) => {
          const Icon = link.icon;
          return (
            <NavLink
              key={link.to}
              to={link.to}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2 rounded-md transition-colors ${
                  isActive
                    ? 'bg-blue-50 text-blue-700 font-medium'
                    : 'text-slate-600 hover:bg-slate-50 hover:text-slate-900'
                }`
              }
            >
              <Icon size={18} />
              {link.label}
            </NavLink>
          );
        })}
      </nav>
      <div className="p-4 border-t">
        <button
          onClick={logout}
          className="flex items-center gap-3 px-3 py-2 w-full text-left text-red-600 hover:bg-red-50 rounded-md transition-colors"
        >
          <LogOut size={18} />
          Logout
        </button>
      </div>
    </div>
  );
};

export default Sidebar;
