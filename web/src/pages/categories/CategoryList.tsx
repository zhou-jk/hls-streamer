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
    message.success('分类已创建');
    setCatModalOpen(false);
    catForm.resetFields();
    fetchCategories();
  };

  const handleCreateTag = async (values: { slug: string }) => {
    await tagsApi.create(values);
    message.success('标签已创建');
    setTagModalOpen(false);
    tagForm.resetFields();
    fetchTags();
  };

  const handleUpsertTranslation = async (values: { language_code: string; name: string; description?: string }) => {
    if (!selectedCat) return;
    await categoriesApi.upsertTranslation(selectedCat.id, values.language_code, { name: values.name, description: values.description });
    message.success('翻译已保存');
    setTransModalOpen(false);
    transForm.resetFields();
    fetchCategories();
  };

  return (
    <Tabs items={[
      {
        key: 'categories', label: '分类',
        children: (
          <>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setCatModalOpen(true)} style={{ marginBottom: 16 }}>新建分类</Button>
            <Table
              dataSource={categories}
              rowKey="id"
              size="small"
              columns={[
                { title: 'ID', dataIndex: 'id', width: 60 },
                { title: 'Slug', dataIndex: 'slug' },
                { title: '名称', key: 'name', render: (_: unknown, r: Category) => r.translations?.[0]?.name || '-' },
                { title: '排序', dataIndex: 'sort_order', width: 80 },
                { title: '状态', dataIndex: 'is_active', width: 80, render: (v: boolean) => <Tag color={v ? 'green' : 'red'}>{v ? '启用' : '停用'}</Tag> },
                {
                  title: '操作', width: 200, render: (_: unknown, r: Category) => (
                    <Space>
                      <Button type="link" size="small" onClick={() => { setSelectedCat(r); setTransModalOpen(true); }}>翻译</Button>
                      <Popconfirm title="确认删除?" onConfirm={async () => { await categoriesApi.delete(r.id); message.success('已删除'); fetchCategories(); }}>
                        <Button type="link" size="small" danger>删除</Button>
                      </Popconfirm>
                    </Space>
                  ),
                },
              ]}
            />
            <Modal title="新建分类" open={catModalOpen} onCancel={() => setCatModalOpen(false)} onOk={() => catForm.submit()} destroyOnClose>
              <Form form={catForm} layout="vertical" onFinish={handleCreateCategory}>
                <Form.Item name="slug" label="Slug" rules={[{ required: true }]}><Input /></Form.Item>
              </Form>
            </Modal>
            <Modal title={`翻译 - ${selectedCat?.slug}`} open={transModalOpen} onCancel={() => setTransModalOpen(false)} onOk={() => transForm.submit()} destroyOnClose>
              <Form form={transForm} layout="vertical" onFinish={handleUpsertTranslation}>
                <Form.Item name="language_code" label="语言代码" rules={[{ required: true }]}><Input placeholder="en / zh / ja" /></Form.Item>
                <Form.Item name="name" label="名称" rules={[{ required: true }]}><Input /></Form.Item>
                <Form.Item name="description" label="描述"><Input.TextArea rows={2} /></Form.Item>
              </Form>
            </Modal>
          </>
        ),
      },
      {
        key: 'tags', label: '标签',
        children: (
          <>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setTagModalOpen(true)} style={{ marginBottom: 16 }}>新建标签</Button>
            <Table
              dataSource={tags}
              rowKey="id"
              size="small"
              columns={[
                { title: 'ID', dataIndex: 'id', width: 60 },
                { title: 'Slug', dataIndex: 'slug' },
                { title: '名称', key: 'name', render: (_: unknown, r: TagType) => r.translations?.[0]?.name || '-' },
                {
                  title: '操作', width: 100, render: (_: unknown, r: TagType) => (
                    <Popconfirm title="确认删除?" onConfirm={async () => { await tagsApi.delete(r.id); message.success('已删除'); fetchTags(); }}>
                      <Button type="link" size="small" danger>删除</Button>
                    </Popconfirm>
                  ),
                },
              ]}
            />
            <Modal title="新建标签" open={tagModalOpen} onCancel={() => setTagModalOpen(false)} onOk={() => tagForm.submit()} destroyOnClose>
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
