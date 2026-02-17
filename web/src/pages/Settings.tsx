import { useEffect, useState } from 'react';
import { Form, InputNumber, Switch, Button, Card, message, Spin, Descriptions } from 'antd';
import { settingsApi, type AppSetting } from '../api/settings';

export default function Settings() {
  const [settings, setSettings] = useState<AppSetting[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [form] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      const res = await settingsApi.list();
      setSettings(res.data.data || []);
      const values: Record<string, unknown> = {};
      for (const s of res.data.data || []) {
        if (s.value === 'true' || s.value === 'false') {
          values[s.key] = s.value === 'true';
        } else {
          values[s.key] = parseInt(s.value, 10);
        }
      }
      form.setFieldsValue(values);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const handleSave = async (values: Record<string, unknown>) => {
    setSaving(true);
    try {
      for (const s of settings) {
        const newVal = String(values[s.key] ?? s.value);
        if (newVal !== s.value) {
          await settingsApi.update(s.key, newVal);
        }
      }
      message.success('Settings saved');
      load();
    } catch {
      message.error('Failed to save');
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <Spin />;

  return (
    <Card title="Settings">
      <Form form={form} layout="vertical" onFinish={handleSave} style={{ maxWidth: 480 }}>
        <Form.Item
          name="drm_enabled"
          label="DRM Encryption"
          extra="When enabled, all new transcodes will be encrypted with ClearKey DRM."
          valuePropName="checked"
        >
          <Switch />
        </Form.Item>

        <Form.Item
          name="hls_segment_duration"
          label="HLS Segment Duration (seconds)"
          extra="Default segment duration for HLS transcoding. Each transcode job can override this value."
          rules={[{ required: true, message: 'Required' }]}
        >
          <InputNumber min={1} max={60} addonAfter="s" style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item>
          <Button type="primary" htmlType="submit" loading={saving}>Save</Button>
        </Form.Item>
      </Form>

      <Descriptions title="All Settings" column={1} size="small" bordered style={{ marginTop: 24 }}>
        {settings.map((s) => (
          <Descriptions.Item key={s.key} label={s.key}>{s.value} — {s.description}</Descriptions.Item>
        ))}
      </Descriptions>
    </Card>
  );
}
