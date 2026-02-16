import { useEffect, useState } from 'react';
import { Card, Col, Row, Statistic, Table, Tag } from 'antd';
import { VideoCameraOutlined, ThunderboltOutlined, CloudServerOutlined } from '@ant-design/icons';
import { videosApi } from '../api/videos';
import { workersApi } from '../api/workers';
import type { Video, Worker } from '../types';

const statusColors: Record<string, string> = {
  draft: 'default', processing: 'processing', ready: 'success', error: 'error', archived: 'warning',
  online: 'success', offline: 'default', busy: 'processing',
};

export default function Dashboard() {
  const [videos, setVideos] = useState<Video[]>([]);
  const [workers, setWorkers] = useState<Worker[]>([]);
  const [videoTotal, setVideoTotal] = useState(0);

  useEffect(() => {
    videosApi.list({ per_page: 5 }).then((res) => {
      setVideos(res.data.data);
      setVideoTotal(res.data.meta?.total || 0);
    }).catch(() => {});
    workersApi.list().then((res) => setWorkers(res.data.data || [])).catch(() => {});
  }, []);

  const onlineWorkers = workers.filter((w) => w.status !== 'offline').length;

  return (
    <>
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={8}>
          <Card><Statistic title="视频总数" value={videoTotal} prefix={<VideoCameraOutlined />} /></Card>
        </Col>
        <Col span={8}>
          <Card><Statistic title="在线 Worker" value={onlineWorkers} suffix={`/ ${workers.length}`} prefix={<CloudServerOutlined />} /></Card>
        </Col>
        <Col span={8}>
          <Card><Statistic title="处理中视频" value={videos.filter((v) => v.status === 'processing').length} prefix={<ThunderboltOutlined />} /></Card>
        </Col>
      </Row>

      <Card title="最近视频" style={{ marginBottom: 24 }}>
        <Table
          dataSource={videos}
          rowKey="uuid"
          pagination={false}
          size="small"
          columns={[
            { title: '标题', dataIndex: 'translations', render: (t: Video['translations']) => t?.[0]?.title || '-' },
            { title: '状态', dataIndex: 'status', render: (s: string) => <Tag color={statusColors[s]}>{s}</Tag> },
            { title: '时长', dataIndex: 'duration_seconds', render: (d?: number) => d ? `${Math.floor(d / 60)}:${String(Math.floor(d % 60)).padStart(2, '0')}` : '-' },
            { title: '创建时间', dataIndex: 'created_at', render: (t: string) => new Date(t).toLocaleDateString() },
          ]}
        />
      </Card>

      <Card title="Worker 状态">
        <Table
          dataSource={workers}
          rowKey="id"
          pagination={false}
          size="small"
          columns={[
            { title: 'ID', dataIndex: 'id' },
            { title: '主机名', dataIndex: 'hostname' },
            { title: 'IP', dataIndex: 'ip_address' },
            { title: '状态', dataIndex: 'status', render: (s: string) => <Tag color={statusColors[s]}>{s}</Tag> },
            { title: '最后心跳', dataIndex: 'last_heartbeat', render: (t: string) => t ? new Date(t).toLocaleString() : '-' },
          ]}
        />
      </Card>
    </>
  );
}
