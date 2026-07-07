// src/components/layout/RequireAuth.tsx

"use client";

import { useEffect, ReactNode } from "react";
import { usePathname, useRouter } from "next/navigation";
import { useAuth } from "@/contexts/AuthContext";
import { ROUTES } from "@/constants/routes";

// Pages a signed-out visitor is allowed to see
const PUBLIC_PATHS: string[] = [ROUTES.AUTH, ROUTES.RESET_PASSWORD];

// Gates the whole customer app behind login: visitors sign in before they
// see the homepage or any other page, instead of being bounced to /auth
// mid-flow (e.g. at add-to-cart).
export function RequireAuth({ children }: { children: ReactNode }) {
  const { user, isLoading } = useAuth();
  const router = useRouter();
  const pathname = usePathname();
  const isPublic = PUBLIC_PATHS.some((p) => pathname.startsWith(p));

  useEffect(() => {
    if (!isLoading && !user && !isPublic) {
      router.replace(ROUTES.AUTH);
    }
  }, [isLoading, user, isPublic, router]);

  // While restoring the session, or while the redirect is in flight,
  // show a spinner instead of flashing the protected page.
  if (!isPublic && (isLoading || !user)) {
    return (
      <div className="min-h-[50vh] flex items-center justify-center">
        <div className="w-8 h-8 border-2 border-amber-600 border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  return <>{children}</>;
}
