# HLS Streamer

影视站视频管理后端，提供视频上传、HLS 转码、多分辨率输出、DRM 加密、S3 存储等完整视频处理流水线。设计为 WordPress 前端通过 REST API 消费。

## 架构概览

系统由两个独立进程组成：

- **API Server** (`hls-api`) — 提供 REST API，处理用户认证、视频元数据管理、上传、播放代理等
- **Worker** (`hls-worker`) — 分布式转码节点，从 Redis Streams 拉取任务，执行 FFmpeg 转码、缩略图提取、DRM 加密等

```
┌──────────┐     REST API     ┌──────────┐     Redis Streams     ┌──────────┐
│ WordPress│ ───────────────→ │ API      │ ──────────────────→   │ Worker 1 │
│ / Client │ ←─────────────── │ Server   │ ←── HTTP Callback ── │ Worker 2 │
└──────────┘                  └────┬─────┘                       │ Worker N │
                                   │                             └────┬─────┘
                              ┌────┴─────┐                       ┌────┴─────┐
                              │  MySQL   │                       │  FFmpeg  │
                              │  Redis   │                       │  Shaka   │
                              └──────────┘                       └────┬─────┘
                                                                 ┌────┴─────┐
                                                                 │    S3    │
                                                                 └──────────┘
```

## 技术栈

| 组件 | 选型 |
|------|------|
| 语言 | Go 1.24+ |
| Web 框架 | Gin |
| ORM | GORM |
| 数据库 | MySQL |
| 消息队列 | Redis Streams |
| 对象存储 | S3 兼容 (AWS S3 / MinIO / Cloudflare R2) |
| 转码 | FFmpeg |
| DRM 打包 | Shaka Packager |
| 认证 | JWT (access + refresh token) |
| 部署 | Bare metal + systemd |

## 核心特性

- 视频上传 (S3 presigned multipart upload) 与 HLS 转码
- 多分辨率输出，每个视频可自定义分辨率列表
- HLS 段伪装：`.ts` 重命名为 `.jpeg`，绕过 CDN 缓存限制
- DRM 加密 (Widevine + FairPlay，通过 Shaka Packager CENC)
- 分布式 Worker 架构，支持水平扩展
- 缩略图自动生成
- WebVTT 软字幕管理
- 多语言元数据 (翻译表模式，支持任意语言)
- 演职人员管理 (导演/演员/编剧/制片)
- 分类与标签 (支持层级分类，均可多语言)
- RBAC 权限控制 (admin / editor / viewer)
- 播放代理端点 (master playlist / variant / segment / subtitle)

## 前置依赖

- Go 1.24+
- MySQL 8.0+
- Redis 6.0+
- FFmpeg / FFprobe
- Shaka Packager (DRM 加密需要)
- S3 兼容存储

Ubuntu/Debian 可使用脚本一键安装 FFmpeg、Shaka Packager、Redis：

```bash
sudo bash scripts/setup-deps.sh
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
│   ├── middleware/              # JWT 认证、日志、限流、CORS
│   ├── router/                  # 路由注册
│   ├── queue/                   # Redis Streams 生产者/消费者
│   ├── storage/                 # S3 客户端封装
│   └── worker/                  # Worker 处理逻辑
├── pkg/
│   ├── ffmpeg/                  # FFmpeg 命令构建器
│   ├── hls/                     # M3U8 解析/改写
│   ├── shaka/                   # Shaka Packager 命令构建器
│   └── response/                # 统一 API 响应格式
├── migrations/                  # SQL 迁移文件 (001-007)
├── configs/                     # 配置文件模板
├── deploy/                      # systemd unit 文件
├── scripts/                     # 部署辅助脚本
├── Makefile
├── go.mod
└── go.sum
```

## 快速开始

### 1. 构建

```bash
# 构建 API Server 和 Worker
make build

# 单独构建
make build-api
make build-worker

# 交叉编译 Linux (用于部署)
make build-linux
```

### 2. 数据库初始化

创建 MySQL 数据库并执行迁移：

```bash
mysql -u root -e "CREATE DATABASE hls_streamer CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# 运行迁移脚本
DB_HOST=localhost DB_USER=root DB_NAME=hls_streamer bash scripts/migrate.sh
```

> API Server 启动时也会通过 GORM AutoMigrate 自动建表，但建议使用迁移脚本以获得完整的索引和约束。

### 3. 配置

复制并编辑配置文件：

```bash
cp configs/config.yaml configs/config.production.yaml
```

关键配置项：

```yaml
database:
  host: localhost
  port: 3306
  name: hls_streamer
  user: hls
  password: "your-password"

redis:
  addr: localhost:6379

s3:
  endpoint: "https://s3.amazonaws.com"  # 或 MinIO 地址
  region: us-east-1
  bucket: hls-videos
  access_key: "your-access-key"
  secret_key: "your-secret-key"

jwt:
  secret: "your-jwt-secret"

hls:
  segment_duration: 6
  segment_extension: ".jpeg"  # 段伪装后缀
```

### 4. 运行

```bash
# 启动 API Server
make run-api
# 或
./bin/hls-api -config configs/config.production.yaml

# 启动 Worker (可启动多个实例)
make run-worker
# 或
./bin/hls-worker -config configs/config.production.yaml -api http://localhost:8080
```

## API 文档

所有端点前缀 `/api/v1`，统一响应格式：

```json
{
  "success": true,
  "data": {},
  "meta": {"page": 1, "per_page": 20, "total": 150},
  "error": null
}
```

### 认证

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/auth/login` | 登录，返回 access + refresh token |
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
| GET | `/roles` | 角色列表 |

### 视频

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/videos` | 视频列表 (`?status=&category=&lang=&q=`) |
| GET | `/videos/:uuid` | 视频详情 |
| POST | `/videos` | 创建视频元数据 |
| PUT | `/videos/:uuid` | 更新元数据 |
| DELETE | `/videos/:uuid` | 软删除 |
| POST | `/videos/:uuid/restore` | 恢复 |

### 视频 - 多语言翻译

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/videos/:uuid/translations` | 所有翻译 |
| PUT | `/videos/:uuid/translations/:lang` | 新增/更新翻译 |
| DELETE | `/videos/:uuid/translations/:lang` | 删除翻译 |

### 视频 - 上传

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/videos/:uuid/upload/initiate` | 获取 S3 presigned multipart upload URL |
| POST | `/videos/:uuid/upload/complete` | 上传完成，触发 probe |

### 视频 - 转码

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/videos/:uuid/transcode` | 开始转码 `{"resolutions":["720p","1080p"],"codec":"h264","drm":true}` |
| GET | `/videos/:uuid/tasks` | 任务列表 |
| GET | `/videos/:uuid/tasks/:task_uuid` | 任务详情 + 进度 |

### 视频 - 变体 / 缩略图 / 字幕 / 演职人员

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/videos/:uuid/variants` | 可用分辨率变体 |
| GET | `/videos/:uuid/thumbnails` | 缩略图列表 |
| POST | `/videos/:uuid/thumbnails/generate` | 生成缩略图 |
| PUT | `/videos/:uuid/thumbnails/:id/default` | 设为默认缩略图 |
| DELETE | `/videos/:uuid/thumbnails/:id` | 删除缩略图 |
| GET | `/videos/:uuid/subtitles` | 字幕列表 |
| DELETE | `/videos/:uuid/subtitles/:id` | 删除字幕 |
| GET | `/videos/:uuid/cast` | 演职人员 |
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

### 播放

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/play/:uuid/master.m3u8` | Master playlist |
| GET | `/play/:uuid/:variant/playlist.m3u8` | Variant playlist |
| GET | `/play/:uuid/:variant/:segment.jpeg` | 段代理 (S3 presigned redirect) |
| GET | `/play/:uuid/subtitles/:lang.vtt` | 字幕文件 |

### DRM License

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/drm/widevine/license` | Widevine license 请求 |
| POST | `/drm/fairplay/certificate` | FairPlay 证书 |
| POST | `/drm/fairplay/license` | FairPlay license 请求 |

### Worker 内部 API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/workers/register` | Worker 注册 |
| POST | `/workers/heartbeat` | 心跳 |
| POST | `/workers/tasks/:task_uuid/progress` | 上报进度 |
| POST | `/workers/tasks/:task_uuid/complete` | 任务完成 |
| POST | `/workers/tasks/:task_uuid/fail` | 任务失败 |

## S3 存储结构

```
videos/{uuid}/
├── original/source.mp4          # 原始上传文件
├── thumbnails/                   # 缩略图
│   ├── thumb_001.jpg
│   └── thumb_002.jpg
├── subtitles/                    # WebVTT 字幕
│   ├── en.vtt
│   └── zh-CN.vtt
├── variants/
│   ├── 720p/
│   │   ├── playlist.m3u8
│   │   ├── segment_000.jpeg      # 实际是 .ts 内容
│   │   └── segment_001.jpeg
│   └── 1080p/
│       ├── playlist.m3u8
│       └── ...
└── master.m3u8                   # 主播放列表
```

## 转码流水线

```
上传完成
  → Probe (ffprobe 获取元数据)
  → 创建转码任务 (每个分辨率一个)
  → Worker 从 Redis Streams 拉取任务
  → FFmpeg 转码为 HLS (.ts 段)
  → .ts 重命名为 .jpeg + 改写 m3u8
  → (可选) Shaka Packager DRM 加密
  → 上传到 S3
  → 回调 API Server 上报完成
  → 所有变体完成 → 生成 master.m3u8 → 视频状态改为 ready
```

## 多语言支持

采用"每个实体一张翻译表"模式。可翻译字段放在 `_translations` 表中，以 `(entity_id, language_code)` 为联合唯一键。

API 通过 `Accept-Language` 请求头决定返回语言，缺失时回退到默认语言。

支持多语言的实体：视频、分类、标签、演职人员。

## 生产部署

### systemd

```bash
# 安装依赖
sudo bash scripts/setup-deps.sh

# 复制文件
sudo cp bin/hls-api-linux /opt/hls-streamer/bin/hls-api
sudo cp bin/hls-worker-linux /opt/hls-streamer/bin/hls-worker
sudo cp configs/config.production.yaml /opt/hls-streamer/configs/

# 安装 systemd 服务
sudo cp deploy/hls-api.service /etc/systemd/system/
sudo cp deploy/hls-worker.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now hls-api hls-worker
```

### 扩展 Worker

Worker 节点可独立部署在多台机器上，只需确保能访问 Redis 和 API Server：

```bash
./hls-worker -config config.yaml -api http://api-server:8080
```

每个 Worker 自动生成唯一 ID，通过 Redis Streams Consumer Group 实现任务自动分配和故障转移。

## License

MIT
