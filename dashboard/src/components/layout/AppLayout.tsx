import type { ReactNode } from 'react';
import Sidebar from './Sidebar';
import BottomNav from './BottomNav';

interface AppLayoutProps {
  children: ReactNode;
}

const AppLayout = ({ children }: AppLayoutProps) => {
  return (
    <div className="flex flex-col lg:flex-row min-h-screen bg-brand-50">
      <Sidebar />
      <main className="flex-1 overflow-auto">
        <div className="max-w-7xl mx-auto p-4 lg:p-8 w-full">
          {children}
        </div>
      </main>
      <BottomNav />
    </div>
  );
};

export default AppLayout;
