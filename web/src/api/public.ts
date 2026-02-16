import axios from 'axios';
import type { ApiResponse, Video } from '../types';

const publicClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE || '',
  headers: { 'Content-Type': 'application/json' },
});

export const publicApi = {
  listVideos: (params?: { page?: number; per_page?: number; category?: number; q?: string }) =>
    publicClient.get<ApiResponse<Video[]>>('/api/v1/public/videos', { params }),

  getVideo: (uuid: string) =>
    publicClient.get<ApiResponse<Video>>(`/api/v1/public/videos/${uuid}`),
};
