import { create } from 'zustand';

interface AuthState {
  token: string | null;
  setToken: (token: string) => void;
  logout: () => void;
  isAuthenticated: () => boolean;
}

export const useAuthStore = create<AuthState>((set, get) => ({
  token: localStorage.getItem('dashboard_token'),
  setToken: (token: string) => {
    localStorage.setItem('dashboard_token', token);
    set({ token });
  },
  logout: () => {
    localStorage.removeItem('dashboard_token');
    set({ token: null });
  },
  isAuthenticated: () => !!get().token,
}));
