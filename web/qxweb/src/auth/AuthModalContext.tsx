import {
  createContext,
  lazy,
  Suspense,
  useCallback,
  useContext,
  useState,
  type ReactNode,
} from 'react';
import { useLocation } from 'react-router-dom';
import { buildAuthReturnPath } from '@/auth/authReturn';

const AuthModal = lazy(() =>
  import('@/components/AuthModal').then((module) => ({ default: module.AuthModal })),
);

export type AuthMode = 'login' | 'register';

type AuthModalContextValue = {
  openAuthModal: (mode?: AuthMode, returnTo?: string) => void;
  closeAuthModal: () => void;
};

const AuthModalContext = createContext<AuthModalContextValue | null>(null);

export function AuthModalProvider({ children }: { children: ReactNode }) {
  const location = useLocation();
  const [open, setOpen] = useState(false);
  const [mode, setMode] = useState<AuthMode>('login');
  const [returnTo, setReturnTo] = useState('/');

  const openAuthModal = useCallback(
    (nextMode: AuthMode = 'login', explicitReturnTo?: string) => {
      const path =
        explicitReturnTo ??
        buildAuthReturnPath(location.pathname, location.search, location.hash);
      setReturnTo(path);
      setMode(nextMode);
      setOpen(true);
    },
    [location.pathname, location.search, location.hash],
  );

  const closeAuthModal = useCallback(() => {
    setOpen(false);
  }, []);

  return (
    <AuthModalContext.Provider value={{ openAuthModal, closeAuthModal }}>
      {children}
      {open ? (
        <Suspense fallback={null}>
          <AuthModal
            open={open}
            mode={mode}
            returnTo={returnTo}
            onModeChange={setMode}
            onClose={closeAuthModal}
          />
        </Suspense>
      ) : null}
    </AuthModalContext.Provider>
  );
}

export function useAuthModal() {
  const context = useContext(AuthModalContext);
  if (!context) {
    throw new Error('useAuthModal must be used within AuthModalProvider');
  }
  return context;
}
