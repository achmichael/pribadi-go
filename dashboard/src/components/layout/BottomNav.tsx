import { NavLink } from 'react-router-dom';
import { 
  LayoutDashboard, 
  Smartphone, 
  Brain, 
  BookOpen, 
  Menu
} from 'lucide-react';
import { COPY } from '../../lib/copy';
import { useState } from 'react';

const BottomNav = () => {
  const [isMoreOpen, setIsMoreOpen] = useState(false);

  const mainLinks = [
    { to: '/', icon: LayoutDashboard, label: COPY.MENU_HOME },
    { to: '/connections', icon: Smartphone, label: 'Koneksi' }, // Shortened for mobile
    { to: '/personality', icon: Brain, label: 'Sifat AI' },
    { to: '/knowledge', icon: BookOpen, label: 'Ilmu' },
  ];

  return (
    <>
      {/* Spacer for bottom nav */}
      <div className="h-16 lg:hidden" />
      
      <nav className="lg:hidden fixed bottom-0 left-0 right-0 bg-surface border-t border-ink-primary/10 flex items-center justify-around z-50 px-2 pb-safe">
        {mainLinks.map((link) => {
          const Icon = link.icon;
          return (
            <NavLink
              key={link.to}
              to={link.to}
              className={({ isActive }) =>
                `flex flex-col items-center justify-center w-full h-16 space-y-1 transition-colors ${
                  isActive
                    ? 'text-accent-primary'
                    : 'text-ink-muted hover:text-ink-primary'
                }`
              }
            >
              <Icon size={20} />
              <span className="text-[10px] font-medium">{link.label}</span>
            </NavLink>
          );
        })}
        
        {/* More Button */}
        <button 
          className={`flex flex-col items-center justify-center w-full h-16 space-y-1 transition-colors ${isMoreOpen ? 'text-accent-primary' : 'text-ink-muted'}`}
          onClick={() => setIsMoreOpen(!isMoreOpen)}
        >
          <Menu size={20} />
          <span className="text-[10px] font-medium">Lainnya</span>
        </button>
      </nav>

      {/* Mobile More Menu Overlay (Placeholder for now, will implement properly in Phase 2/11) */}
      {isMoreOpen && (
        <div className="lg:hidden fixed inset-0 z-40 bg-black/50" onClick={() => setIsMoreOpen(false)}>
          <div className="absolute bottom-16 left-0 right-0 bg-surface rounded-t-xl p-4 shadow-lg animate-in slide-in-from-bottom-2">
            <p className="text-center text-ink-muted text-sm py-4">Menu lanjutan akan muncul di sini (Tahap selanjutnya)</p>
          </div>
        </div>
      )}
    </>
  );
};

export default BottomNav;
