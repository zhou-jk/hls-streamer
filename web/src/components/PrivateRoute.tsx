import { Navigate } from 'react-router-dom';
import { useAuthStore } from '../store/auth';
import { Spin } from 'antd';

export default function PrivateRoute({ children }: { children: React.ReactNode }) {
  const { token, loading } = useAuthStore();
  if (loading) return <Spin size="large" style={{ display: 'block', margin: '200px auto' }} />;
  if (!token) return <Navigate to="/login" replace />;
  return <>{children}</>;
}
