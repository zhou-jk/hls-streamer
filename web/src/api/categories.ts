import client from './client';
import type { ApiResponse, Category, Tag } from '../types';

export const categoriesApi = {
  list: () =>
    client.get<ApiResponse<Category[]>>('/api/v1/categories'),

  create: (data: { slug: string; parent_id?: number }) =>
    client.post<ApiResponse<Category>>('/api/v1/categories', data),

  update: (id: number, data: Partial<{ slug: string; parent_id: number; sort_order: number; is_active: boolean }>) =>
    client.put<ApiResponse<Category>>(`/api/v1/categories/${id}`, data),

  delete: (id: number) =>
    client.delete(`/api/v1/categories/${id}`),

  upsertTranslation: (id: number, lang: string, data: { name: string; description?: string }) =>
    client.put(`/api/v1/categories/${id}/translations/${lang}`, data),
};

export const tagsApi = {
  list: () =>
    client.get<ApiResponse<Tag[]>>('/api/v1/tags'),

  create: (data: { slug: string }) =>
    client.post<ApiResponse<Tag>>('/api/v1/tags', data),

  delete: (id: number) =>
    client.delete(`/api/v1/tags/${id}`),
};
