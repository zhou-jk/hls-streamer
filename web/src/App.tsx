import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider } from 'antd';
import enUS from 'antd/locale/en_US';
import { useEffect } from 'react';
import { useAuthStore } from './store/auth';
import PrivateRoute from './components/PrivateRoute';
import AppLayout from './components/Layout';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import VideoList from './pages/videos/VideoList';
import VideoDetail from './pages/videos/VideoDetail';
import VideoPlayer from './pages/videos/VideoPlayer';
import UserList from './pages/users/UserList';
import CategoryList from './pages/categories/CategoryList';
import TaskList from './pages/tasks/TaskList';
import Settings from './pages/Settings';
import Home from './pages/Home';
import Watch from './pages/Watch';

export default function App() {
  const init = useAuthStore((s) => s.init);
  useEffect(() => { init(); }, []);

  return (
    <ConfigProvider locale={enUS}>
      <BrowserRouter>
        <Routes>
          {/* Public pages */}
          <Route path="/" element={<Home />} />
          <Route path="/watch/:uuid" element={<Watch />} />

          {/* Admin */}
          <Route path="/admin/login" element={<Login />} />
          <Route path="/admin/player/:uuid" element={<PrivateRoute><VideoPlayer /></PrivateRoute>} />
          <Route path="/admin" element={<PrivateRoute><AppLayout /></PrivateRoute>}>
            <Route index element={<Dashboard />} />
            <Route path="videos" element={<VideoList />} />
            <Route path="videos/:uuid" element={<VideoDetail />} />
            <Route path="users" element={<UserList />} />
            <Route path="categories" element={<CategoryList />} />
            <Route path="tasks" element={<TaskList />} />
            <Route path="settings" element={<Settings />} />
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  );
}
