import client from './client';
import type { ApiResponse, TokenPair, User } from '../types';

export const authApi = {
  login: (username: string, password: string) =>
    client.post<ApiResponse<TokenPair>>('/api/v1/auth/login', { username, password }),

  register: (data: { username: string; email: string; password: string; role_id?: number }) =>
    client.post<ApiResponse<User>>('/api/v1/auth/register', data),

  refresh: (refresh_token: string) =>
    client.post<ApiResponse<TokenPair>>('/api/v1/auth/refresh', { refresh_token }),

  logout: (refresh_token: string) =>
    client.post('/api/v1/auth/logout', { refresh_token }),

  me: () => client.get<ApiResponse<User>>('/api/v1/auth/me'),

  changePassword: (old_password: string, new_password: string) =>
    client.put('/api/v1/auth/me/password', { old_password, new_password }),
};
