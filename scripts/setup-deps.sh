#!/bin/bash
# setup-deps.sh - Install FFmpeg and Shaka Packager on Ubuntu/Debian
set -e

echo "=== Installing FFmpeg ==="
if command -v ffmpeg &>/dev/null; then
    echo "FFmpeg already installed: $(ffmpeg -version | head -1)"
else
    apt-get update
    apt-get install -y ffmpeg
    echo "FFmpeg installed: $(ffmpeg -version | head -1)"
fi

echo ""
echo "=== Installing Shaka Packager ==="
SHAKA_VERSION="3.3.0"
SHAKA_BIN="/usr/local/bin/packager"

if [ -f "$SHAKA_BIN" ]; then
    echo "Shaka Packager already installed at $SHAKA_BIN"
else
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64)  SHAKA_ARCH="linux-x64" ;;
        aarch64) SHAKA_ARCH="linux-arm64" ;;
        *)       echo "Unsupported architecture: $ARCH"; exit 1 ;;
    esac

    URL="https://github.com/shaka-project/shaka-packager/releases/download/v${SHAKA_VERSION}/packager-${SHAKA_ARCH}"
    echo "Downloading Shaka Packager v${SHAKA_VERSION} for ${SHAKA_ARCH}..."
    curl -fSL "$URL" -o "$SHAKA_BIN"
    chmod +x "$SHAKA_BIN"
    echo "Shaka Packager installed at $SHAKA_BIN"
fi

echo ""
echo "=== Installing Redis ==="
if command -v redis-server &>/dev/null; then
    echo "Redis already installed: $(redis-server --version)"
else
    apt-get install -y redis-server
    systemctl enable redis-server
    systemctl start redis-server
    echo "Redis installed and started"
fi

echo ""
echo "=== Creating hls user ==="
if id "hls" &>/dev/null; then
    echo "User 'hls' already exists"
else
    useradd -r -m -s /bin/bash hls
    echo "User 'hls' created"
fi

echo ""
echo "=== Creating directories ==="
mkdir -p /opt/hls-streamer/bin
mkdir -p /opt/hls-streamer/configs
mkdir -p /tmp/hls-worker
chown -R hls:hls /opt/hls-streamer
chown -R hls:hls /tmp/hls-worker

echo ""
echo "=== Setup complete ==="
echo "Next steps:"
echo "  1. Copy binaries to /opt/hls-streamer/bin/"
echo "  2. Copy config to /opt/hls-streamer/configs/config.production.yaml"
echo "  3. Copy systemd units to /etc/systemd/system/"
echo "  4. Run: systemctl daemon-reload"
echo "  5. Run: systemctl enable --now hls-api hls-worker"
