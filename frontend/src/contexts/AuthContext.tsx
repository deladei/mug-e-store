// contexts/AuthContext.tsx

"use client";

import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  ReactNode,
} from "react";
import { useRouter } from "next/navigation";
import { User, LoginPayload, RegisterPayload } from "@/types";
import { authService } from "@/services/auth.service";
import { ROUTES } from "@/constants/routes";

// ─── Shape of the context value ───────────────────────────────────────────────

interface AuthContextValue {
  // State
  user: User | null;
  isLoading: boolean;       // true while we're checking the session on mount
  isAuthenticated: boolean;

  // Actions
  login: (payload: LoginPayload) => Promise<void>;
  register: (payload: RegisterPayload) => Promise<void>;
  logout: () => Promise<void>;
}

// ─── Context ──────────────────────────────────────────────────────────────────

const AuthContext = createContext<AuthContextValue | null>(null);

// ─── Provider ─────────────────────────────────────────────────────────────────

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const router = useRouter();

  // ── Session restore on mount ──────────────────────────────────────────────
  // When the page loads (or refreshes), we don't know if the user is logged in.
  // We call /auth/refresh — if the httpOnly cookie is valid, we get back a
  // fresh access token and the user object. If not, we stay logged out.
  // This happens silently; the user sees a loading state, not a login redirect.
  useEffect(() => {
    const restoreSession = async () => {
      try {
        const data = await authService.refresh();
        setUser(data.user);
      } catch {
        // No valid cookie — user is not logged in. This is not an error.
        setUser(null);
      } finally {
        setIsLoading(false);
      }
    };

    restoreSession();
  }, []);

  // ── Login ─────────────────────────────────────────────────────────────────
  const login = useCallback(async (payload: LoginPayload) => {
    const data = await authService.login(payload);
    setUser(data.user);
  }, []);

  // ── Register ──────────────────────────────────────────────────────────────
  const register = useCallback(async (payload: RegisterPayload) => {
    const data = await authService.register(payload);
    setUser(data.user);
  }, []);

  // ── Logout ────────────────────────────────────────────────────────────────
  const logout = useCallback(async () => {
    try {
      await authService.logout();
    } finally {
      // Always clear local state even if the network call fails
      setUser(null);
      router.push(ROUTES.HOME);
    }
  }, [router]);

  const value: AuthContextValue = {
    user,
    isLoading,
    isAuthenticated: user !== null,
    login,
    register,
    logout,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

// ─── Hook ─────────────────────────────────────────────────────────────────────

// useAuth() is the only way components access the context.
// It throws if used outside <AuthProvider> — catches wiring mistakes at dev time.
export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}