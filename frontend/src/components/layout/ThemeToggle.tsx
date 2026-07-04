// src/components/layout/ThemeToggle.tsx

"use client";

import { useTheme } from "next-themes";
import { Sun, Moon } from "lucide-react";
import { cn } from "@/utils";

export function ThemeToggle({ className }: { className?: string }) {
  const { theme, setTheme } = useTheme();

  return (
    <button
      onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
      aria-label="Toggle theme"
      className={cn(
        "w-9 h-9 flex items-center justify-center rounded-xl",
        "text-stone-500 hover:text-stone-800 dark:hover:text-stone-200",
        "hover:bg-stone-100 dark:hover:bg-stone-800",
        "transition-colors duration-150",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500",
        className
      )}
    >
      <Sun size={18} className="hidden dark:block" />
      <Moon size={18} className="block dark:hidden" />
    </button>
  );
}