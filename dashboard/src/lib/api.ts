export const API_BASE = process.env.NODE_ENV === 'development' 
  ? 'http://localhost:8090/api/v1' 
  : '/api/v1';

export const getAuthHeader = () => {
  // In a real app, get from localStorage or cookies
  // For now using mock admin token if exists, or nothing
  const token = typeof window !== 'undefined' ? localStorage.getItem('token') : null;
  return token ? { 'Authorization': `Bearer ${token}` } : {};
};

export const fetchApi = async (endpoint: string, options: RequestInit = {}) => {
  const headers = {
    'Content-Type': 'application/json',
    ...getAuthHeader(),
    ...options.headers,
  };

  const res = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers,
  });

  if (!res.ok) {
    if (res.status === 401 || res.status === 403) {
      if (typeof window !== 'undefined') {
        localStorage.removeItem('token');
        // Prevent redirect loop if already on login page
        if (window.location.pathname !== '/login') {
          window.location.href = '/login';
        }
      }
    }
    throw new Error(await res.text());
  }
  return res.json();
};
