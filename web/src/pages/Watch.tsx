import { useEffect, useRef, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import Hls from 'hls.js';
import { publicApi } from '../api/public';
import type { Video, Subtitle } from '../types';

function formatDuration(seconds?: number) {
  if (!seconds) return '';
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return `${m}:${String(s).padStart(2, '0')}`;
}

export default function Watch() {
  const { uuid } = useParams<{ uuid: string }>();
  const videoRef = useRef<HTMLVideoElement>(null);
  const hlsRef = useRef<Hls | null>(null);
  const [video, setVideo] = useState<Video | null>(null);
  const [error, setError] = useState(false);
  const [levels, setLevels] = useState<{ height: number; width: number; bitrate: number; index: number }[]>([]);
  const [currentLevel, setCurrentLevel] = useState(-1);

  useEffect(() => {
    if (!uuid) return;
    publicApi.getVideo(uuid).then((r) => setVideo(r.data.data)).catch(() => setError(true));
  }, [uuid]);

  useEffect(() => {
    if (!video || !video.master_playlist_key || !videoRef.current) return;

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
          height: l.height, width: l.width, bitrate: l.bitrate, index: i,
        })));
        videoRef.current?.play().catch(() => {});
      });

      hls.on(Hls.Events.LEVEL_SWITCHED, (_event, data) => {
        setCurrentLevel(data.level);
      });

      hls.on(Hls.Events.ERROR, (_event, data) => {
        if (data.fatal) {
          if (data.type === Hls.ErrorTypes.NETWORK_ERROR) hls.startLoad();
          else if (data.type === Hls.ErrorTypes.MEDIA_ERROR) hls.recoverMediaError();
          else hls.destroy();
        }
      });

      return () => { hls.destroy(); hlsRef.current = null; };
    } else if (videoRef.current.canPlayType('application/vnd.apple.mpegurl')) {
      videoRef.current.src = src;
      videoRef.current.play().catch(() => {});
    }
  }, [video]);

  // Add subtitle tracks
  useEffect(() => {
    if (!video || !videoRef.current) return;
    const el = videoRef.current;
    while (el.firstChild) el.removeChild(el.firstChild);
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

  const handleQualityChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const level = Number(e.target.value);
    if (hlsRef.current) hlsRef.current.currentLevel = level;
  };

  const title = video?.translations?.[0]?.title || video?.slug || video?.uuid || '';

  if (error) {
    return (
      <div style={{ minHeight: '100vh', background: '#0a0a0a', color: '#e5e5e5', display: 'flex', alignItems: 'center', justifyContent: 'center', flexDirection: 'column' }}>
        <div style={{ fontSize: 48, marginBottom: 16 }}>404</div>
        <p style={{ color: '#888' }}>Video not found</p>
        <Link to="/" style={{ marginTop: 16, color: '#4a9eff', textDecoration: 'none' }}>Back to home</Link>
      </div>
    );
  }

  if (!video) {
    return (
      <div style={{ minHeight: '100vh', background: '#0a0a0a', color: '#666', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        Loading...
      </div>
    );
  }

  return (
    <div style={{ minHeight: '100vh', background: '#0a0a0a', color: '#e5e5e5' }}>
      {/* Header */}
      <header style={{ borderBottom: '1px solid #222', padding: '12px 0' }}>
        <div style={{ maxWidth: 1280, margin: '0 auto', padding: '0 24px', display: 'flex', alignItems: 'center', gap: 16 }}>
          <Link to="/" style={{ color: '#888', textDecoration: 'none', fontSize: 14 }}>← Back</Link>
          <span style={{ color: '#fff', fontSize: 16, fontWeight: 600 }}>{title}</span>
        </div>
      </header>

      {/* Player */}
      <div style={{ maxWidth: 1280, margin: '0 auto' }}>
        <div style={{ background: '#000', position: 'relative' }}>
          <video
            ref={videoRef}
            controls
            style={{ width: '100%', maxHeight: 'calc(100vh - 200px)', display: 'block' }}
            crossOrigin="anonymous"
          />
        </div>

        {/* Controls bar */}
        <div style={{ padding: '16px 24px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid #222' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            {levels.length > 0 && (
              <select
                value={currentLevel}
                onChange={handleQualityChange}
                style={{
                  background: '#1a1a1a', border: '1px solid #333', borderRadius: 6,
                  padding: '6px 10px', color: '#e5e5e5', fontSize: 13,
                }}
              >
                <option value={-1}>Auto</option>
                {levels.map((l) => (
                  <option key={l.index} value={l.index}>{l.height}p ({(l.bitrate / 1000).toFixed(0)}k)</option>
                ))}
              </select>
            )}
          </div>
        </div>

        {/* Video info */}
        <div style={{ padding: '24px 24px 48px' }}>
          <h1 style={{ fontSize: 22, fontWeight: 600, margin: '0 0 12px' }}>{title}</h1>
          {video.translations?.[0]?.description && (
            <p style={{ color: '#aaa', fontSize: 14, lineHeight: 1.6, margin: '0 0 16px', maxWidth: 800 }}>
              {video.translations[0].description}
            </p>
          )}
          <div style={{ display: 'flex', gap: 16, fontSize: 13, color: '#666', flexWrap: 'wrap' }}>
            {video.duration_seconds && <span>Duration: {formatDuration(video.duration_seconds)}</span>}
            {video.width && video.height && <span>Resolution: {video.width}x{video.height}</span>}
            {video.codec && <span>Codec: {video.codec}</span>}
            {video.view_count > 0 && <span>{video.view_count} views</span>}
          </div>
          {(video.categories?.length || video.tags?.length) && (
            <div style={{ marginTop: 16, display: 'flex', gap: 8, flexWrap: 'wrap' }}>
              {video.categories?.map(c => (
                <span key={c.id} style={{ background: '#1a1a1a', border: '1px solid #333', borderRadius: 16, padding: '4px 12px', fontSize: 12 }}>
                  {c.translations?.[0]?.name || c.slug}
                </span>
              ))}
              {video.tags?.map(t => (
                <span key={t.id} style={{ background: '#1a1a1a', border: '1px solid #333', borderRadius: 16, padding: '4px 12px', fontSize: 12, color: '#888' }}>
                  #{t.translations?.[0]?.name || t.slug}
                </span>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
