import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Tabs, Descriptions, Tag, Button, Form, Input, Select, Table, Space, Card, message, Popconfirm, Progress } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { videosApi } from '../../api/videos';
import type { Video, VideoTranslation, VideoVariant, Thumbnail, Subtitle, TranscodeTask } from '../../types';

const statusColors: Record<string, string> = {
  draft: 'default', processing: 'processing', ready: 'success', error: 'error', archived: 'warning',
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

export default function VideoDetail() {
  const { uuid } = useParams<{ uuid: string }>();
  const navigate = useNavigate();
  const [video, setVideo] = useState<Video | null>(null);
  const [tasks, setTasks] = useState<TranscodeTask[]>([]);
  const [transForm] = Form.useForm();
  const [transLang] = Form.useForm();
  const [editForm] = Form.useForm();

  const load = () => {
    if (!uuid) return;
    videosApi.get(uuid).then((r) => setVideo(r.data.data)).catch(() => message.error('视频不存在'));
    videosApi.listTasks(uuid).then((r) => setTasks(r.data.data || [])).catch(() => {});
  };

  useEffect(() => { load(); }, [uuid]);
  // Poll tasks every 5s if any are processing
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

  const handleTranscode = async (values: { resolutions: string[]; codec: string }) => {
    const resolutions = PRESETS.filter((p) => values.resolutions.includes(p.name));
    await videosApi.startTranscode(video.uuid, { resolutions, codec: values.codec || 'h264' });
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
                <Descriptions.Item label="播放次数">{video.view_count}</Descriptions.Item>
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
                <Form.Item><Button type="primary" htmlType="submit">开始转码</Button></Form.Item>
              </Form>

              <h4>变体</h4>
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
              <Table
                dataSource={video.thumbnails || []}
                rowKey="id"
                size="small"
                pagination={false}
                columns={[
                  { title: 'S3 Key', dataIndex: 's3_key', ellipsis: true },
                  { title: '尺寸', render: (_: unknown, r: Thumbnail) => `${r.width}x${r.height}` },
                  { title: '时间点', dataIndex: 'timestamp_s', render: (t: number) => `${t.toFixed(1)}s` },
                  { title: '默认', dataIndex: 'is_default', render: (v: boolean) => v ? <Tag color="green">默认</Tag> : null },
                  {
                    title: '操作', render: (_: unknown, r: Thumbnail) => (
                      <Space>
                        {!r.is_default && <Button type="link" size="small" onClick={async () => { await videosApi.setDefaultThumbnail(video.uuid, r.id); message.success('已设为默认'); load(); }}>设为默认</Button>}
                        <Popconfirm title="确认删除?" onConfirm={async () => { await videosApi.deleteThumbnail(video.uuid, r.id); message.success('已删除'); load(); }}>
                          <Button type="link" size="small" danger>删除</Button>
                        </Popconfirm>
                      </Space>
                    ),
                  },
                ]}
              />
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
