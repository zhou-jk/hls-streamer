package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Redis      RedisConfig      `yaml:"redis"`
	S3         S3Config         `yaml:"s3"`
	JWT        JWTConfig        `yaml:"jwt"`
	FFmpeg     FFmpegConfig     `yaml:"ffmpeg"`
	Shaka      ShakaConfig      `yaml:"shaka"`
	DRM        DRMConfig        `yaml:"drm"`
	Worker     WorkerConfig     `yaml:"worker"`
	Thumbnails ThumbnailConfig  `yaml:"thumbnails"`
	HLS        HLSConfig        `yaml:"hls"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"` // debug / release
}

func (c ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type DatabaseConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	Name         string `yaml:"name"`
	User         string `yaml:"user"`
	Password     string `yaml:"password"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
}

func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.User, c.Password, c.Host, c.Port, c.Name)
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type S3Config struct {
	Endpoint       string `yaml:"endpoint"`
	PublicEndpoint string `yaml:"public_endpoint"` // public-facing endpoint for presigned URLs (browser access)
	Region         string `yaml:"region"`
	Bucket         string `yaml:"bucket"`
	AccessKey      string `yaml:"access_key"`
	SecretKey      string `yaml:"secret_key"`
	UsePathStyle   bool   `yaml:"use_path_style"`
}

type JWTConfig struct {
	Secret     string        `yaml:"secret"`
	AccessTTL  time.Duration `yaml:"access_ttl"`
	RefreshTTL time.Duration `yaml:"refresh_ttl"`
}

type FFmpegConfig struct {
	Path       string `yaml:"path"`
	FFprobePath string `yaml:"ffprobe_path"`
	Threads    int    `yaml:"threads"`
}

type ShakaConfig struct {
	Path string `yaml:"path"`
}

type DRMConfig struct {
	FairPlayCertPath string `yaml:"fairplay_cert_path"`
	FairPlayKeyPath  string `yaml:"fairplay_key_path"`
}

type WorkerConfig struct {
	ID                string        `yaml:"id"`
	Concurrency       int           `yaml:"concurrency"`
	TempDir           string        `yaml:"temp_dir"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
}

type ThumbnailConfig struct {
	DefaultCount  int `yaml:"default_count"`
	DefaultWidth  int `yaml:"default_width"`
	DefaultHeight int `yaml:"default_height"`
}

type HLSConfig struct {
	SegmentDuration  int    `yaml:"segment_duration"`
	SegmentExtension string `yaml:"segment_extension"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	cfg := &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
			Mode: "debug",
		},
		Database: DatabaseConfig{
			Host:         "localhost",
			Port:         3306,
			MaxOpenConns: 50,
			MaxIdleConns: 10,
		},
		Redis: RedisConfig{
			Addr: "localhost:6379",
			DB:   0,
		},
		S3: S3Config{
			Region: "us-east-1",
		},
		JWT: JWTConfig{
			AccessTTL:  15 * time.Minute,
			RefreshTTL: 7 * 24 * time.Hour,
		},
		FFmpeg: FFmpegConfig{
			Path:       "ffmpeg",
			FFprobePath: "ffprobe",
		},
		Shaka: ShakaConfig{
			Path: "packager",
		},
		Worker: WorkerConfig{
			Concurrency:       2,
			TempDir:           "/tmp/hls-worker",
			HeartbeatInterval: 10 * time.Second,
		},
		Thumbnails: ThumbnailConfig{
			DefaultCount:  5,
			DefaultWidth:  320,
			DefaultHeight: 180,
		},
		HLS: HLSConfig{
			SegmentDuration:  6,
			SegmentExtension: ".jpeg",
		},
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}
