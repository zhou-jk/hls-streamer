import client from './client';
import type { ApiResponse, Video, VideoTranslation, VideoVariant, Thumbnail, Subtitle, VideoCast, TranscodeTask } from '../types';

export const videosApi = {
  list: (params?: { page?: number; per_page?: number; status?: string; category?: number; q?: string }) =>
    client.get<ApiResponse<Video[]>>('/api/v1/videos', { params }),

  get: (uuid: string) =>
    client.get<ApiResponse<Video>>(`/api/v1/videos/${uuid}`),

  create: (data: { slug?: string; original_filename?: string; language: string; title: string; description?: string; synopsis?: string; rating?: string; release_date?: string }) =>
    client.post<ApiResponse<Video>>('/api/v1/videos', data),

  update: (uuid: string, data: Partial<{ slug: string; rating: string; release_date: string; has_drm: boolean; is_public: boolean; sort_order: number }>) =>
    client.put<ApiResponse<Video>>(`/api/v1/videos/${uuid}`, data),

  delete: (uuid: string) =>
    client.delete(`/api/v1/videos/${uuid}`),

  restore: (uuid: string) =>
    client.post(`/api/v1/videos/${uuid}/restore`),

  // Translations
  listTranslations: (uuid: string) =>
    client.get<ApiResponse<VideoTranslation[]>>(`/api/v1/videos/${uuid}/translations`),

  upsertTranslation: (uuid: string, lang: string, data: { title: string; description?: string; synopsis?: string; seo_title?: string; seo_description?: string }) =>
    client.put(`/api/v1/videos/${uuid}/translations/${lang}`, data),

  deleteTranslation: (uuid: string, lang: string) =>
    client.delete(`/api/v1/videos/${uuid}/translations/${lang}`),

  // Upload
  initiateUpload: (uuid: string, data: { filename: string; content_type: string; part_count: number }) =>
    client.post(`/api/v1/videos/${uuid}/upload/initiate`, data),

  completeUpload: (uuid: string, data: Record<string, unknown>) =>
    client.post(`/api/v1/videos/${uuid}/upload/complete`, data),

  // Transcode
  startTranscode: (uuid: string, data: { resolutions: Array<{ name: string; width: number; height: number; bitrate_kbps: number; audio_bitrate_kbps?: number }>; codec?: string; drm?: boolean }) =>
    client.post(`/api/v1/videos/${uuid}/transcode`, data),

  listTasks: (uuid: string) =>
    client.get(`/api/v1/videos/${uuid}/tasks`),

  getTask: (uuid: string, taskUuid: string) =>
    client.get(`/api/v1/videos/${uuid}/tasks/${taskUuid}`),

  // Variants
  listVariants: (uuid: string) =>
    client.get<ApiResponse<VideoVariant[]>>(`/api/v1/videos/${uuid}/variants`),

  deleteVariants: (uuid: string) =>
    client.delete(`/api/v1/videos/${uuid}/variants`),

  deleteVariant: (uuid: string, id: number) =>
    client.delete(`/api/v1/videos/${uuid}/variants/${id}`),

  // Thumbnails
  listThumbnails: (uuid: string) =>
    client.get<ApiResponse<Thumbnail[]>>(`/api/v1/videos/${uuid}/thumbnails`),

  generateThumbnails: (uuid: string, data: { count?: number; width?: number }) =>
    client.post(`/api/v1/videos/${uuid}/thumbnails/generate`, data),

  setDefaultThumbnail: (uuid: string, id: number) =>
    client.put(`/api/v1/videos/${uuid}/thumbnails/${id}/default`),

  deleteThumbnail: (uuid: string, id: number) =>
    client.delete(`/api/v1/videos/${uuid}/thumbnails/${id}`),

  // Subtitles
  listSubtitles: (uuid: string) =>
    client.get<ApiResponse<Subtitle[]>>(`/api/v1/videos/${uuid}/subtitles`),

  uploadSubtitle: (uuid: string, data: FormData) =>
    client.post(`/api/v1/videos/${uuid}/subtitles/upload`, data, {
      headers: { 'Content-Type': 'multipart/form-data' },
    }),

  deleteSubtitle: (uuid: string, id: number) =>
    client.delete(`/api/v1/videos/${uuid}/subtitles/${id}`),

  // Cast
  listCast: (uuid: string) =>
    client.get<ApiResponse<VideoCast[]>>(`/api/v1/videos/${uuid}/cast`),

  deleteCast: (uuid: string, id: number) =>
    client.delete(`/api/v1/videos/${uuid}/cast/${id}`),

  // Global tasks
  listAllTasks: (params?: { page?: number; per_page?: number; status?: string; type?: string }) =>
    client.get<ApiResponse<TranscodeTask[]>>('/api/v1/tasks', { params }),

  cancelTask: (taskUuid: string) =>
    client.post(`/api/v1/tasks/${taskUuid}/cancel`),

  retryTask: (taskUuid: string) =>
    client.post(`/api/v1/tasks/${taskUuid}/retry`),
};
