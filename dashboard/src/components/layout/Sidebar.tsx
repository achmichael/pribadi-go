import { NavLink } from 'react-router-dom';
import { 
  LayoutDashboard, 
  Smartphone, 
  Brain, 
  BookOpen, 
  Clock, 
  Database, 
  TrendingUp, 
  Settings,
  LogOut
} from 'lucide-react';
import { useAuthStore } from '../../lib/auth';
import { COPY } from '../../lib/copy';

const Sidebar = () => {
  const logout = useAuthStore((state) => state.logout);

  const mainLinks = [
    { to: '/', icon: LayoutDashboard, label: COPY.MENU_HOME },
    { to: '/connections', icon: Smartphone, label: COPY.MENU_CONNECTIONS },
    { to: '/personality', icon: Brain, label: COPY.MENU_PERSONALITY },
    { to: '/knowledge', icon: BookOpen, label: COPY.MENU_KNOWLEDGE },
  ];

  const advancedLinks = [
    { to: '/schedules', icon: Clock, label: COPY.MENU_SCHEDULES },
    { to: '/data-types', icon: Database, label: COPY.MENU_DATA_TYPES },
    { to: '/watchlist', icon: TrendingUp, label: COPY.MENU_WATCHLIST },
    { to: '/settings', icon: Settings, label: COPY.MENU_SETTINGS },
  ];

  const NavItem = ({ to, icon: Icon, label }: { to: string, icon: any, label: string }) => (
    <NavLink
      to={to}
      className={({ isActive }) =>
        `flex items-center gap-3 px-3 py-2 rounded-lg transition-colors ${
          isActive
            ? 'bg-emerald-50 text-emerald-700 font-medium'
            : 'text-brand-600 hover:bg-brand-100 hover:text-brand-900'
        }`
      }
    >
      <Icon size={20} />
      <span>{label}</span>
    </NavLink>
  );

  return (
    <aside className="hidden lg:flex w-64 bg-white border-r border-brand-200 h-full flex-col">
      <div className="p-6">
        <h1 className="text-xl font-bold text-brand-900">Control Panel</h1>
        <p className="text-sm text-brand-500 mt-1">AI Assistant</p>
      </div>
      
      <div className="flex-1 overflow-y-auto py-2 px-4 space-y-6">
        <div>
          <nav className="space-y-1">
            {mainLinks.map((link) => (
              <NavItem key={link.to} {...link} />
            ))}
          </nav>
        </div>

        <div>
          <div className="px-3 mb-2 text-xs font-semibold text-brand-400 uppercase tracking-wider">
            Lanjutan
          </div>
          <nav className="space-y-1">
            {advancedLinks.map((link) => (
              <NavItem key={link.to} {...link} />
            ))}
          </nav>
        </div>
      </div>
      
      <div className="p-4 border-t border-brand-200">
        <button
          onClick={logout}
          className="flex items-center gap-3 px-3 py-2 w-full text-left text-rose-600 hover:bg-rose-50 rounded-lg transition-colors"
        >
          <LogOut size={20} />
          <span>Keluar</span>
        </button>
      </div>
    </aside>
  );
};

export default Sidebar;
