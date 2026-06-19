import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type PropsWithChildren,
} from "react";

import { getCurrentUser, login as loginRequest } from "../api/auth";
import { ApiError, setUnauthorizedCallback } from "../api/client";
import type { AuthUser } from "../types/auth";
import { storage } from "../utils/storage";

interface LoginCredentials {
  email: string;
  password: string;
}

interface AuthContextValue {
  user: AuthUser | null;
  isAuthenticated: boolean;
  isInitializing: boolean;
  sessionError: string | null;
  login: (credentials: LoginCredentials) => Promise<void>;
  logout: () => void;
  refreshSession: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

function getSessionErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    return error.message;
  }

  if (error instanceof Error && error.message) {
    return error.message;
  }

  return "Failed to verify the current session.";
}

export function AuthProvider({ children }: PropsWithChildren): JSX.Element {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [isInitializing, setIsInitializing] = useState(true);
  const [sessionError, setSessionError] = useState<string | null>(null);

  const clearSession = useCallback((): void => {
    storage.clearAccessToken();
    setUser(null);
    setSessionError(null);
  }, []);

  const refreshSession = useCallback(async (): Promise<void> => {
    const token = storage.getAccessToken();

    if (!token) {
      setUser(null);
      setSessionError(null);
      setIsInitializing(false);
      return;
    }

    setIsInitializing(true);
    setSessionError(null);

    try {
      const data = await getCurrentUser();
      setUser(data.user);
    } catch (error) {
      if (error instanceof ApiError && (error.status === 401 || error.status === 403)) {
        clearSession();
      } else {
        setUser(null);
        setSessionError(getSessionErrorMessage(error));
      }
    } finally {
      setIsInitializing(false);
    }
  }, [clearSession]);

  const login = useCallback(async (credentials: LoginCredentials): Promise<void> => {
    const data = await loginRequest(credentials);
    storage.setAccessToken(data.access_token);
    setUser(data.user);
    setSessionError(null);
    setIsInitializing(false);
  }, []);

  const logout = useCallback((): void => {
    clearSession();
    setIsInitializing(false);
  }, [clearSession]);

  useEffect(() => {
    setUnauthorizedCallback(clearSession);
    return () => {
      setUnauthorizedCallback(null);
    };
  }, [clearSession]);

  useEffect(() => {
    void refreshSession();
  }, [refreshSession]);

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      isAuthenticated: user !== null,
      isInitializing,
      sessionError,
      login,
      logout,
      refreshSession,
    }),
    [user, isInitializing, sessionError, login, logout, refreshSession],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);

  if (!context) {
    throw new Error("useAuth must be used within AuthProvider");
  }

  return context;
}
