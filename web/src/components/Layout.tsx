import { useState } from 'react';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import { Layout as AntLayout, Menu, Button, theme, Dropdown } from 'antd';
import {
  VideoCameraOutlined,
  UserOutlined,
  AppstoreOutlined,
  DashboardOutlined,
  ThunderboltOutlined,
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
} from '@ant-design/icons';
import { useAuthStore } from '../store/auth';

const { Header, Sider, Content } = AntLayout;

const menuItems = [
  { key: '/', icon: <DashboardOutlined />, label: '仪表盘' },
  { key: '/videos', icon: <VideoCameraOutlined />, label: '视频管理' },
  { key: '/users', icon: <UserOutlined />, label: '用户管理' },
  { key: '/categories', icon: <AppstoreOutlined />, label: '分类标签' },
  { key: '/tasks', icon: <ThunderboltOutlined />, label: '任务监控' },
];

export default function AppLayout() {
  const [collapsed, setCollapsed] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const { user, logout } = useAuthStore();
  const { token: { colorBgContainer, borderRadiusLG } } = theme.useToken();

  const selectedKey = menuItems.find(
    (item) => item.key !== '/' && location.pathname.startsWith(item.key),
  )?.key || '/';

  return (
    <AntLayout style={{ minHeight: '100vh' }}>
      <Sider trigger={null} collapsible collapsed={collapsed} theme="dark">
        <div style={{ height: 48, margin: 16, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontWeight: 700, fontSize: collapsed ? 16 : 18, whiteSpace: 'nowrap', overflow: 'hidden' }}>
          {collapsed ? 'HLS' : 'HLS Streamer'}
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[selectedKey]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <AntLayout>
        <Header style={{ padding: '0 24px', background: colorBgContainer, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Button
            type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={() => setCollapsed(!collapsed)}
          />
          <Dropdown
            menu={{
              items: [
                { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', danger: true },
              ],
              onClick: ({ key }) => {
                if (key === 'logout') {
                  logout();
                  navigate('/login');
                }
              },
            }}
          >
            <Button type="text">{user?.username || 'User'}</Button>
          </Dropdown>
        </Header>
        <Content style={{ margin: 24, padding: 24, background: colorBgContainer, borderRadius: borderRadiusLG, minHeight: 280 }}>
          <Outlet />
        </Content>
      </AntLayout>
    </AntLayout>
  );
}
