import { useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useAuthModal, type AuthMode } from '@/auth/AuthModalContext';

export function AuthRedirect() {
  const { mode } = useParams<{ mode: string }>();
  const { openAuthModal } = useAuthModal();
  const navigate = useNavigate();

  useEffect(() => {
    const authMode: AuthMode = mode === 'register' ? 'register' : 'login';
    openAuthModal(authMode);
    navigate('/', { replace: true });
  }, [mode, navigate, openAuthModal]);

  return null;
}
