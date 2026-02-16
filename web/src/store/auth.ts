import { create } from 'zustand';
import type { User } from '../types';
import { authApi } from '../api/auth';

interface AuthState {
  user: User | null;
  token: string | null;
  loading: boolean;
  login: (username: string, password: string) => Promise<void>;
  logout: () => void;
  fetchUser: () => Promise<void>;
  init: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  token: localStorage.getItem('access_token'),
  loading: false,

  login: async (username, password) => {
    const res = await authApi.login(username, password);
    const { access_token, refresh_token } = res.data.data;
    localStorage.setItem('access_token', access_token);
    localStorage.setItem('refresh_token', refresh_token);
    set({ token: access_token });
    const me = await authApi.me();
    set({ user: me.data.data });
  },

  logout: () => {
    const rt = localStorage.getItem('refresh_token');
    if (rt) authApi.logout(rt).catch(() => {});
    localStorage.clear();
    set({ user: null, token: null });
  },

  fetchUser: async () => {
    try {
      set({ loading: true });
      const res = await authApi.me();
      set({ user: res.data.data });
    } catch {
      set({ user: null, token: null });
      localStorage.clear();
    } finally {
      set({ loading: false });
    }
  },

  init: () => {
    const token = localStorage.getItem('access_token');
    if (token) {
      set({ token });
      useAuthStore.getState().fetchUser();
    }
  },
}));
