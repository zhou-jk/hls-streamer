import client from './client';
import type { ApiResponse, User, Role } from '../types';

export const usersApi = {
  list: (page = 1, perPage = 20) =>
    client.get<ApiResponse<User[]>>('/api/v1/users', { params: { page, per_page: perPage } }),

  get: (id: number) =>
    client.get<ApiResponse<User>>(`/api/v1/users/${id}`),

  create: (data: { username: string; email: string; password: string; role_id: number }) =>
    client.post<ApiResponse<User>>('/api/v1/users', data),

  update: (id: number, data: Partial<{ username: string; email: string; role_id: number; is_active: boolean }>) =>
    client.put<ApiResponse<User>>(`/api/v1/users/${id}`, data),

  delete: (id: number) =>
    client.delete(`/api/v1/users/${id}`),

  listRoles: () =>
    client.get<ApiResponse<Role[]>>('/api/v1/roles'),
};
