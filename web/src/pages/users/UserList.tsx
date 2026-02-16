import { useEffect, useState } from 'react';
import { Table, Tag, Button, Space, Modal, Form, Input, Select, message, Popconfirm } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { usersApi } from '../../api/users';
import type { User, Role, PaginationMeta } from '../../types';

export default function UserList() {
  const [users, setUsers] = useState<User[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [meta, setMeta] = useState<PaginationMeta>({ page: 1, per_page: 20, total: 0 });
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<User | null>(null);
  const [form] = Form.useForm();

  const fetch = async (page = 1) => {
    setLoading(true);
    try {
      const res = await usersApi.list(page);
      setUsers(res.data.data || []);
      if (res.data.meta) setMeta(res.data.meta);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetch();
    usersApi.listRoles().then((r) => setRoles(r.data.data || [])).catch(() => {});
  }, []);

  const handleSubmit = async (values: { username: string; email: string; password?: string; role_id: number }) => {
    try {
      if (editing) {
        await usersApi.update(editing.id, { username: values.username, email: values.email, role_id: values.role_id });
      } else {
        await usersApi.create({ username: values.username, email: values.email, password: values.password!, role_id: values.role_id });
      }
      message.success(editing ? 'Updated' : 'Created');
      setModalOpen(false);
      setEditing(null);
      form.resetFields();
      fetch(meta.page);
    } catch {
      message.error('Operation failed');
    }
  };

  const openEdit = (user: User) => {
    setEditing(user);
    form.setFieldsValue({ username: user.username, email: user.email, role_id: user.role_id });
    setModalOpen(true);
  };

  const toggleActive = async (user: User) => {
    await usersApi.update(user.id, { is_active: !user.is_active });
    message.success(user.is_active ? 'Deactivated' : 'Activated');
    fetch(meta.page);
  };

  return (
    <>
      <Button type="primary" icon={<PlusOutlined />} onClick={() => { setEditing(null); form.resetFields(); setModalOpen(true); }} style={{ marginBottom: 16 }}>
        New User
      </Button>

      <Table
        dataSource={users}
        rowKey="id"
        loading={loading}
        pagination={{ current: meta.page, pageSize: meta.per_page, total: meta.total, onChange: (p) => fetch(p) }}
        columns={[
          { title: 'Username', dataIndex: 'username' },
          { title: 'Email', dataIndex: 'email' },
          { title: 'Role', dataIndex: 'role', render: (r?: Role) => r ? <Tag>{r.name}</Tag> : '-' },
          { title: 'Status', dataIndex: 'is_active', render: (v: boolean) => <Tag color={v ? 'green' : 'red'}>{v ? 'Active' : 'Inactive'}</Tag> },
          { title: 'Last Login', dataIndex: 'last_login_at', render: (t?: string) => t ? new Date(t).toLocaleString() : '-' },
          {
            title: 'Actions', render: (_: unknown, r: User) => (
              <Space>
                <Button type="link" size="small" onClick={() => openEdit(r)}>Edit</Button>
                <Popconfirm title={r.is_active ? 'Confirm deactivate?' : 'Confirm activate?'} onConfirm={() => toggleActive(r)}>
                  <Button type="link" size="small" danger={r.is_active}>{r.is_active ? 'Deactivate' : 'Activate'}</Button>
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />

      <Modal title={editing ? 'Edit User' : 'New User'} open={modalOpen} onCancel={() => { setModalOpen(false); setEditing(null); }} onOk={() => form.submit()} destroyOnClose>
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item name="username" label="Username" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="email" label="Email" rules={[{ required: true, type: 'email' }]}><Input /></Form.Item>
          {!editing && <Form.Item name="password" label="Password" rules={[{ required: true, min: 8 }]}><Input.Password /></Form.Item>}
          <Form.Item name="role_id" label="Role" rules={[{ required: true }]}>
            <Select options={roles.map((r) => ({ value: r.id, label: r.name }))} />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
