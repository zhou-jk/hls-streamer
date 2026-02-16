import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Table, Tag, Button, Space, Input, Select, Modal, Form, message, Popconfirm } from 'antd';
import { PlusOutlined, SearchOutlined, ReloadOutlined } from '@ant-design/icons';
import { videosApi } from '../../api/videos';
import type { Video, PaginationMeta } from '../../types';

const statusColors: Record<string, string> = {
  draft: 'default', uploaded: 'blue', processing: 'processing', ready: 'success', error: 'error', archived: 'warning',
};

export default function VideoList() {
  const [videos, setVideos] = useState<Video[]>([]);
  const [meta, setMeta] = useState<PaginationMeta>({ page: 1, per_page: 20, total: 0 });
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>();
  const [createOpen, setCreateOpen] = useState(false);
  const [form] = Form.useForm();
  const navigate = useNavigate();

  const fetchVideos = async (page = 1) => {
    setLoading(true);
    try {
      const res = await videosApi.list({ page, per_page: 20, q: search || undefined, status: statusFilter });
      setVideos(res.data.data || []);
      if (res.data.meta) setMeta(res.data.meta);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchVideos(); }, []);

  const handleCreate = async (values: Record<string, string>) => {
    try {
      const res = await videosApi.create({
        slug: '',
        original_filename: '',
        language: values.language || 'en',
        title: values.title,
        description: values.description,
      });
      message.success('视频创建成功');
      setCreateOpen(false);
      form.resetFields();
      navigate(`/videos/${res.data.data.uuid}`);
    } catch {
      message.error('创建失败');
    }
  };

  const handleDelete = async (uuid: string) => {
    await videosApi.delete(uuid);
    message.success('已删除');
    fetchVideos(meta.page);
  };

  return (
    <>
      <Space style={{ marginBottom: 16 }} wrap>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>新建视频</Button>
        <Input placeholder="搜索标题" prefix={<SearchOutlined />} value={search} onChange={(e) => setSearch(e.target.value)} onPressEnter={() => fetchVideos()} style={{ width: 200 }} />
        <Select placeholder="状态筛选" allowClear style={{ width: 120 }} value={statusFilter} onChange={(v) => { setStatusFilter(v); }} options={[
          { value: 'draft', label: 'Draft' }, { value: 'processing', label: 'Processing' },
          { value: 'ready', label: 'Ready' }, { value: 'error', label: 'Error' },
        ]} />
        <Button icon={<ReloadOutlined />} onClick={() => fetchVideos()}>刷新</Button>
      </Space>

      <Table
        dataSource={videos}
        rowKey="uuid"
        loading={loading}
        pagination={{
          current: meta.page, pageSize: meta.per_page, total: meta.total,
          onChange: (page) => fetchVideos(page),
        }}
        columns={[
          { title: '标题', key: 'title', render: (_: unknown, r: Video) => r.translations?.[0]?.title || r.slug },
          { title: '状态', dataIndex: 'status', width: 120, render: (s: string) => <Tag color={statusColors[s]}>{s}</Tag> },
          { title: '公开', dataIndex: 'is_public', width: 80, render: (v: boolean) => v ? <Tag color="green">是</Tag> : <Tag>否</Tag> },
          { title: '分辨率', key: 'res', width: 120, render: (_: unknown, r: Video) => r.width && r.height ? `${r.width}x${r.height}` : '-' },
          { title: '时长', dataIndex: 'duration_seconds', width: 100, render: (d?: number) => d ? `${Math.floor(d / 60)}:${String(Math.floor(d % 60)).padStart(2, '0')}` : '-' },
          { title: '创建时间', dataIndex: 'created_at', width: 120, render: (t: string) => new Date(t).toLocaleDateString() },
          {
            title: '操作', width: 160, render: (_: unknown, r: Video) => (
              <Space>
                <Button type="link" size="small" onClick={() => navigate(`/videos/${r.uuid}`)}>详情</Button>
                <Popconfirm title="确认删除?" onConfirm={() => handleDelete(r.uuid)}>
                  <Button type="link" size="small" danger>删除</Button>
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />

      <Modal title="新建视频" open={createOpen} onCancel={() => setCreateOpen(false)} onOk={() => form.submit()} destroyOnClose>
        <Form form={form} layout="vertical" onFinish={handleCreate}>
          <Form.Item name="language" label="语言" initialValue="en"><Input /></Form.Item>
          <Form.Item name="title" label="标题" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="description" label="描述"><Input.TextArea rows={3} /></Form.Item>
        </Form>
      </Modal>
    </>
  );
}
