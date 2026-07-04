// src/components/layout/CustomerNav.tsx

"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { User, History } from "lucide-react";
import { useAuth } from "@/contexts/AuthContext";
import { ThemeToggle } from "./ThemeToggle";
import { CartButton } from "./CartButton";
import { cn } from "@/utils";
import { ROUTES } from "@/constants/routes";

export function CustomerNav() {
  const { isAuthenticated } = useAuth();
  const pathname = usePathname();

  const isActive = (path: string) => pathname === path;

  return (
    <header
      className={cn(
        "sticky top-0 z-40",
        "bg-white/80 dark:bg-stone-950/80",
        "backdrop-blur-md",
        "border-b border-stone-200/60 dark:border-stone-800/60"
      )}
    >
      <nav className="max-w-2xl mx-auto px-4 h-14 flex items-center justify-between">
        {/* Logo */}
        <Link
          href={ROUTES.HOME}
          className="flex items-center gap-2 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500 rounded-lg"
        >
          <span className="text-xl">☕</span>
          <span className="font-bold text-stone-900 dark:text-stone-100 tracking-tight">
            Coffee Mug
          </span>
        </Link>

        {/* Right side actions */}
        <div className="flex items-center gap-1">
          {/* Order history — only when authenticated */}
          {isAuthenticated && (
            <Link
              href={ROUTES.ORDER_HISTORY}
              aria-label="Order history"
              className={cn(
                "w-10 h-10 flex items-center justify-center rounded-xl",
                "transition-colors duration-150",
                "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500",
                isActive(ROUTES.ORDER_HISTORY)
                  ? "text-amber-700 bg-amber-50 dark:bg-amber-950/30"
                  : "text-stone-500 hover:text-stone-800 dark:hover:text-stone-200 hover:bg-stone-100 dark:hover:bg-stone-800"
              )}
            >
              <History size={18} />
            </Link>
          )}

          {/* Profile or login */}
          <Link
            href={isAuthenticated ? ROUTES.PROFILE : ROUTES.AUTH}
            aria-label={isAuthenticated ? "Profile" : "Sign in"}
            className={cn(
              "w-10 h-10 flex items-center justify-center rounded-xl",
              "transition-colors duration-150",
              "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500",
              isActive(ROUTES.PROFILE) || isActive(ROUTES.AUTH)
                ? "text-amber-700 bg-amber-50 dark:bg-amber-950/30"
                : "text-stone-500 hover:text-stone-800 dark:hover:text-stone-200 hover:bg-stone-100 dark:hover:bg-stone-800"
            )}
          >
            <User size={18} />
          </Link>

          <ThemeToggle />
          <CartButton />
        </div>
      </nav>
    </header>
  );
}