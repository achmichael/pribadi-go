import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useAuthStore } from './lib/auth';
import Sidebar from './components/layout/Sidebar';
import Login from './pages/Login';
import Overview from './pages/Overview';
import AgentConfig from './pages/AgentConfig';
import SchemaList from './pages/CustomEntities/SchemaList';
import SchemaBuilder from './pages/CustomEntities/SchemaBuilder';
import EntityRecords from './pages/CustomEntities/EntityRecords';
import CronJobList from './pages/CronJobs/JobList';
import CronJobForm from './pages/CronJobs/JobForm';
import StockWatchlist from './pages/StockWatchlist';

const queryClient = new QueryClient();

const ProtectedRoute = ({ children }: { children: React.ReactNode }) => {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated());
  if (!isAuthenticated) {
    return <Navigate to="/login" />;
  }
  return (
    <div className="flex h-screen bg-slate-50">
      <Sidebar />
      <div className="flex-1 overflow-auto p-8">
        {children}
      </div>
    </div>
  );
};

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          
          <Route path="/" element={<ProtectedRoute><Overview /></ProtectedRoute>} />
          <Route path="/config" element={<ProtectedRoute><AgentConfig /></ProtectedRoute>} />
          
          <Route path="/entities" element={<ProtectedRoute><SchemaList /></ProtectedRoute>} />
          <Route path="/entities/new" element={<ProtectedRoute><SchemaBuilder /></ProtectedRoute>} />
          <Route path="/entities/:id/edit" element={<ProtectedRoute><SchemaBuilder /></ProtectedRoute>} />
          <Route path="/entities/:id/records" element={<ProtectedRoute><EntityRecords /></ProtectedRoute>} />
          
          <Route path="/cron" element={<ProtectedRoute><CronJobList /></ProtectedRoute>} />
          <Route path="/cron/new" element={<ProtectedRoute><CronJobForm /></ProtectedRoute>} />
          <Route path="/cron/:id/edit" element={<ProtectedRoute><CronJobForm /></ProtectedRoute>} />
          
          <Route path="/stocks" element={<ProtectedRoute><StockWatchlist /></ProtectedRoute>} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}

export default App;
