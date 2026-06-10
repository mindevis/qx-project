import {
  createContext,
  useCallback,
  useContext,
  useState,
  type ReactNode,
} from 'react';
import { AuthModal } from '@/components/AuthModal';

export type AuthMode = 'login' | 'register';

type AuthModalContextValue = {
  openAuthModal: (mode?: AuthMode) => void;
  closeAuthModal: () => void;
};

const AuthModalContext = createContext<AuthModalContextValue | null>(null);

export function AuthModalProvider({ children }: { children: ReactNode }) {
  const [open, setOpen] = useState(false);
  const [mode, setMode] = useState<AuthMode>('login');

  const openAuthModal = useCallback((nextMode: AuthMode = 'login') => {
    setMode(nextMode);
    setOpen(true);
  }, []);

  const closeAuthModal = useCallback(() => {
    setOpen(false);
  }, []);

  return (
    <AuthModalContext.Provider value={{ openAuthModal, closeAuthModal }}>
      {children}
      <AuthModal
        open={open}
        mode={mode}
        onModeChange={setMode}
        onClose={closeAuthModal}
      />
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
