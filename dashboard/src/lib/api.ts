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
    throw new Error(await res.text());
  }
  return res.json();
};
