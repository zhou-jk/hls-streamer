import { useEffect, useState } from 'react';
import { Tabs, Table, Tag, Button, Space, Modal, Form, Input, message, Popconfirm } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { categoriesApi, tagsApi } from '../../api/categories';
import type { Category, Tag as TagType } from '../../types';

export default function CategoryList() {
  const [categories, setCategories] = useState<Category[]>([]);
  const [tags, setTags] = useState<TagType[]>([]);
  const [catModalOpen, setCatModalOpen] = useState(false);
  const [tagModalOpen, setTagModalOpen] = useState(false);
  const [transModalOpen, setTransModalOpen] = useState(false);
  const [selectedCat, setSelectedCat] = useState<Category | null>(null);
  const [catForm] = Form.useForm();
  const [tagForm] = Form.useForm();
  const [transForm] = Form.useForm();

  const fetchCategories = () => categoriesApi.list().then((r) => setCategories(r.data.data || [])).catch(() => {});
  const fetchTags = () => tagsApi.list().then((r) => setTags(r.data.data || [])).catch(() => {});

  useEffect(() => { fetchCategories(); fetchTags(); }, []);

  const handleCreateCategory = async (values: { slug: string }) => {
    await categoriesApi.create(values);
    message.success('Category created');
    setCatModalOpen(false);
    catForm.resetFields();
    fetchCategories();
  };

  const handleCreateTag = async (values: { slug: string }) => {
    await tagsApi.create(values);
    message.success('Tag created');
    setTagModalOpen(false);
    tagForm.resetFields();
    fetchTags();
  };

  const handleUpsertTranslation = async (values: { language_code: string; name: string; description?: string }) => {
    if (!selectedCat) return;
    await categoriesApi.upsertTranslation(selectedCat.id, values.language_code, { name: values.name, description: values.description });
    message.success('Translation saved');
    setTransModalOpen(false);
    transForm.resetFields();
    fetchCategories();
  };

  return (
    <Tabs items={[
      {
        key: 'categories', label: 'Categories',
        children: (
          <>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setCatModalOpen(true)} style={{ marginBottom: 16 }}>New Category</Button>
            <Table
              dataSource={categories}
              rowKey="id"
              size="small"
              columns={[
                { title: 'ID', dataIndex: 'id', width: 60 },
                { title: 'Slug', dataIndex: 'slug' },
                { title: 'Name', key: 'name', render: (_: unknown, r: Category) => r.translations?.[0]?.name || '-' },
                { title: 'Sort', dataIndex: 'sort_order', width: 80 },
                { title: 'Status', dataIndex: 'is_active', width: 80, render: (v: boolean) => <Tag color={v ? 'green' : 'red'}>{v ? 'Active' : 'Inactive'}</Tag> },
                {
                  title: 'Actions', width: 200, render: (_: unknown, r: Category) => (
                    <Space>
                      <Button type="link" size="small" onClick={() => { setSelectedCat(r); setTransModalOpen(true); }}>Translate</Button>
                      <Popconfirm title="Confirm delete?" onConfirm={async () => { await categoriesApi.delete(r.id); message.success('Deleted'); fetchCategories(); }}>
                        <Button type="link" size="small" danger>Delete</Button>
                      </Popconfirm>
                    </Space>
                  ),
                },
              ]}
            />
            <Modal title="New Category" open={catModalOpen} onCancel={() => setCatModalOpen(false)} onOk={() => catForm.submit()} destroyOnClose>
              <Form form={catForm} layout="vertical" onFinish={handleCreateCategory}>
                <Form.Item name="slug" label="Slug" rules={[{ required: true }]}><Input /></Form.Item>
              </Form>
            </Modal>
            <Modal title={`Translation - ${selectedCat?.slug}`} open={transModalOpen} onCancel={() => setTransModalOpen(false)} onOk={() => transForm.submit()} destroyOnClose>
              <Form form={transForm} layout="vertical" onFinish={handleUpsertTranslation}>
                <Form.Item name="language_code" label="Language Code" rules={[{ required: true }]}><Input placeholder="en / zh / ja" /></Form.Item>
                <Form.Item name="name" label="Name" rules={[{ required: true }]}><Input /></Form.Item>
                <Form.Item name="description" label="Description"><Input.TextArea rows={2} /></Form.Item>
              </Form>
            </Modal>
          </>
        ),
      },
      {
        key: 'tags', label: 'Tags',
        children: (
          <>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setTagModalOpen(true)} style={{ marginBottom: 16 }}>New Tag</Button>
            <Table
              dataSource={tags}
              rowKey="id"
              size="small"
              columns={[
                { title: 'ID', dataIndex: 'id', width: 60 },
                { title: 'Slug', dataIndex: 'slug' },
                { title: 'Name', key: 'name', render: (_: unknown, r: TagType) => r.translations?.[0]?.name || '-' },
                {
                  title: 'Actions', width: 100, render: (_: unknown, r: TagType) => (
                    <Popconfirm title="Confirm delete?" onConfirm={async () => { await tagsApi.delete(r.id); message.success('Deleted'); fetchTags(); }}>
                      <Button type="link" size="small" danger>Delete</Button>
                    </Popconfirm>
                  ),
                },
              ]}
            />
            <Modal title="New Tag" open={tagModalOpen} onCancel={() => setTagModalOpen(false)} onOk={() => tagForm.submit()} destroyOnClose>
              <Form form={tagForm} layout="vertical" onFinish={handleCreateTag}>
                <Form.Item name="slug" label="Slug" rules={[{ required: true }]}><Input /></Form.Item>
              </Form>
            </Modal>
          </>
        ),
      },
    ]} />
  );
}
