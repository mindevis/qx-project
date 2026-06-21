import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import App from './App';
import { AuthProvider } from '@/auth/AuthContext';
import { BackendStatusProvider } from '@/backend/BackendStatusContext';
import { I18nProvider } from '@/i18n/I18nContext';
import { ThemeProvider } from '@/theme/ThemeContext';
import './index.css';

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <I18nProvider>
      <ThemeProvider>
        <BackendStatusProvider>
          <AuthProvider>
            <App />
          </AuthProvider>
        </BackendStatusProvider>
      </ThemeProvider>
    </I18nProvider>
  </StrictMode>,
);
