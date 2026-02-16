import client from './client';
import type { ApiResponse, Worker } from '../types';

export const workersApi = {
  list: () =>
    client.get<ApiResponse<Worker[]>>('/api/v1/workers'),
};
