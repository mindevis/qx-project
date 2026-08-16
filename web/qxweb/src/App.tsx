import { lazy, Suspense, type ReactNode } from 'react';
import { Spin } from 'antd';
import { BrowserRouter, Route, Routes } from 'react-router-dom';
import { AuthModalProvider } from '@/auth/AuthModalContext';
import { AppLayout } from '@/layouts/AppLayout';
import { HomePage } from '@/pages/HomePage';

const ProfilePage = lazy(() =>
  import('@/pages/ProfilePage').then((module) => ({ default: module.ProfilePage })),
);
const LauncherLinkPage = lazy(() =>
  import('@/pages/LauncherLinkPage').then((module) => ({ default: module.LauncherLinkPage })),
);
const LauncherPage = lazy(() =>
  import('@/pages/LauncherPage').then((module) => ({ default: module.LauncherPage })),
);
const AuthRedirect = lazy(() =>
  import('@/pages/AuthRedirect').then((module) => ({ default: module.AuthRedirect })),
);
const MonitoringPage = lazy(() =>
  import('@/pages/MonitoringPage').then((module) => ({ default: module.MonitoringPage })),
);
const ServersPage = lazy(() =>
  import('@/pages/ServersPage').then((module) => ({ default: module.ServersPage })),
);
const NotFoundPage = lazy(() =>
  import('@/pages/NotFoundPage').then((module) => ({ default: module.NotFoundPage })),
);
const SkinsPage = lazy(() =>
  import('@/pages/SkinsPage').then((module) => ({ default: module.SkinsPage })),
);

function RouteFallback() {
  return (
    <div style={{ display: 'flex', justifyContent: 'center', padding: '48px 0' }}>
      <Spin size="large" />
    </div>
  );
}

function LazyRoute({ children }: { children: ReactNode }) {
  return <Suspense fallback={<RouteFallback />}>{children}</Suspense>;
}

export default function App() {
  return (
    <BrowserRouter>
      <AuthModalProvider>
        <Routes>
          <Route element={<AppLayout />}>
            <Route index element={<HomePage />} />
            <Route
              path="auth/:mode"
              element={
                <LazyRoute>
                  <AuthRedirect />
                </LazyRoute>
              }
            />
            <Route
              path="profile"
              element={
                <LazyRoute>
                  <ProfilePage />
                </LazyRoute>
              }
            />
            <Route
              path="skins"
              element={
                <LazyRoute>
                  <SkinsPage />
                </LazyRoute>
              }
            />
            <Route
              path="launcher/link"
              element={
                <LazyRoute>
                  <LauncherLinkPage />
                </LazyRoute>
              }
            />
            <Route
              path="launcher/*"
              element={
                <LazyRoute>
                  <LauncherPage />
                </LazyRoute>
              }
            />
            <Route
              path="monitoring"
              element={
                <LazyRoute>
                  <MonitoringPage />
                </LazyRoute>
              }
            />
            <Route
              path="servers/*"
              element={
                <LazyRoute>
                  <ServersPage />
                </LazyRoute>
              }
            />
            <Route
              path="*"
              element={
                <LazyRoute>
                  <NotFoundPage />
                </LazyRoute>
              }
            />
          </Route>
        </Routes>
      </AuthModalProvider>
    </BrowserRouter>
  );
}
