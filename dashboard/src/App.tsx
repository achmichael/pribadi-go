import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useAuthStore } from './lib/auth';
import Login from './pages/Login';
import Overview from './pages/Overview';
import AgentConfig from './pages/AgentConfig';
import SchemaList from './pages/CustomEntities/SchemaList';
import SchemaBuilder from './pages/CustomEntities/SchemaBuilder';
import EntityRecords from './pages/CustomEntities/EntityRecords';
import CronJobList from './pages/CronJobs/JobList';
import CronJobForm from './pages/CronJobs/JobForm';
import StockWatchlist from './pages/StockWatchlist';
import AppLayout from './components/layout/AppLayout';

// Placeholders
import Connections from './pages/Connections';
import Knowledge from './pages/Knowledge';
import SettingsPage from './pages/Settings';
import Onboarding from './pages/Onboarding';

const queryClient = new QueryClient();

const ProtectedRoute = ({ children }: { children: React.ReactNode }) => {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated());
  if (!isAuthenticated) {
    return <Navigate to="/login" />;
  }
  return (
    <AppLayout>
      {children}
    </AppLayout>
  );
};

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          {/* Auth & Onboarding */}
          <Route path="/login" element={<Login />} />
          <Route path="/onboarding" element={<Onboarding />} />
          
          {/* Main App */}
          <Route path="/" element={<ProtectedRoute><Overview /></ProtectedRoute>} />
          <Route path="/connections" element={<ProtectedRoute><Connections /></ProtectedRoute>} />
          <Route path="/personality" element={<ProtectedRoute><AgentConfig /></ProtectedRoute>} />
          <Route path="/knowledge" element={<ProtectedRoute><Knowledge /></ProtectedRoute>} />
          
          {/* Advanced / Features */}
          <Route path="/schedules" element={<ProtectedRoute><CronJobList /></ProtectedRoute>} />
          <Route path="/schedules/new" element={<ProtectedRoute><CronJobForm /></ProtectedRoute>} />
          <Route path="/schedules/:id/edit" element={<ProtectedRoute><CronJobForm /></ProtectedRoute>} />
          
          <Route path="/data-types" element={<ProtectedRoute><SchemaList /></ProtectedRoute>} />
          <Route path="/data-types/new" element={<ProtectedRoute><SchemaBuilder /></ProtectedRoute>} />
          <Route path="/data-types/:id/edit" element={<ProtectedRoute><SchemaBuilder /></ProtectedRoute>} />
          <Route path="/data-types/:id/records" element={<ProtectedRoute><EntityRecords /></ProtectedRoute>} />
          
          <Route path="/watchlist" element={<ProtectedRoute><StockWatchlist /></ProtectedRoute>} />
          <Route path="/settings" element={<ProtectedRoute><SettingsPage /></ProtectedRoute>} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}

export default App;
