#!/bin/bash
# 清理 HLS Streamer 的所有 Docker 容器、镜像和数据卷，以便重新编译安装。
# 用法: bash scripts/docker-clean.sh

set -e

cd "$(dirname "$0")/.."

echo "==> 停止并删除容器..."
docker compose down --remove-orphans 2>/dev/null || true

echo "==> 删除数据卷 (MySQL/Redis/MinIO/Worker 临时文件)..."
docker compose down -v 2>/dev/null || true

echo "==> 删除构建的镜像..."
docker compose down --rmi local 2>/dev/null || true

echo "==> 清理悬空镜像和构建缓存..."
docker image prune -f 2>/dev/null || true
docker builder prune -f 2>/dev/null || true

echo "==> 清理完成。运行以下命令重新构建："
echo "    docker compose up -d --build"
