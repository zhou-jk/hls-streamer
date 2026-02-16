import { useEffect, useState } from 'react';
import { Table, Tag, Card, Row, Col, Select, Space, Button, Progress, message, Popconfirm } from 'antd';
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
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [workers, setWorkers] = useState<Worker[]>([]);
  const [statusFilter, setStatusFilter] = useState<string>();
  const [typeFilter, setTypeFilter] = useState<string>();

  const fetchWorkers = () => workersApi.list().then((r) => setWorkers(r.data.data || [])).catch(() => {});

  const fetchTasks = (p = page) => {
    videosApi.listAllTasks({ page: p, per_page: 20, status: statusFilter, type: typeFilter })
      .then((r) => {
        setTasks(r.data.data || []);
        setTotal(r.data.meta?.total || 0);
      })
      .catch(() => {});
  };

  useEffect(() => { fetchTasks(1); setPage(1); fetchWorkers(); }, [statusFilter, typeFilter]);

  // Auto-refresh if any tasks are active
  useEffect(() => {
    if (!tasks.some((t) => ['queued', 'processing', 'pending'].includes(t.status))) return;
    const timer = setInterval(() => { fetchTasks(); fetchWorkers(); }, 5000);
    return () => clearInterval(timer);
  }, [tasks]);

  const handleCancel = async (taskUuid: string) => {
    await videosApi.cancelTask(taskUuid);
    message.success('Task cancelled');
    fetchTasks();
  };

  const handleRetry = async (taskUuid: string) => {
    await videosApi.retryTask(taskUuid);
    message.success('Task retried');
    fetchTasks();
  };

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
                Heartbeat: {w.last_heartbeat ? new Date(w.last_heartbeat).toLocaleString() : '-'}
              </div>
            </Card>
          </Col>
        ))}
        {workers.length === 0 && <Col span={24}><Card size="small">No workers online</Card></Col>}
      </Row>

      <Space style={{ marginBottom: 16 }}>
        <Select placeholder="Status" allowClear style={{ width: 120 }} value={statusFilter} onChange={setStatusFilter} options={[
          { value: 'pending', label: 'Pending' }, { value: 'queued', label: 'Queued' },
          { value: 'processing', label: 'Processing' }, { value: 'completed', label: 'Completed' },
          { value: 'failed', label: 'Failed' }, { value: 'cancelled', label: 'Cancelled' },
        ]} />
        <Select placeholder="Type" allowClear style={{ width: 120 }} value={typeFilter} onChange={setTypeFilter} options={[
          { value: 'probe', label: 'Probe' }, { value: 'transcode', label: 'Transcode' },
          { value: 'thumbnail', label: 'Thumbnail' }, { value: 'drm_package', label: 'DRM' },
        ]} />
        <Button icon={<ReloadOutlined />} onClick={() => { fetchTasks(); fetchWorkers(); }}>Refresh</Button>
      </Space>

      <Table
        dataSource={tasks}
        rowKey="task_uuid"
        size="small"
        pagination={{
          current: page,
          pageSize: 20,
          total,
          onChange: (p) => { setPage(p); fetchTasks(p); },
          showTotal: (t) => `${t} total`,
        }}
        columns={[
          { title: 'Task ID', dataIndex: 'task_uuid', width: 280, ellipsis: true },
          { title: 'Type', dataIndex: 'type', width: 100, render: (t: string) => <Tag>{t}</Tag> },
          { title: 'Status', dataIndex: 'status', width: 110, render: (s: string) => <Tag color={statusColors[s]}>{s}</Tag> },
          { title: 'Progress', dataIndex: 'progress', width: 180, render: (p: number, r: TranscodeTask) => r.status === 'processing' || r.status === 'completed' ? <Progress percent={p} size="small" /> : '-' },
          { title: 'Worker', dataIndex: 'worker_id', width: 150, render: (w?: string) => w || '-' },
          { title: 'Attempts', key: 'attempts', width: 80, render: (_: unknown, r: TranscodeTask) => `${r.attempts}/${r.max_attempts}` },
          { title: 'Error', dataIndex: 'error_message', ellipsis: true, render: (e?: string) => e ? <span style={{ color: 'red' }}>{e}</span> : '-' },
          { title: 'Created', dataIndex: 'created_at', width: 170, render: (t: string) => new Date(t).toLocaleString() },
          {
            title: 'Actions', width: 130, render: (_: unknown, r: TranscodeTask) => (
              <Space size="small">
                {['pending', 'queued', 'processing'].includes(r.status) && (
                  <Popconfirm title="Confirm cancel?" onConfirm={() => handleCancel(r.task_uuid)}>
                    <Button type="link" size="small" danger>Cancel</Button>
                  </Popconfirm>
                )}
                {r.status === 'failed' && (
                  <Button type="link" size="small" onClick={() => handleRetry(r.task_uuid)}>Retry</Button>
                )}
              </Space>
            ),
          },
        ]}
      />
    </>
  );
}
