import { useEffect } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router-dom';
import { buildAuthReturnPath, isSafeReturnPath } from '@/auth/authReturn';
import { useAuthModal, type AuthMode } from '@/auth/AuthModalContext';

export function AuthRedirect() {
  const { mode } = useParams<{ mode: string }>();
  const location = useLocation();
  const { openAuthModal } = useAuthModal();
  const navigate = useNavigate();

  useEffect(() => {
    const authMode: AuthMode = mode === 'register' ? 'register' : 'login';
    const queryReturnTo = new URLSearchParams(location.search).get('returnTo');
    const returnTo =
      queryReturnTo && isSafeReturnPath(queryReturnTo)
        ? queryReturnTo
        : buildAuthReturnPath('/', '', '');
    openAuthModal(authMode, returnTo);
    navigate(returnTo, { replace: true });
  }, [location.search, mode, navigate, openAuthModal]);

  return null;
}
