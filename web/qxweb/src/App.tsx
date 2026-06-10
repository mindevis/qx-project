import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';
import { AuthModalProvider } from '@/auth/AuthModalContext';
import { AppLayout } from '@/layouts/AppLayout';
import { HomePage } from '@/pages/HomePage';
import { LauncherPage } from '@/pages/LauncherPage';
import { AuthRedirect } from '@/pages/AuthRedirect';
import { ProfilePage } from '@/pages/ProfilePage';
import { PlaceholderPage } from '@/pages/PlaceholderPage';

export default function App() {
  return (
    <BrowserRouter>
      <AuthModalProvider>
        <Routes>
          <Route element={<AppLayout />}>
            <Route index element={<HomePage />} />
              <Route path="auth/:mode" element={<AuthRedirect />} />
            <Route path="profile" element={<ProfilePage />} />
            <Route path="launcher/*" element={<LauncherPage />} />
            <Route
              path="servers/*"
              element={<PlaceholderPage title="Серверы" phase="Phase 2" />}
            />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Route>
        </Routes>
      </AuthModalProvider>
    </BrowserRouter>
  );
}
