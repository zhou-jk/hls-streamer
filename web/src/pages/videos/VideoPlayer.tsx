import { useEffect, useRef, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Button, Select, Space, message } from 'antd';
import { ArrowLeftOutlined, DownloadOutlined, ExpandOutlined, CompressOutlined } from '@ant-design/icons';
import Hls from 'hls.js';
import { videosApi } from '../../api/videos';
import type { Video, Subtitle } from '../../types';

export default function VideoPlayer() {
  const { uuid } = useParams<{ uuid: string }>();
  const navigate = useNavigate();
  const videoRef = useRef<HTMLVideoElement>(null);
  const hlsRef = useRef<Hls | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const [video, setVideo] = useState<Video | null>(null);
  const [levels, setLevels] = useState<{ height: number; width: number; bitrate: number; index: number }[]>([]);
  const [currentLevel, setCurrentLevel] = useState(-1);
  const [isFullscreen, setIsFullscreen] = useState(false);

  useEffect(() => {
    if (!uuid) return;
    videosApi.get(uuid).then((r) => setVideo(r.data.data)).catch(() => message.error('视频不存在'));
  }, [uuid]);

  useEffect(() => {
    if (!video || video.status !== 'ready' || !videoRef.current) return;

    const src = `/play/${video.uuid}/master.m3u8`;

    if (Hls.isSupported()) {
      const hls = new Hls({
        startLevel: -1,
        capLevelToPlayerSize: true,
      });
      hlsRef.current = hls;
      hls.loadSource(src);
      hls.attachMedia(videoRef.current);

      hls.on(Hls.Events.MANIFEST_PARSED, (_event, data) => {
        setLevels(data.levels.map((l, i) => ({
          height: l.height,
          width: l.width,
          bitrate: l.bitrate,
          index: i,
        })));
        videoRef.current?.play().catch(() => {});
      });

      hls.on(Hls.Events.LEVEL_SWITCHED, (_event, data) => {
        setCurrentLevel(data.level);
      });

      hls.on(Hls.Events.ERROR, (_event, data) => {
        if (data.fatal) {
          switch (data.type) {
            case Hls.ErrorTypes.NETWORK_ERROR:
              hls.startLoad();
              break;
            case Hls.ErrorTypes.MEDIA_ERROR:
              hls.recoverMediaError();
              break;
            default:
              hls.destroy();
              break;
          }
        }
      });

      return () => {
        hls.destroy();
        hlsRef.current = null;
      };
    } else if (videoRef.current.canPlayType('application/vnd.apple.mpegurl')) {
      // Safari native HLS
      videoRef.current.src = src;
      videoRef.current.play().catch(() => {});
    }
  }, [video]);

  // Add subtitle tracks
  useEffect(() => {
    if (!video || !videoRef.current) return;
    const el = videoRef.current;
    // Remove existing tracks
    while (el.firstChild) el.removeChild(el.firstChild);
    // Add subtitle tracks
    (video.subtitles || []).forEach((sub: Subtitle) => {
      const track = document.createElement('track');
      track.kind = 'subtitles';
      track.label = sub.label;
      track.srclang = sub.language_code;
      track.src = `/play/${video.uuid}/subtitles/${sub.language_code}.vtt`;
      if (sub.is_default) track.default = true;
      el.appendChild(track);
    });
  }, [video]);

  useEffect(() => {
    const handler = () => setIsFullscreen(!!document.fullscreenElement);
    document.addEventListener('fullscreenchange', handler);
    return () => document.removeEventListener('fullscreenchange', handler);
  }, []);

  const handleQualityChange = (level: number) => {
    if (hlsRef.current) {
      hlsRef.current.currentLevel = level;
    }
  };

  const toggleFullscreen = () => {
    if (!containerRef.current) return;
    if (document.fullscreenElement) {
      document.exitFullscreen();
    } else {
      containerRef.current.requestFullscreen();
    }
  };

  const title = video?.translations?.[0]?.title || video?.original_filename || video?.uuid || '';

  return (
    <div ref={containerRef} style={{ background: '#000', minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
      <div style={{ padding: '12px 16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center', background: 'rgba(0,0,0,0.8)' }}>
        <Space>
          <Button type="text" icon={<ArrowLeftOutlined />} onClick={() => navigate(-1)} style={{ color: '#fff' }}>返回</Button>
          <span style={{ color: '#fff', fontSize: 16 }}>{title}</span>
        </Space>
        <Space>
          {levels.length > 0 && (
            <Select
              value={currentLevel}
              onChange={handleQualityChange}
              style={{ width: 140 }}
              size="small"
              options={[
                { value: -1, label: '自动' },
                ...levels.map((l) => ({
                  value: l.index,
                  label: `${l.height}p (${(l.bitrate / 1000).toFixed(0)}k)`,
                })),
              ]}
            />
          )}
          {video && video.status !== 'draft' && (
            <Button type="text" icon={<DownloadOutlined />} href={`/play/${video.uuid}/download`} target="_blank" style={{ color: '#fff' }}>
              下载原片
            </Button>
          )}
          <Button type="text" icon={isFullscreen ? <CompressOutlined /> : <ExpandOutlined />} onClick={toggleFullscreen} style={{ color: '#fff' }} />
        </Space>
      </div>

      <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        {video?.status === 'ready' ? (
          <video
            ref={videoRef}
            controls
            style={{ width: '100%', maxHeight: 'calc(100vh - 56px)' }}
            crossOrigin="anonymous"
          />
        ) : (
          <div style={{ color: '#999', fontSize: 18 }}>
            {video ? `视频状态: ${video.status}，暂不可播放` : '加载中...'}
          </div>
        )}
      </div>
    </div>
  );
}
