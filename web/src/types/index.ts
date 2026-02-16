export interface User {
  id: number;
  username: string;
  email: string;
  role_id: number;
  role?: Role;
  is_active: boolean;
  last_login_at?: string;
  created_at: string;
  updated_at: string;
}

export interface Role {
  id: number;
  name: string;
  permissions: Record<string, unknown>;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
}

export interface Video {
  id: number;
  uuid: string;
  slug: string;
  status: string;
  original_filename: string;
  original_s3_key?: string;
  duration_seconds?: number;
  width?: number;
  height?: number;
  file_size_bytes?: number;
  codec: string;
  fps?: number;
  has_drm: boolean;
  is_public: boolean;
  master_playlist_key?: string;
  release_date?: string;
  rating: string;
  sort_order: number;
  view_count: number;
  created_by: number;
  creator?: User;
  created_at: string;
  updated_at: string;
  deleted_at?: string;
  translations?: VideoTranslation[];
  variants?: VideoVariant[];
  thumbnails?: Thumbnail[];
  subtitles?: Subtitle[];
  categories?: Category[];
  tags?: Tag[];
  cast?: VideoCast[];
}

export interface VideoTranslation {
  id: number;
  video_id: number;
  language_code: string;
  title: string;
  description: string;
  synopsis: string;
  seo_title: string;
  seo_description: string;
  created_at: string;
  updated_at: string;
}

export interface VideoVariant {
  id: number;
  video_id: number;
  resolution_name: string;
  width: number;
  height: number;
  bitrate_kbps: number;
  codec: string;
  playlist_s3_key: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface Thumbnail {
  id: number;
  video_id: number;
  s3_key: string;
  width: number;
  height: number;
  timestamp_s: number;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface Subtitle {
  id: number;
  video_id: number;
  language_code: string;
  label: string;
  s3_key: string;
  is_default: boolean;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export interface Category {
  id: number;
  parent_id?: number;
  slug: string;
  sort_order: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  translations?: CategoryTranslation[];
  children?: Category[];
}

export interface CategoryTranslation {
  id: number;
  category_id: number;
  language_code: string;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface Tag {
  id: number;
  slug: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  translations?: TagTranslation[];
}

export interface TagTranslation {
  id: number;
  tag_id: number;
  language_code: string;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface Person {
  id: number;
  slug: string;
  photo_s3_key: string;
  created_at: string;
  updated_at: string;
  translations?: PersonTranslation[];
}

export interface PersonTranslation {
  id: number;
  person_id: number;
  language_code: string;
  name: string;
  biography: string;
}

export interface VideoCast {
  id: number;
  video_id: number;
  person_id: number;
  role: string;
  character: string;
  sort_order: number;
  person?: Person;
}

export interface TranscodeTask {
  id: number;
  task_uuid: string;
  video_id: number;
  type: string;
  status: string;
  priority: number;
  worker_id?: string;
  params: Record<string, unknown>;
  result?: Record<string, unknown>;
  error_message?: string;
  progress: number;
  attempts: number;
  max_attempts: number;
  started_at?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface Worker {
  id: string;
  hostname: string;
  ip_address: string;
  capabilities: Record<string, unknown>;
  status: string;
  current_task_id?: number;
  last_heartbeat: string;
  registered_at: string;
}

export interface ApiResponse<T> {
  success: boolean;
  data: T;
  meta?: PaginationMeta;
  error?: { code: string; message: string };
}

export interface PaginationMeta {
  page: number;
  per_page: number;
  total: number;
}
