import { useEffect, useState, useRef, useCallback } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { Tabs, Descriptions, Tag, Button, Form, Input, Select, Table, Space, Card, message, Popconfirm, Progress, Upload, Switch, Image } from 'antd';
import { ArrowLeftOutlined, UploadOutlined, InboxOutlined, CopyOutlined, DownloadOutlined, PlayCircleOutlined, DeleteOutlined } from '@ant-design/icons';
import Hls from 'hls.js';
import { videosApi } from '../../api/videos';
import type { Video, VideoTranslation, VideoVariant, Thumbnail, Subtitle, TranscodeTask } from '../../types';
import axios from 'axios';

const { Dragger } = Upload;

const statusColors: Record<string, string> = {
  draft: 'default', uploaded: 'blue', processing: 'processing', ready: 'success', error: 'error', archived: 'warning',
  pending: 'default', queued: 'cyan', completed: 'success', failed: 'error', cancelled: 'warning',
};

const PRESETS = [
  { name: '360p', width: 640, height: 360, bitrate_kbps: 800 },
  { name: '480p', width: 854, height: 480, bitrate_kbps: 1400 },
  { name: '720p', width: 1280, height: 720, bitrate_kbps: 2800 },
  { name: '1080p', width: 1920, height: 1080, bitrate_kbps: 5000 },
  { name: '1440p', width: 2560, height: 1440, bitrate_kbps: 8000 },
  { name: '2160p', width: 3840, height: 2160, bitrate_kbps: 14000 },
];

const PART_SIZE = 10 * 1024 * 1024; // 10MB per part

export default function VideoDetail() {
  const { uuid } = useParams<{ uuid: string }>();
  const navigate = useNavigate();
  const [video, setVideo] = useState<Video | null>(null);
  const [tasks, setTasks] = useState<TranscodeTask[]>([]);
  const [transForm] = Form.useForm();
  const [transLang] = Form.useForm();
  const [editForm] = Form.useForm();
  const [subForm] = Form.useForm();
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [uploadDetail, setUploadDetail] = useState({ uploaded: 0, total: 0, speed: 0, part: 0, partCount: 0 });
  const [subUploading, setSubUploading] = useState(false);
  const subFileRef = useRef<File | null>(null);
  const previewRef = useRef<HTMLVideoElement>(null);
  const previewHlsRef = useRef<Hls | null>(null);

  const initPreviewPlayer = useCallback(() => {
    if (!video || !video.master_playlist_key || !previewRef.current) return;
    // Already initialized
    if (previewHlsRef.current) return;

    const src = `/play/${video.uuid}/master.m3u8`;
    if (Hls.isSupported()) {
      const hls = new Hls({ startLevel: -1, capLevelToPlayerSize: true });
      previewHlsRef.current = hls;
      hls.loadSource(src);
      hls.attachMedia(previewRef.current);
      hls.on(Hls.Events.ERROR, (_event, data) => {
        if (data.fatal) {
          if (data.type === Hls.ErrorTypes.NETWORK_ERROR) hls.startLoad();
          else if (data.type === Hls.ErrorTypes.MEDIA_ERROR) hls.recoverMediaError();
          else hls.destroy();
        }
      });
    } else if (previewRef.current.canPlayType('application/vnd.apple.mpegurl')) {
      previewRef.current.src = src;
    }

    // Add subtitle tracks
    const el = previewRef.current;
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

  // Cleanup HLS on unmount
  useEffect(() => {
    return () => {
      if (previewHlsRef.current) {
        previewHlsRef.current.destroy();
        previewHlsRef.current = null;
      }
    };
  }, []);

  const formatSize = (bytes: number) => {
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
    return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
  };

  const load = () => {
    if (!uuid) return;
    videosApi.get(uuid).then((r) => setVideo(r.data.data)).catch(() => message.error('视频不存在'));
    videosApi.listTasks(uuid).then((r) => setTasks(r.data.data || [])).catch(() => {});
  };

  useEffect(() => { load(); }, [uuid]);
  useEffect(() => {
    if (!tasks.some((t) => ['queued', 'processing', 'pending'].includes(t.status))) return;
    const timer = setInterval(() => {
      if (uuid) videosApi.listTasks(uuid).then((r) => setTasks(r.data.data || [])).catch(() => {});
    }, 5000);
    return () => clearInterval(timer);
  }, [tasks, uuid]);

  if (!video) return null;

  const handleUpdate = async (values: Record<string, string>) => {
    await videosApi.update(video.uuid, values);
    message.success('已更新');
    load();
  };

  // Video file upload via S3 presigned multipart
  const handleVideoUpload = async (file: File) => {
    if (!uuid) return;
    setUploading(true);
    setUploadProgress(0);
    setUploadDetail({ uploaded: 0, total: file.size, speed: 0, part: 0, partCount: 0 });
    try {
      const partCount = Math.ceil(file.size / PART_SIZE);
      const contentType = file.type || 'video/mp4';
      setUploadDetail(d => ({ ...d, partCount }));

      // 1. Initiate multipart upload
      const initRes = await videosApi.initiateUpload(uuid, {
        filename: file.name,
        content_type: contentType,
        part_count: partCount,
      });
      const { upload_id, part_urls, s3_key } = initRes.data.data;

      // 2. Upload each part to presigned URL
      const parts: { part_number: number; etag: string }[] = [];
      let totalUploaded = 0;
      for (let i = 1; i <= partCount; i++) {
        const start = (i - 1) * PART_SIZE;
        const end = Math.min(i * PART_SIZE, file.size);
        const blob = file.slice(start, end);
        const partSize = end - start;

        setUploadDetail(d => ({ ...d, part: i }));
        const partStart = Date.now();

        const res = await axios.put(part_urls[i], blob, {
          headers: { 'Content-Type': contentType },
          onUploadProgress: (e) => {
            if (e.total) {
              const partUploaded = e.loaded;
              const currentTotal = totalUploaded + partUploaded;
              const elapsed = (Date.now() - partStart) / 1000;
              const speed = elapsed > 0 ? partUploaded / elapsed : 0;
              setUploadDetail(d => ({ ...d, uploaded: currentTotal, speed }));
              setUploadProgress(Math.round((currentTotal / file.size) * 90));
            }
          },
        });
        const etag = res.headers['etag'] || res.headers['ETag'] || '';
        parts.push({ part_number: i, etag: etag.replace(/"/g, '') });
        totalUploaded += partSize;
      }

      // 3. Complete multipart upload
      setUploadProgress(95);
      await videosApi.completeUpload(uuid, { upload_id, s3_key, parts });
      setUploadProgress(100);
      message.success('视频上传完成，已开始探测');
      load();
    } catch (err) {
      message.error('上传失败');
      console.error(err);
    } finally {
      setUploading(false);
    }
  };

  const handleTranscode = async (values: { resolutions: string[]; codec: string; drm: boolean }) => {
    const resolutions = PRESETS.filter((p) => values.resolutions.includes(p.name));
    await videosApi.startTranscode(video.uuid, { resolutions, codec: values.codec || 'h264', drm: values.drm || false });
    message.success('转码任务已创建');
    load();
  };

  const handleUpsertTranslation = async (values: { language_code: string; title: string; description?: string; synopsis?: string }) => {
    await videosApi.upsertTranslation(video.uuid, values.language_code, { title: values.title, description: values.description, synopsis: values.synopsis });
    message.success('翻译已保存');
    transLang.resetFields();
    load();
  };

  const handleGenThumbnails = async () => {
    await videosApi.generateThumbnails(video.uuid, { count: 5, width: 320 });
    message.success('缩略图生成任务已创建');
    load();
  };

  // Subtitle file upload
  const handleSubtitleUpload = async (values: { language_code: string; label: string }) => {
    if (!subFileRef.current) {
      message.error('请选择字幕文件');
      return;
    }
    setSubUploading(true);
    try {
      const fd = new FormData();
      fd.append('file', subFileRef.current);
      fd.append('language_code', values.language_code);
      fd.append('label', values.label);
      await videosApi.uploadSubtitle(video.uuid, fd);
      message.success('字幕上传成功');
      subForm.resetFields();
      subFileRef.current = null;
      load();
    } catch {
      message.error('字幕上传失败');
    } finally {
      setSubUploading(false);
    }
  };

  return (
    <>
      <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/videos')} style={{ marginBottom: 16 }}>返回列表</Button>

      <Tabs defaultActiveKey="info" items={[
        {
          key: 'info', label: '基本信息',
          children: (
            <Card>
              <Descriptions column={2} bordered size="small" style={{ marginBottom: 24 }}>
                <Descriptions.Item label="UUID">{video.uuid}</Descriptions.Item>
                <Descriptions.Item label="状态"><Tag color={statusColors[video.status]}>{video.status}</Tag></Descriptions.Item>
                <Descriptions.Item label="原始文件">{video.original_filename}</Descriptions.Item>
                <Descriptions.Item label="编码">{video.codec || '-'}</Descriptions.Item>
                <Descriptions.Item label="分辨率">{video.width && video.height ? `${video.width}x${video.height}` : '-'}</Descriptions.Item>
                <Descriptions.Item label="时长">{video.duration_seconds ? `${Math.floor(video.duration_seconds / 60)}:${String(Math.floor(video.duration_seconds % 60)).padStart(2, '0')}` : '-'}</Descriptions.Item>
                <Descriptions.Item label="FPS">{video.fps || '-'}</Descriptions.Item>
                <Descriptions.Item label="文件大小">{video.file_size_bytes ? `${(video.file_size_bytes / 1024 / 1024).toFixed(1)} MB` : '-'}</Descriptions.Item>
                <Descriptions.Item label="DRM">{video.has_drm ? '是' : '否'}</Descriptions.Item>
                <Descriptions.Item label="公开">
                  <Switch checked={video.is_public} onChange={async (checked) => { await videosApi.update(video.uuid, { is_public: checked }); message.success('已更新'); load(); }} />
                </Descriptions.Item>
                <Descriptions.Item label="播放次数">{video.view_count}</Descriptions.Item>
                {video.master_playlist_key && (
                  <Descriptions.Item label="播放地址">
                    <Space>
                      <a href={`/play/${video.uuid}/master.m3u8`} target="_blank" rel="noreferrer">/play/{video.uuid}/master.m3u8</a>
                      <Button type="link" size="small" icon={<CopyOutlined />} onClick={() => {
                        const url = `${window.location.origin}/play/${video.uuid}/master.m3u8`;
                        navigator.clipboard.writeText(url);
                        message.success('播放地址已复制');
                      }} />
                    </Space>
                  </Descriptions.Item>
                )}
                {video.status !== 'draft' && (
                  <Descriptions.Item label="下载原片">
                    <Button type="link" size="small" icon={<DownloadOutlined />} href={`/play/${video.uuid}/download`} target="_blank">
                      下载
                    </Button>
                  </Descriptions.Item>
                )}
              </Descriptions>
              <Form layout="inline" initialValues={{ slug: video.slug, rating: video.rating }} onFinish={handleUpdate} form={editForm}>
                <Form.Item name="slug" label="Slug"><Input /></Form.Item>
                <Form.Item name="rating" label="评级"><Input style={{ width: 80 }} /></Form.Item>
                <Form.Item><Button type="primary" htmlType="submit">保存</Button></Form.Item>
              </Form>
            </Card>
          ),
        },
        {
          key: 'preview', label: '预览播放',
          children: (
            <Card>
              {video.master_playlist_key ? (
                <div>
                  <div style={{ background: '#000', borderRadius: 8, overflow: 'hidden', marginBottom: 16 }}>
                    <video
                      ref={(el) => {
                        (previewRef as React.MutableRefObject<HTMLVideoElement | null>).current = el;
                        if (el) initPreviewPlayer();
                      }}
                      controls
                      style={{ width: '100%', maxHeight: 500, display: 'block' }}
                      crossOrigin="anonymous"
                    />
                  </div>
                  <Space>
                    <Link to={`/player/${video.uuid}`}>
                      <Button icon={<PlayCircleOutlined />}>全屏播放器</Button>
                    </Link>
                    <Button icon={<DownloadOutlined />} href={`/play/${video.uuid}/download`} target="_blank">下载原片</Button>
                    <Button icon={<CopyOutlined />} onClick={() => {
                      const url = `${window.location.origin}/play/${video.uuid}/master.m3u8`;
                      navigator.clipboard.writeText(url);
                      message.success('播放地址已复制');
                    }}>复制播放地址</Button>
                  </Space>
                </div>
              ) : (
                <div style={{ textAlign: 'center', padding: 60, color: '#999' }}>
                  <PlayCircleOutlined style={{ fontSize: 48, marginBottom: 16 }} />
                  <p>视频状态: {video.status}，转码完成后可预览</p>
                </div>
              )}
            </Card>
          ),
        },
        {
          key: 'upload', label: '上传视频',
          children: (
            <Card>
              {uploading ? (
                <div style={{ textAlign: 'center', padding: 40 }}>
                  <Progress type="circle" percent={uploadProgress} />
                  <div style={{ marginTop: 16, color: '#666' }}>
                    <p style={{ margin: '4px 0', fontSize: 16 }}>
                      {formatSize(uploadDetail.uploaded)} / {formatSize(uploadDetail.total)}
                    </p>
                    <p style={{ margin: '4px 0' }}>
                      分片 {uploadDetail.part} / {uploadDetail.partCount}
                      {uploadDetail.speed > 0 && ` · ${formatSize(uploadDetail.speed)}/s`}
                    </p>
                    {uploadDetail.speed > 0 && uploadDetail.total > uploadDetail.uploaded && (
                      <p style={{ margin: '4px 0', color: '#999' }}>
                        预计剩余 {Math.ceil((uploadDetail.total - uploadDetail.uploaded) / uploadDetail.speed)}s
                      </p>
                    )}
                  </div>
                </div>
              ) : (
                <Dragger
                  accept="video/*"
                  maxCount={1}
                  beforeUpload={(file) => { handleVideoUpload(file); return false; }}
                  showUploadList={false}
                >
                  <p className="ant-upload-drag-icon"><InboxOutlined /></p>
                  <p className="ant-upload-text">点击或拖拽视频文件到此区域上传</p>
                  <p className="ant-upload-hint">支持 MP4、MKV、AVI 等常见视频格式，文件将通过分片上传到 S3</p>
                </Dragger>
              )}
            </Card>
          ),
        },
        {
          key: 'translations', label: '多语言翻译',
          children: (
            <Card>
              <Table
                dataSource={video.translations || []}
                rowKey="id"
                size="small"
                pagination={false}
                style={{ marginBottom: 24 }}
                columns={[
                  { title: '语言', dataIndex: 'language_code', width: 80 },
                  { title: '标题', dataIndex: 'title' },
                  { title: '描述', dataIndex: 'description', ellipsis: true },
                  {
                    title: '操作', width: 80, render: (_: unknown, r: VideoTranslation) => (
                      <Popconfirm title="确认删除?" onConfirm={async () => { await videosApi.deleteTranslation(video.uuid, r.language_code); message.success('已删除'); load(); }}>
                        <Button type="link" size="small" danger>删除</Button>
                      </Popconfirm>
                    ),
                  },
                ]}
              />
              <Form form={transLang} layout="inline" onFinish={handleUpsertTranslation}>
                <Form.Item name="language_code" rules={[{ required: true }]}><Input placeholder="语言代码 (en/zh)" style={{ width: 120 }} /></Form.Item>
                <Form.Item name="title" rules={[{ required: true }]}><Input placeholder="标题" style={{ width: 200 }} /></Form.Item>
                <Form.Item name="description"><Input placeholder="描述" style={{ width: 200 }} /></Form.Item>
                <Form.Item name="synopsis"><Input placeholder="简介" style={{ width: 200 }} /></Form.Item>
                <Form.Item><Button type="primary" htmlType="submit">添加/更新翻译</Button></Form.Item>
              </Form>
            </Card>
          ),
        },
        {
          key: 'transcode', label: '转码',
          children: (
            <Card>
              <Form form={transForm} layout="inline" onFinish={handleTranscode} style={{ marginBottom: 24 }}>
                <Form.Item name="resolutions" label="分辨率" rules={[{ required: true }]}>
                  <Select mode="multiple" style={{ width: 300 }} placeholder="选择分辨率" options={PRESETS.map((p) => ({ value: p.name, label: `${p.name} (${p.width}x${p.height})` }))} />
                </Form.Item>
                <Form.Item name="codec" label="编码" initialValue="h264">
                  <Select style={{ width: 100 }} options={[{ value: 'h264' }, { value: 'h265' }]} />
                </Form.Item>
                <Form.Item name="drm" label="DRM加密" valuePropName="checked" initialValue={false}>
                  <Switch />
                </Form.Item>
                <Form.Item><Button type="primary" htmlType="submit">开始转码</Button></Form.Item>
              </Form>

              <h4>变体</h4>
              {(video.variants || []).length > 0 && (
                <Popconfirm title="确认删除所有变体？这将删除所有已转码的文件。" onConfirm={async () => { await videosApi.deleteVariants(video.uuid); message.success('变体已删除'); load(); }}>
                  <Button danger icon={<DeleteOutlined />} style={{ marginBottom: 12 }}>删除所有变体</Button>
                </Popconfirm>
              )}
              <Table
                dataSource={video.variants || []}
                rowKey="id"
                size="small"
                pagination={false}
                style={{ marginBottom: 24 }}
                columns={[
                  { title: '分辨率', dataIndex: 'resolution_name' },
                  { title: '尺寸', render: (_: unknown, r: VideoVariant) => `${r.width}x${r.height}` },
                  { title: '码率', dataIndex: 'bitrate_kbps', render: (v: number) => `${v} kbps` },
                  { title: '编码', dataIndex: 'codec' },
                  { title: '状态', dataIndex: 'status', render: (s: string) => <Tag color={statusColors[s]}>{s}</Tag> },
                  {
                    title: '操作', width: 80, render: (_: unknown, r: VideoVariant) => (
                      <Popconfirm title={`确认删除 ${r.resolution_name} 变体？`} onConfirm={async () => { await videosApi.deleteVariant(video.uuid, r.id); message.success('变体已删除'); load(); }}>
                        <Button type="link" size="small" danger>删除</Button>
                      </Popconfirm>
                    ),
                  },
                ]}
              />

              <h4>任务</h4>
              <Table
                dataSource={tasks}
                rowKey="task_uuid"
                size="small"
                pagination={false}
                columns={[
                  { title: '类型', dataIndex: 'type', width: 100 },
                  { title: '状态', dataIndex: 'status', width: 100, render: (s: string) => <Tag color={statusColors[s]}>{s}</Tag> },
                  { title: '进度', dataIndex: 'progress', width: 200, render: (p: number) => <Progress percent={p} size="small" /> },
                  { title: 'Worker', dataIndex: 'worker_id', render: (w?: string) => w || '-' },
                  { title: '创建时间', dataIndex: 'created_at', render: (t: string) => new Date(t).toLocaleString() },
                ]}
              />
            </Card>
          ),
        },
        {
          key: 'thumbnails', label: '缩略图',
          children: (
            <Card>
              <Button type="primary" onClick={handleGenThumbnails} style={{ marginBottom: 16 }}>生成缩略图</Button>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                {(video.thumbnails || []).map((t: Thumbnail) => {
                  const filename = t.s3_key.split('/').pop();
                  const imgUrl = `/play/${video.uuid}/thumbnails/${filename}`;
                  return (
                    <Card key={t.id} size="small" style={{ width: 200 }}
                      cover={<Image src={imgUrl} alt={`thumbnail-${t.id}`} style={{ height: 120, objectFit: 'cover' }} />}
                      actions={[
                        !t.is_default ? <Button type="link" size="small" onClick={async () => { await videosApi.setDefaultThumbnail(video.uuid, t.id); message.success('已设为默认'); load(); }}>设为默认</Button> : <Tag color="green">默认</Tag>,
                        <Popconfirm key="del" title="确认删除?" onConfirm={async () => { await videosApi.deleteThumbnail(video.uuid, t.id); message.success('已删除'); load(); }}>
                          <Button type="link" size="small" danger>删除</Button>
                        </Popconfirm>,
                      ]}
                    >
                      <Card.Meta description={`${t.width}x${t.height} · ${t.timestamp_s.toFixed(1)}s`} />
                    </Card>
                  );
                })}
              </div>
              {(video.thumbnails || []).length === 0 && <p style={{ color: '#999' }}>暂无缩略图，点击上方按钮生成</p>}
            </Card>
          ),
        },
        {
          key: 'subtitles', label: '字幕',
          children: (
            <Card>
              <Table
                dataSource={video.subtitles || []}
                rowKey="id"
                size="small"
                pagination={false}
                style={{ marginBottom: 24 }}
                columns={[
                  { title: '语言', dataIndex: 'language_code' },
                  { title: '标签', dataIndex: 'label' },
                  { title: 'S3 Key', dataIndex: 's3_key', ellipsis: true },
                  { title: '默认', dataIndex: 'is_default', render: (v: boolean) => v ? <Tag color="green">默认</Tag> : null },
                  {
                    title: '操作', render: (_: unknown, r: Subtitle) => (
                      <Popconfirm title="确认删除?" onConfirm={async () => { await videosApi.deleteSubtitle(video.uuid, r.id); message.success('已删除'); load(); }}>
                        <Button type="link" size="small" danger>删除</Button>
                      </Popconfirm>
                    ),
                  },
                ]}
              />
              <Form form={subForm} layout="inline" onFinish={handleSubtitleUpload}>
                <Form.Item name="language_code" rules={[{ required: true, message: '请输入语言代码' }]}>
                  <Input placeholder="语言代码 (en/zh-CN)" style={{ width: 140 }} />
                </Form.Item>
                <Form.Item name="label" rules={[{ required: true, message: '请输入标签' }]}>
                  <Input placeholder="标签 (如: 中文字幕)" style={{ width: 160 }} />
                </Form.Item>
                <Form.Item>
                  <Upload
                    accept=".vtt,.srt,.ass"
                    maxCount={1}
                    beforeUpload={(file) => { subFileRef.current = file; return false; }}
                    onRemove={() => { subFileRef.current = null; }}
                  >
                    <Button icon={<UploadOutlined />}>选择字幕文件</Button>
                  </Upload>
                </Form.Item>
                <Form.Item>
                  <Button type="primary" htmlType="submit" loading={subUploading}>上传字幕</Button>
                </Form.Item>
              </Form>
            </Card>
          ),
        },
        {
          key: 'cast', label: '演职人员',
          children: (
            <Card>
              <Table
                dataSource={video.cast || []}
                rowKey="id"
                size="small"
                pagination={false}
                columns={[
                  { title: '姓名', render: (_: unknown, r: { person?: { translations?: { name: string }[] } }) => r.person?.translations?.[0]?.name || '-' },
                  { title: '角色类型', dataIndex: 'role' },
                  { title: '角色名', dataIndex: 'character' },
                  {
                    title: '操作', render: (_: unknown, r: { id?: number; person?: { translations?: { name: string }[] } }) => (
                      <Popconfirm title="确认移除?" onConfirm={async () => { if (r.id) { await videosApi.deleteCast(video.uuid, r.id); message.success('已移除'); load(); } }}>
                        <Button type="link" size="small" danger>移除</Button>
                      </Popconfirm>
                    ),
                  },
                ]}
              />
            </Card>
          ),
        },
      ]} />
    </>
  );
}
