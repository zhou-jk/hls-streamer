import client from './client';
import type { ApiResponse } from '../types';

export interface AppSetting {
  id: number;
  key: string;
  value: string;
  description: string;
}

export const settingsApi = {
  list: () => client.get<ApiResponse<AppSetting[]>>('/api/v1/settings'),
  update: (key: string, value: string) => client.put<ApiResponse<null>>('/api/v1/settings', { key, value }),
};
