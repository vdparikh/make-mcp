import { createContext, useContext, useMemo, useState, type ReactNode } from 'react';

export interface TryTarget {
  type?: string;
  id?: string;
  name?: string;
  /** Hosted MCP endpoint URL (shown immediately in Try Chat sidebar). */
  endpoint?: string;
  /** Tool names when known (e.g. passed from caller); otherwise fetched from server. */
  toolNames?: string[];
}

interface TryChatContextValue {
  open: boolean;
  target: TryTarget | null;
  openTryChat: (target?: TryTarget) => void;
  closeTryChat: () => void;
}

const TryChatContext = createContext<TryChatContextValue | undefined>(undefined);

export function TryChatProvider({ children }: { children: ReactNode }) {
  const [open, setOpen] = useState(false);
  const [target, setTarget] = useState<TryTarget | null>(null);

  const value = useMemo<TryChatContextValue>(
    () => ({
      open,
      target,
      openTryChat: (nextTarget?: TryTarget) => {
        setTarget(nextTarget || null);
        setOpen(true);
      },
      closeTryChat: () => {
        setOpen(false);
        setTarget(null);
      },
    }),
    [open, target]
  );

  return <TryChatContext.Provider value={value}>{children}</TryChatContext.Provider>;
}

export function useTryChat() {
  const ctx = useContext(TryChatContext);
  if (!ctx) {
    throw new Error('useTryChat must be used within TryChatProvider');
  }
  return ctx;
}

