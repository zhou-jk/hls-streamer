import { useEffect, useState } from 'react';
import { Table, Tag, Card, Row, Col, Progress, Select, Space, Button } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { videosApi } from '../../api/videos';
import { workersApi } from '../../api/workers';
import type { TranscodeTask, Worker } from '../../types';

const statusColors: Record<string, string> = {
  pending: 'default', queued: 'cyan', processing: 'processing', completed: 'success', failed: 'error', cancelled: 'warning',
  online: 'success', offline: 'default', busy: 'processing',
};

export default function TaskList() {
  const [tasks, setTasks] = useState<TranscodeTask[]>([]);
  const [workers, setWorkers] = useState<Worker[]>([]);
  const [statusFilter, setStatusFilter] = useState<string>();
  const [typeFilter, setTypeFilter] = useState<string>();

  const fetchWorkers = () => workersApi.list().then((r) => setWorkers(r.data.data || [])).catch(() => {});

  // We need to fetch tasks from all videos — use a simple approach:
  // fetch recent videos and aggregate their tasks
  const fetchTasks = async () => {
    try {
      const vRes = await videosApi.list({ per_page: 50 });
      const videos = vRes.data.data || [];
      const allTasks: TranscodeTask[] = [];
      await Promise.all(
        videos.map(async (v) => {
          try {
            const tRes = await videosApi.listTasks(v.uuid);
            const t = tRes.data.data || [];
            allTasks.push(...t);
          } catch { /* skip */ }
        }),
      );
      allTasks.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
      setTasks(allTasks);
    } catch { /* skip */ }
  };

  useEffect(() => { fetchTasks(); fetchWorkers(); }, []);

  // Auto-refresh if any tasks are active
  useEffect(() => {
    if (!tasks.some((t) => ['queued', 'processing', 'pending'].includes(t.status))) return;
    const timer = setInterval(() => { fetchTasks(); fetchWorkers(); }, 5000);
    return () => clearInterval(timer);
  }, [tasks]);

  const filtered = tasks.filter((t) => {
    if (statusFilter && t.status !== statusFilter) return false;
    if (typeFilter && t.type !== typeFilter) return false;
    return true;
  });

  return (
    <>
      <Row gutter={16} style={{ marginBottom: 24 }}>
        {workers.map((w) => (
          <Col key={w.id} span={6}>
            <Card size="small">
              <div style={{ fontWeight: 600 }}>{w.hostname || w.id}</div>
              <div>IP: {w.ip_address}</div>
              <Tag color={statusColors[w.status]} style={{ marginTop: 4 }}>{w.status}</Tag>
              <div style={{ fontSize: 12, color: '#999', marginTop: 4 }}>
                心跳: {w.last_heartbeat ? new Date(w.last_heartbeat).toLocaleString() : '-'}
              </div>
            </Card>
          </Col>
        ))}
        {workers.length === 0 && <Col span={24}><Card size="small">暂无 Worker 在线</Card></Col>}
      </Row>

      <Space style={{ marginBottom: 16 }}>
        <Select placeholder="状态" allowClear style={{ width: 120 }} value={statusFilter} onChange={setStatusFilter} options={[
          { value: 'pending', label: 'Pending' }, { value: 'queued', label: 'Queued' },
          { value: 'processing', label: 'Processing' }, { value: 'completed', label: 'Completed' },
          { value: 'failed', label: 'Failed' }, { value: 'cancelled', label: 'Cancelled' },
        ]} />
        <Select placeholder="类型" allowClear style={{ width: 120 }} value={typeFilter} onChange={setTypeFilter} options={[
          { value: 'probe', label: 'Probe' }, { value: 'transcode', label: 'Transcode' },
          { value: 'thumbnail', label: 'Thumbnail' }, { value: 'drm_package', label: 'DRM' },
        ]} />
        <Button icon={<ReloadOutlined />} onClick={() => { fetchTasks(); fetchWorkers(); }}>刷新</Button>
      </Space>

      <Table
        dataSource={filtered}
        rowKey="task_uuid"
        size="small"
        pagination={{ pageSize: 20 }}
        columns={[
          { title: '任务 ID', dataIndex: 'task_uuid', width: 280, ellipsis: true },
          { title: '类型', dataIndex: 'type', width: 100, render: (t: string) => <Tag>{t}</Tag> },
          { title: '状态', dataIndex: 'status', width: 110, render: (s: string) => <Tag color={statusColors[s]}>{s}</Tag> },
          { title: '进度', dataIndex: 'progress', width: 180, render: (p: number, r: TranscodeTask) => r.status === 'processing' || r.status === 'completed' ? <Progress percent={p} size="small" /> : '-' },
          { title: 'Worker', dataIndex: 'worker_id', width: 150, render: (w?: string) => w || '-' },
          { title: '尝试', key: 'attempts', width: 80, render: (_: unknown, r: TranscodeTask) => `${r.attempts}/${r.max_attempts}` },
          { title: '错误', dataIndex: 'error_message', ellipsis: true, render: (e?: string) => e ? <span style={{ color: 'red' }}>{e}</span> : '-' },
          { title: '创建时间', dataIndex: 'created_at', width: 170, render: (t: string) => new Date(t).toLocaleString() },
        ]}
      />
    </>
  );
}
