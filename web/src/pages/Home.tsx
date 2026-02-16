import { useEffect, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { publicApi } from '../api/public';
import type { Video, PaginationMeta } from '../types';

const statusBadge = (status: string) => {
  const colors: Record<string, string> = { ready: '#52c41a', processing: '#1890ff', error: '#ff4d4f' };
  return colors[status] || '#999';
};

function formatDuration(seconds?: number) {
  if (!seconds) return '';
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return `${m}:${String(s).padStart(2, '0')}`;
}

export default function Home() {
  const [videos, setVideos] = useState<Video[]>([]);
  const [meta, setMeta] = useState<PaginationMeta>({ page: 1, per_page: 24, total: 0 });
  const [loading, setLoading] = useState(true);
  const [searchParams, setSearchParams] = useSearchParams();
  const [search, setSearch] = useState(searchParams.get('q') || '');

  const page = Number(searchParams.get('page')) || 1;

  useEffect(() => {
    setLoading(true);
    publicApi.listVideos({
      page,
      per_page: 24,
      q: searchParams.get('q') || undefined,
    }).then((res) => {
      setVideos(res.data.data || []);
      if (res.data.meta) setMeta(res.data.meta);
    }).catch(() => {}).finally(() => setLoading(false));
  }, [page, searchParams.get('q')]);

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    const params: Record<string, string> = {};
    if (search) params.q = search;
    setSearchParams(params);
  };

  const totalPages = Math.ceil(meta.total / meta.per_page);

  const getThumbUrl = (video: Video) => {
    const thumb = video.thumbnails?.find(t => t.is_default) || video.thumbnails?.[0];
    if (thumb) {
      const filename = thumb.s3_key.split('/').pop();
      return `/play/${video.uuid}/thumbnails/${filename}`;
    }
    return null;
  };

  return (
    <div style={{ minHeight: '100vh', background: '#0a0a0a', color: '#e5e5e5' }}>
      {/* Header */}
      <header style={{ borderBottom: '1px solid #222', padding: '16px 0' }}>
        <div style={{ maxWidth: 1280, margin: '0 auto', padding: '0 24px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Link to="/" style={{ color: '#fff', textDecoration: 'none', fontSize: 22, fontWeight: 700, letterSpacing: -0.5 }}>
            HLS Streamer
          </Link>
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <form onSubmit={handleSearch} style={{ display: 'flex' }}>
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search videos..."
                style={{
                  background: '#1a1a1a', border: '1px solid #333', borderRadius: '6px 0 0 6px',
                  padding: '8px 14px', color: '#e5e5e5', fontSize: 14, width: 240, outline: 'none',
                }}
              />
              <button type="submit" style={{
                background: '#333', border: '1px solid #333', borderLeft: 'none', borderRadius: '0 6px 6px 0',
                padding: '8px 16px', color: '#e5e5e5', cursor: 'pointer', fontSize: 14,
              }}>
                Search
              </button>
            </form>
            <Link to="/admin" style={{ color: '#888', textDecoration: 'none', fontSize: 13 }}>Admin</Link>
          </div>
        </div>
      </header>

      {/* Content */}
      <main style={{ maxWidth: 1280, margin: '0 auto', padding: '32px 24px' }}>
        {loading ? (
          <div style={{ textAlign: 'center', padding: 80, color: '#666' }}>Loading...</div>
        ) : videos.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 80, color: '#666' }}>
            <div style={{ fontSize: 48, marginBottom: 16 }}>🎬</div>
            <p style={{ fontSize: 18 }}>No videos found</p>
            {searchParams.get('q') && (
              <button onClick={() => setSearchParams({})} style={{
                marginTop: 12, background: '#222', border: '1px solid #444', borderRadius: 6,
                padding: '8px 20px', color: '#e5e5e5', cursor: 'pointer',
              }}>
                Clear search
              </button>
            )}
          </div>
        ) : (
          <>
            <div style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
              gap: 24,
            }}>
              {videos.map((video) => {
                const title = video.translations?.[0]?.title || video.slug || video.uuid;
                const thumbUrl = getThumbUrl(video);
                return (
                  <Link
                    key={video.uuid}
                    to={`/watch/${video.uuid}`}
                    style={{ textDecoration: 'none', color: 'inherit' }}
                  >
                    <div style={{
                      borderRadius: 10, overflow: 'hidden', background: '#141414',
                      transition: 'transform 0.2s, box-shadow 0.2s',
                    }}
                      onMouseEnter={(e) => { e.currentTarget.style.transform = 'translateY(-4px)'; e.currentTarget.style.boxShadow = '0 8px 30px rgba(0,0,0,0.4)'; }}
                      onMouseLeave={(e) => { e.currentTarget.style.transform = ''; e.currentTarget.style.boxShadow = ''; }}
                    >
                      {/* Thumbnail */}
                      <div style={{ position: 'relative', paddingTop: '56.25%', background: '#1a1a1a' }}>
                        {thumbUrl ? (
                          <img src={thumbUrl} alt={title} style={{
                            position: 'absolute', top: 0, left: 0, width: '100%', height: '100%', objectFit: 'cover',
                          }} />
                        ) : (
                          <div style={{
                            position: 'absolute', top: 0, left: 0, width: '100%', height: '100%',
                            display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#444', fontSize: 40,
                          }}>
                            ▶
                          </div>
                        )}
                        {video.duration_seconds && (
                          <span style={{
                            position: 'absolute', bottom: 8, right: 8, background: 'rgba(0,0,0,0.8)',
                            padding: '2px 6px', borderRadius: 4, fontSize: 12, fontWeight: 500,
                          }}>
                            {formatDuration(video.duration_seconds)}
                          </span>
                        )}
                      </div>
                      {/* Info */}
                      <div style={{ padding: '12px 14px' }}>
                        <div style={{
                          fontSize: 15, fontWeight: 500, lineHeight: 1.4,
                          overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
                        }}>
                          {title}
                        </div>
                        <div style={{ marginTop: 6, fontSize: 12, color: '#888', display: 'flex', gap: 8, alignItems: 'center' }}>
                          {video.width && video.height && <span>{video.width}x{video.height}</span>}
                          {video.codec && <span>{video.codec}</span>}
                          {video.view_count > 0 && <span>{video.view_count} views</span>}
                          <span style={{ width: 6, height: 6, borderRadius: '50%', background: statusBadge(video.status), display: 'inline-block' }} />
                        </div>
                      </div>
                    </div>
                  </Link>
                );
              })}
            </div>

            {/* Pagination */}
            {totalPages > 1 && (
              <div style={{ display: 'flex', justifyContent: 'center', gap: 8, marginTop: 40 }}>
                {page > 1 && (
                  <button onClick={() => setSearchParams(prev => { const p = Object.fromEntries(prev); p.page = String(page - 1); return p; })} style={paginationBtn}>
                    Previous
                  </button>
                )}
                <span style={{ padding: '8px 16px', color: '#888', fontSize: 14 }}>
                  Page {page} of {totalPages}
                </span>
                {page < totalPages && (
                  <button onClick={() => setSearchParams(prev => { const p = Object.fromEntries(prev); p.page = String(page + 1); return p; })} style={paginationBtn}>
                    Next
                  </button>
                )}
              </div>
            )}
          </>
        )}
      </main>
    </div>
  );
}

const paginationBtn: React.CSSProperties = {
  background: '#1a1a1a', border: '1px solid #333', borderRadius: 6,
  padding: '8px 20px', color: '#e5e5e5', cursor: 'pointer', fontSize: 14,
};
