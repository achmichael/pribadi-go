import { create } from 'zustand';
import { fetchApi } from '@/lib/api';

export interface MonitorTask {
  id: number;
  user_id: string;
  platform: string;
  category?: string;
  name: string;
  source_type: 'http_api' | 'web_scrape';
  source_config: string;
  condition_mode: 'structural' | 'nl_judge';
  operator?: string;
  condition_value?: number;
  condition_text?: string;
  condition_prompt?: string;
  poll_interval_seconds: number;
  last_value?: number;
  last_value_text?: string;
  last_checked_at?: string;
  status: string;
}

interface MonitorStore {
  tasks: MonitorTask[];
  isLoading: boolean;
  error: string | null;
  fetchTasks: () => Promise<void>;
  createTask: (task: Partial<MonitorTask>) => Promise<void>;
  deleteTask: (id: number) => Promise<void>;
  testExtract: (config: any) => Promise<any>;
  testDeriveRule: (url: string, description: string) => Promise<any>;
}

export const useMonitorStore = create<MonitorStore>((set) => ({
  tasks: [],
  isLoading: false,
  error: null,

  fetchTasks: async () => {
    set({ isLoading: true, error: null });
    try {
      const tasks = await fetchApi('/monitors');
      set({ tasks, isLoading: false });
    } catch (err: any) {
      set({ error: err.message, isLoading: false });
    }
  },

  createTask: async (task) => {
    set({ isLoading: true, error: null });
    try {
      await fetchApi('/monitors', { method: 'POST', body: JSON.stringify(task) });
      const tasks = await fetchApi('/monitors');
      set({ tasks, isLoading: false });
    } catch (err: any) {
      set({ error: err.message, isLoading: false });
    }
  },

  deleteTask: async (id) => {
    set({ isLoading: true, error: null });
    try {
      await fetchApi(`/monitors/${id}`, { method: 'DELETE' });
      set((state) => ({
        tasks: state.tasks.filter((t) => t.id !== id),
        isLoading: false,
      }));
    } catch (err: any) {
      set({ error: err.message, isLoading: false });
    }
  },

  testExtract: async (config) => {
    return await fetchApi('/monitors/test-extract', { method: 'POST', body: JSON.stringify(config) });
  },

  testDeriveRule: async (url, description) => {
    return await fetchApi('/monitors/test-derive-rule', { method: 'POST', body: JSON.stringify({ url, description }) });
  },
}));