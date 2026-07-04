// components/layout/ProtectedRoute.tsx

"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/contexts/AuthContext";
import { ROUTES } from "@/constants/routes";

interface ProtectedRouteProps {
  children: React.ReactNode;
  // Optional: restrict to a specific role
  requiredRole?: "customer" | "staff" | "admin";
  // Where to redirect if not authenticated (defaults to /auth)
  redirectTo?: string;
}

export function ProtectedRoute({
  children,
  requiredRole,
  redirectTo = ROUTES.AUTH,
}: ProtectedRouteProps) {
  const { user, isLoading, isAuthenticated } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (isLoading) return; // wait until session check completes

    if (!isAuthenticated) {
      router.replace(redirectTo);
      return;
    }

    // Role check — admin can access staff routes, staff cannot access admin routes
    if (requiredRole) {
      const roleHierarchy = { customer: 0, staff: 1, admin: 2 };
      const userLevel = roleHierarchy[user!.role];
      const requiredLevel = roleHierarchy[requiredRole];

      if (userLevel < requiredLevel) {
        // Authenticated but wrong role — send home
        router.replace(ROUTES.HOME);
      }
    }
  }, [isLoading, isAuthenticated, user, requiredRole, redirectTo, router]);

  // While checking session — show a minimal spinner
  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="w-8 h-8 border-2 border-current border-t-transparent rounded-full animate-spin opacity-40" />
      </div>
    );
  }

  // Not authenticated — render nothing while redirect happens
  if (!isAuthenticated) return null;

  // Role mismatch — render nothing while redirect happens
  if (requiredRole) {
    const roleHierarchy = { customer: 0, staff: 1, admin: 2 };
    if (roleHierarchy[user!.role] < roleHierarchy[requiredRole]) return null;
  }

  return <>{children}</>;
}