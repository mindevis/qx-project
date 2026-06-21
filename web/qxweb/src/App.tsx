import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';
import { AuthModalProvider } from '@/auth/AuthModalContext';
import { AppLayout } from '@/layouts/AppLayout';
import { HomePage } from '@/pages/HomePage';
import { LauncherLinkPage } from '@/pages/LauncherLinkPage';
import { LauncherPage } from '@/pages/LauncherPage';
import { AuthRedirect } from '@/pages/AuthRedirect';
import { ProfilePage } from '@/pages/ProfilePage';
import { ServersPage } from '@/pages/ServersPage';

export default function App() {
  return (
    <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
      <AuthModalProvider>
        <Routes>
          <Route element={<AppLayout />}>
            <Route index element={<HomePage />} />
              <Route path="auth/:mode" element={<AuthRedirect />} />
            <Route path="profile" element={<ProfilePage />} />
            <Route path="launcher/link" element={<LauncherLinkPage />} />
            <Route path="launcher/*" element={<LauncherPage />} />
            <Route path="servers/*" element={<ServersPage />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Route>
        </Routes>
      </AuthModalProvider>
    </BrowserRouter>
  );
}
