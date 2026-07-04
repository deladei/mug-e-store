// src/components/layout/StaffNav.tsx

"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { LogOut, UtensilsCrossed, ClipboardList, BarChart2 } from "lucide-react";
import { useAuth } from "@/contexts/AuthContext";
import { ThemeToggle } from "./ThemeToggle";
import { ROUTES } from "@/constants/routes";
import { cn } from "@/utils";

export function StaffNav() {
  const { user, logout } = useAuth();
  const pathname = usePathname();
  const router = useRouter();

  const isActive = (path: string) => pathname.startsWith(path);

  const navLinks = [
  {
    href: ROUTES.STAFF_QUEUE,
    label: "Orders",
    icon: <ClipboardList size={16} />,
  },
  ...(user?.role === "admin"
    ? [
        {
          href: ROUTES.ADMIN_MENU,
          label: "Menu",
          icon: <UtensilsCrossed size={16} />,
        },
        {
          href: ROUTES.ADMIN_REPORTS,
          label: "Reports",
          icon: <BarChart2 size={16} />,
        },
      ]
    : []),
];

  return (
    <header className="sticky top-0 z-40 bg-white dark:bg-stone-900 border-b border-stone-200 dark:border-stone-800 shadow-sm">
      <nav className="max-w-7xl mx-auto px-6 h-14 flex items-center justify-between">
        {/* Logo + nav links */}
        <div className="flex items-center gap-6">
          <Link
            href={ROUTES.STAFF_QUEUE}
            className="flex items-center gap-2 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500 rounded-lg"
          >
            <span className="text-lg">☕</span>
            <span className="font-bold text-stone-900 dark:text-stone-100 tracking-tight text-sm">
              Coffee Mug
            </span>
            <span className="text-xs px-1.5 py-0.5 rounded bg-amber-100 dark:bg-amber-900/40 text-amber-700 dark:text-amber-400 font-medium">
              {user?.role === "admin" ? "Admin" : "Staff"}
            </span>
          </Link>

          <div className="flex items-center gap-1">
            {navLinks.map((link) => (
              <Link
                key={link.href}
                href={link.href}
                className={cn(
                  "flex items-center gap-1.5 px-3 h-8 rounded-lg text-sm font-medium",
                  "transition-colors duration-150",
                  "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500",
                  isActive(link.href)
                    ? "bg-amber-50 dark:bg-amber-950/30 text-amber-700 dark:text-amber-400"
                    : "text-stone-600 dark:text-stone-400 hover:text-stone-900 dark:hover:text-stone-100 hover:bg-stone-100 dark:hover:bg-stone-800"
                )}
              >
                {link.icon}
                {link.label}
              </Link>
            ))}
          </div>
        </div>

        {/* Right side */}
        <div className="flex items-center gap-3">
          {user && (
            <span className="text-sm text-stone-500 dark:text-stone-400 hidden sm:block">
              {user.name}
            </span>
          )}
          <ThemeToggle />
          <button
            onClick={async () => {
              await logout();
              router.push(ROUTES.STAFF_QUEUE);
            }}
            className="flex items-center gap-1.5 px-3 h-8 rounded-lg text-sm font-medium text-stone-500 hover:text-red-600 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-950/30 transition-colors"
          >
            <LogOut size={15} />
            <span className="hidden sm:block">Sign out</span>
          </button>
        </div>
      </nav>
    </header>
  );
}