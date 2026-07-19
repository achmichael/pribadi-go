import { useState } from 'react';
import { useAuthStore } from '../lib/auth';
import { api } from '../lib/api';
import { useNavigate } from 'react-router-dom';
import { Brain, Lock, User, ArrowRight, Loader2 } from 'lucide-react';

const Login = () => {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const setToken = useAuthStore((state) => state.setToken);
  const navigate = useNavigate();

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError('');
    
    try {
      const res = await api.post('/auth/login', { username, password });
      setToken(res.data.token);
      
      // Tahap 3: Logic onboarding check will be implemented here
      // For now, redirect to dashboard root
      navigate('/');
    } catch (err: any) {
      setError(err.response?.data?.error || 'Kredensial tidak valid. Silakan coba lagi.');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-canvas p-4">
      <div className="bg-surface p-8 sm:p-10 rounded-2xl shadow-sm border border-ink-primary/10 w-full max-w-[420px]">
        <div className="flex flex-col items-center mb-8">
          <div className="w-16 h-16 bg-brand-100 text-ink-muted rounded-2xl flex items-center justify-center mb-4 shadow-sm">
            <Brain size={32} />
          </div>
          <h1 className="font-sora text-2xl font-bold text-ink-primary">Control Panel AI</h1>
          <p className="text-ink-muted text-sm mt-2 text-center">
            Masuk untuk mengelola asisten AI pribadi Anda.
          </p>
        </div>

        {error && (
          <div className="bg-accent-danger/10 border border-rose-200 text-accent-danger px-4 py-3 rounded-lg mb-6 text-sm flex items-start gap-2">
            <span className="shrink-0 mt-0.5">⚠️</span>
            <span>{error}</span>
          </div>
        )}

        <form onSubmit={handleLogin} className="space-y-5">
          <div className="space-y-1.5">
            <label className="text-sm font-medium text-ink-primary ml-1">Username</label>
            <div className="relative">
              <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-ink-muted/70">
                <User size={18} />
              </div>
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="w-full pl-10 pr-4 py-2.5 bg-canvas border border-ink-primary/10 rounded-xl text-ink-primary placeholder-brand-400 focus:bg-surface focus:ring-2 focus:ring-accent-primary/15 focus:border-accent-primary transition-all outline-none"
                placeholder="Masukkan username Anda"
                required
              />
            </div>
          </div>

          <div className="space-y-1.5">
            <label className="text-sm font-medium text-ink-primary ml-1">Password</label>
            <div className="relative">
              <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-ink-muted/70">
                <Lock size={18} />
              </div>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full pl-10 pr-4 py-2.5 bg-canvas border border-ink-primary/10 rounded-xl text-ink-primary placeholder-brand-400 focus:bg-surface focus:ring-2 focus:ring-accent-primary/15 focus:border-accent-primary transition-all outline-none"
                placeholder="••••••••"
                required
              />
            </div>
          </div>

          <button
            type="submit"
            disabled={isLoading || !username || !password}
            className="w-full flex items-center justify-center gap-2 bg-accent-primary text-white rounded-xl py-3 px-4 font-medium hover:bg-accent-primary/90 active:scale-[0.98] transition-all disabled:opacity-70 disabled:cursor-not-allowed disabled:active:scale-100 shadow-sm mt-2"
          >
            {isLoading ? (
              <Loader2 size={20} className="animate-spin" />
            ) : (
              <>
                <span>Masuk ke Dashboard</span>
                <ArrowRight size={18} />
              </>
            )}
          </button>
        </form>
      </div>
    </div>
  );
};

export default Login;
