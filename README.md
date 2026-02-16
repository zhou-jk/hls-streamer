# HLS Streamer

影视站视频管理后端，提供视频上传、HLS 转码、多分辨率输出、DRM 加密、S3 存储等完整视频处理流水线。内置 React 管理前端。设计为 WordPress 前端通过 REST API 消费。

## 架构概览

系统由两个独立进程组成：

- **API Server** (`hls-api`) — 提供 REST API，处理用户认证、视频元数据管理、上传、播放代理，内嵌管理前端
- **Worker** (`hls-worker`) — 转码节点，从 Redis Streams 拉取任务，执行 FFmpeg 转码、缩略图提取、DRM 加密等

```
┌──────────┐     REST API     ┌──────────┐     Redis Streams     ┌──────────┐
│ WordPress│ ───────────────→ │ API      │ ──────────────────→   │ Worker   │
│ / Client │ ←─────────────── │ Server   │ ←── HTTP Callback ── │          │
└──────────┘                  └────┬─────┘                       └────┬─────┘
                                   │                                  │
┌──────────┐               ┌──────┴──────┐                     ┌─────┴──────┐
│ 管理前端 │──────────────→│  MySQL      │                     │  FFmpeg    │
│ (React)  │               │  Redis      │                     │  Shaka     │
└──────────┘               └─────────────┘                     └─────┬──────┘
                                                               ┌─────┴──────┐
                                                               │  S3/MinIO  │
                                                               └────────────┘
```

## 技术栈

| 组件 | 选型 |
|------|------|
| 语言 | Go 1.24+ |
| Web 框架 | Gin |
| ORM | GORM |
| 数据库 | MySQL 8.0 |
| 消息队列 | Redis Streams |
| 对象存储 | S3 兼容 (AWS S3 / MinIO / Cloudflare R2) |
| 转码 | FFmpeg |
| DRM 打包 | Shaka Packager |
| 认证 | JWT (access + refresh token) |
| 管理前端 | React 18 + TypeScript + Ant Design 5 + Vite |
| 部署 | Docker Compose / Bare metal + systemd |

## 核心特性

- 视频上传 (S3 presigned multipart upload，浏览器直传)
- 原始文件 S3 路径随机化，防止猜测下载
- HLS 转码，多分辨率输出 (360p ~ 2160p)
- HLS 段伪装：`.ts` 重命名为 `.jpeg`，绕过 CDN 缓存限制
- DRM 加密 (Widevine ClearKey，通过 Shaka Packager CENC)
- Worker 转码节点，支持水平扩展
- 缩略图自动生成
- WebVTT 软字幕管理（浏览器上传）
- 多语言元数据 (翻译表模式，支持任意语言)
- 演职人员管理 (导演/演员/编剧/制片)
- 分类与标签 (支持层级分类，均可多语言)
- RBAC 权限控制 (admin / editor / viewer)
- S3 公开读支持 (public-read ACL，直接 URL 访问，无需 presign)
- 播放代理端点 (master playlist / variant / segment / subtitle / thumbnail / download)
- 视频公开/私有控制
- 变体管理 (单个/批量删除，自动重建 master playlist)
- 视频硬删除 (清理 S3 + DB 所有关联数据)
- 任务取消/重试
- 内置 React 管理前端 (视频管理、用户管理、分类标签、任务监控)

## Docker Compose 快速部署（推荐）

### 1. 启动所有服务

```bash
docker compose up -d --build
```

这会启动：
- MySQL 8.0 (端口 3306)
- Redis 7 (端口 6379)
- MinIO S3 (端口 9000 API / 9001 控制台，自动创建 bucket 并设置 public-read)
- API Server + 管理前端 (端口 8080)
- Worker 转码节点

### 2. 访问

- 管理前端: `http://localhost:8080`
- MinIO 控制台: `http://localhost:9001` (minioadmin / minioadmin)

### 3. 默认账号

- 用户名: `admin`
- 密码: `admin123`

首次启动自动创建，请登录后修改密码。

### 4. 重新编译部署

修改代码后重新构建：

```bash
docker compose up -d --build
```

完全清理重建（删除所有容器、镜像、数据卷）：

```bash
bash scripts/docker-clean.sh
docker compose up -d --build
```

## 手动部署

### 前置依赖

- Go 1.24+
- Node.js 20+ (构建前端)
- MySQL 8.0+
- Redis 6.0+
- FFmpeg / FFprobe
- Shaka Packager (DRM 加密需要)
- S3 兼容存储

### 1. 构建

```bash
# 构建后端
make build

# 构建前端
cd web && npm ci && npm run build
```

### 2. 配置

```bash
cp configs/config.yaml configs/config.production.yaml
```

关键配置项：

```yaml
database:
  host: localhost
  port: 3306
  name: hls_streamer
  user: root
  password: "your-password"

redis:
  addr: localhost:6379

s3:
  endpoint: "https://s3.amazonaws.com"
  public_endpoint: "https://your-cdn-or-public-s3.example.com"  # 浏览器访问的 S3 地址
  region: us-east-1
  bucket: hls-videos
  access_key: "your-access-key"
  secret_key: "your-secret-key"
  use_path_style: false       # MinIO 设为 true，AWS S3 设为 false
  public_read: true           # 启用后上传自动设置 public-read ACL，播放使用直接 URL

jwt:
  secret: "your-jwt-secret"

worker:
  concurrency: 1

hls:
  segment_duration: 6
  segment_extension: ".jpeg"  # 段文件伪装扩展名
```

### 3. 运行

```bash
# 启动 API Server
./bin/hls-api -config configs/config.production.yaml

# 启动 Worker
./bin/hls-worker -config configs/config.production.yaml -api http://localhost:8080
```

## 项目结构

```
hls-streamer/
├── cmd/
│   ├── api/main.go              # API Server 入口
│   └── worker/main.go           # Worker 节点入口
├── internal/
│   ├── config/                  # 配置加载 (YAML)
│   ├── model/                   # GORM 数据模型
│   ├── repository/              # 数据库操作层
│   ├── service/                 # 业务逻辑层
│   ├── handler/                 # HTTP 处理器
│   ├── middleware/              # JWT 认证、日志、CORS
│   ├── router/                  # 路由注册 + SPA 静态文件服务
│   ├── queue/                   # Redis Streams 生产者/消费者
│   ├── storage/                 # S3 客户端封装
│   └── worker/                  # Worker 处理逻辑
├── pkg/
│   ├── ffmpeg/                  # FFmpeg 命令构建器
│   ├── hls/                     # M3U8 解析/改写
│   ├── shaka/                   # Shaka Packager 命令构建器
│   └── response/                # 统一 API 响应格式
├── web/                         # React 管理前端
│   ├── src/
│   │   ├── api/                 # API 客户端
│   │   ├── pages/               # 页面组件
│   │   ├── components/          # 通用组件
│   │   ├── store/               # 状态管理 (Zustand)
│   │   └── types/               # TypeScript 类型
│   └── vite.config.ts
├── configs/                     # 配置文件
├── scripts/                     # 部署辅助脚本
├── Dockerfile                   # 多阶段构建 (Go + Node → Alpine)
├── docker-compose.yml
├── go.mod
└── go.sum
```

## API 端点

所有端点前缀 `/api/v1`。

### 认证

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/auth/login` | 登录 |
| POST | `/auth/register` | 注册 |
| POST | `/auth/refresh` | 刷新 token |
| POST | `/auth/logout` | 注销 |
| GET | `/auth/me` | 当前用户信息 |
| PUT | `/auth/me/password` | 修改密码 |

### 用户管理 (admin)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/users` | 用户列表 |
| GET | `/users/:id` | 用户详情 |
| POST | `/users` | 创建用户 |
| PUT | `/users/:id` | 更新用户 |
| DELETE | `/users/:id` | 停用用户 |

### 视频

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/videos` | 视频列表 (`?status=&category=&q=&page=&per_page=`) |
| GET | `/videos/:uuid` | 视频详情 (含翻译/变体/缩略图/字幕/演员) |
| POST | `/videos` | 创建视频 |
| PUT | `/videos/:uuid` | 更新 (slug/rating/is_public 等) |
| DELETE | `/videos/:uuid` | 永久删除 (清理 S3 + DB 所有关联数据) |
| POST | `/videos/:uuid/restore` | 恢复软删除 |

### 上传

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/videos/:uuid/upload/initiate` | 发起 S3 分片上传 (返回 presigned URLs) |
| POST | `/videos/:uuid/upload/complete` | 完成上传 (自动触发 probe) |
| POST | `/videos/:uuid/subtitles/upload` | 上传字幕文件 (multipart form) |

### 转码 & 任务

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/videos/:uuid/transcode` | 开始转码 (选择分辨率/编码/DRM) |
| GET | `/videos/:uuid/tasks` | 视频任务列表 |
| GET | `/videos/:uuid/tasks/:task_uuid` | 任务详情 |
| GET | `/tasks` | 全局任务列表 (`?status=&type=&page=&per_page=`) |
| POST | `/tasks/:task_uuid/cancel` | 取消任务 |
| POST | `/tasks/:task_uuid/retry` | 重试失败任务 |

### 变体

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/videos/:uuid/variants` | 变体列表 |
| DELETE | `/videos/:uuid/variants` | 删除所有变体 |
| DELETE | `/videos/:uuid/variants/:resolution` | 删除单个变体 (如 `1080p`) |

### 翻译

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/videos/:uuid/translations` | 翻译列表 |
| PUT | `/videos/:uuid/translations/:lang` | 添加/更新翻译 |
| DELETE | `/videos/:uuid/translations/:lang` | 删除翻译 |

### 缩略图

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/videos/:uuid/thumbnails` | 缩略图列表 |
| POST | `/videos/:uuid/thumbnails/generate` | 生成缩略图 |
| PUT | `/videos/:uuid/thumbnails/:id/default` | 设为默认缩略图 |
| DELETE | `/videos/:uuid/thumbnails/:id` | 删除缩略图 |

### 字幕

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/videos/:uuid/subtitles` | 字幕列表 |
| DELETE | `/videos/:uuid/subtitles/:id` | 删除字幕 |

### 演职人员

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/videos/:uuid/cast` | 演职人员列表 |
| DELETE | `/videos/:uuid/cast/:id` | 移除演职人员 |

### 分类 & 标签

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/categories` | 分类列表 (树形) |
| POST | `/categories` | 创建分类 |
| PUT | `/categories/:id` | 更新分类 |
| DELETE | `/categories/:id` | 删除分类 |
| PUT | `/categories/:id/translations/:lang` | 分类翻译 |
| GET | `/tags` | 标签列表 |
| POST | `/tags` | 创建标签 |
| DELETE | `/tags/:id` | 删除标签 |

### DRM

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/videos/:uuid/drm/keys` | 生成 DRM 密钥 |
| GET | `/videos/:uuid/drm/keys` | 获取 DRM 密钥 |
| POST | `/drm/clearkey/license` | ClearKey license 端点 (播放器调用) |

### 播放 (公开端点)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/play/:uuid/master.m3u8` | Master playlist |
| GET | `/play/:uuid/:variant/playlist.m3u8` | Variant playlist |
| GET | `/play/:uuid/:variant/:segment` | HLS 段 (重定向到 S3) |
| GET | `/play/:uuid/subtitles/:lang.vtt` | 字幕文件 |
| GET | `/play/:uuid/thumbnails/:filename` | 缩略图 |
| GET | `/play/:uuid/download` | 下载原片 |

### Worker 内部 API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/workers/register` | Worker 注册 |
| POST | `/workers/heartbeat` | 心跳上报 |
| POST | `/workers/tasks/:task_uuid/progress` | 进度上报 |
| POST | `/workers/tasks/:task_uuid/complete` | 任务完成回调 |
| POST | `/workers/tasks/:task_uuid/fail` | 任务失败回调 |

## 转码流水线

```
上传完成 → Probe (探测元数据) → 创建转码任务 → Worker 拉取
→ FFmpeg 转码 → .ts 重命名 .jpeg → (可选) Shaka DRM 加密
→ 上传 S3 → 回调 API → 所有变体完成 → 生成 master.m3u8 → 状态改为 ready
```

## 视频生命周期

```
draft → uploaded (上传+探测完成) → processing (转码中) → ready (可播放)
                                                       → error (转码失败)
```

## License

MIT
