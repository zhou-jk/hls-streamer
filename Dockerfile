# ---- Go build stage ----
FROM golang:1.24-alpine AS go-build
RUN apk add --no-cache git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/api   ./cmd/api
RUN CGO_ENABLED=0 go build -o /bin/worker ./cmd/worker

# ---- Frontend build stage ----
FROM node:20-alpine AS web-build
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ .
RUN npm run build

# ---- API runtime ----
FROM alpine:3.20 AS api
RUN apk add --no-cache ca-certificates tzdata
COPY --from=go-build /bin/api /usr/local/bin/api
COPY --from=web-build /web/dist /srv/web
ENTRYPOINT ["api"]
CMD ["-config", "/etc/hls-streamer/config.yaml"]

# ---- Worker runtime ----
FROM alpine:3.20 AS worker
RUN apk add --no-cache ca-certificates tzdata ffmpeg
# Install Shaka Packager
ADD https://github.com/shaka-project/shaka-packager/releases/latest/download/packager-linux-x64 /usr/local/bin/packager
RUN chmod +x /usr/local/bin/packager
COPY --from=go-build /bin/worker /usr/local/bin/worker
ENTRYPOINT ["worker"]
CMD ["-config", "/etc/hls-streamer/config.yaml"]
