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
- HLS 转码，多分辨率输出
- HLS 段伪装：`.ts` 重命名为 `.jpeg`，绕过 CDN 缓存限制
- DRM 加密 (Widevine + FairPlay，通过 Shaka Packager CENC)
- Worker 转码节点，支持水平扩展
- 缩略图自动生成
- WebVTT 软字幕管理（浏览器上传）
- 多语言元数据 (翻译表模式，支持任意语言)
- 演职人员管理 (导演/演员/编剧/制片)
- 分类与标签 (支持层级分类，均可多语言)
- RBAC 权限控制 (admin / editor / viewer)
- 播放代理端点 (master playlist / variant / segment / subtitle)
- 视频公开/私有控制
- 内置 React 管理前端 (视频管理、用户管理、分类标签、任务监控)

## Docker Compose 快速部署（推荐）

### 1. 启动所有服务

```bash
docker compose up -d --build
```

这会启动：
- MySQL 8.0 (端口 3306)
- Redis 7 (端口 6379)
- MinIO S3 (端口 9000 API / 9001 控制台)
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

jwt:
  secret: "your-jwt-secret"

worker:
  concurrency: 1

hls:
  segment_duration: 6
  segment_extension: ".jpeg"
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
├── Dockerfile                   # 多阶段构建
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
| POST | `/users` | 创建用户 |
| PUT | `/users/:id` | 更新用户 |
| DELETE | `/users/:id` | 停用用户 |

### 视频

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/videos` | 视频列表 (`?status=&category=&q=`) |
| GET | `/videos/:uuid` | 视频详情 |
| POST | `/videos` | 创建视频 (只需 language + title) |
| PUT | `/videos/:uuid` | 更新 (slug/rating/is_public 等) |
| DELETE | `/videos/:uuid` | 软删除 |
| POST | `/videos/:uuid/restore` | 恢复 |
| POST | `/videos/:uuid/upload/initiate` | S3 分片上传 |
| POST | `/videos/:uuid/upload/complete` | 上传完成 |
| POST | `/videos/:uuid/subtitles/upload` | 上传字幕 |
| POST | `/videos/:uuid/transcode` | 开始转码 |
| GET | `/videos/:uuid/tasks` | 任务列表 |
| GET | `/videos/:uuid/variants` | 变体列表 |
| GET | `/videos/:uuid/thumbnails` | 缩略图列表 |
| POST | `/videos/:uuid/thumbnails/generate` | 生成缩略图 |

### 全局任务

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/tasks` | 任务列表 (`?status=&type=`) |
| POST | `/tasks/:task_uuid/cancel` | 取消任务 |
| POST | `/tasks/:task_uuid/retry` | 重试任务 |

### 分类 & 标签

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/categories` | 分类列表 |
| POST | `/categories` | 创建分类 |
| PUT | `/categories/:id` | 更新分类 |
| DELETE | `/categories/:id` | 删除分类 |
| GET | `/tags` | 标签列表 |
| POST | `/tags` | 创建标签 |
| DELETE | `/tags/:id` | 删除标签 |

### 播放

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/play/:uuid/master.m3u8` | Master playlist |
| GET | `/play/:uuid/:variant/playlist.m3u8` | Variant playlist |
| GET | `/play/:uuid/:variant/:segment.jpeg` | 段代理 |
| GET | `/play/:uuid/subtitles/:lang.vtt` | 字幕文件 |

### DRM

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/drm/widevine/license` | Widevine license |
| POST | `/drm/fairplay/certificate` | FairPlay 证书 |
| POST | `/drm/fairplay/license` | FairPlay license |

## 转码流水线

```
上传完成 → Probe → 创建转码任务 → Worker 拉取 → FFmpeg 转码
→ .ts 重命名 .jpeg → (可选) DRM 加密 → 上传 S3
→ 回调 API → 所有变体完成 → 生成 master.m3u8 → 状态改为 ready
```

## License

MIT
