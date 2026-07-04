// src/components/ui/Badge.tsx

import { cn } from "@/utils";
import { OrderStatus } from "@/types";
import { getStatusColors, getStatusLabel } from "@/utils";

// ── Generic badge ──────────────────────────────────────────────────────────────

interface BadgeProps {
  children: React.ReactNode;
  variant?: "default" | "success" | "warning" | "danger" | "info";
  className?: string;
}

const badgeVariants: Record<NonNullable<BadgeProps["variant"]>, string> = {
  default:
    "bg-stone-100 text-stone-700 border-stone-200 dark:bg-stone-800 dark:text-stone-300 dark:border-stone-700",
  success:
    "bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-950/30 dark:text-emerald-400 dark:border-emerald-800",
  warning:
    "bg-yellow-50 text-yellow-700 border-yellow-200 dark:bg-yellow-950/30 dark:text-yellow-400 dark:border-yellow-800",
  danger:
    "bg-red-50 text-red-700 border-red-200 dark:bg-red-950/30 dark:text-red-400 dark:border-red-800",
  info: "bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-950/30 dark:text-blue-400 dark:border-blue-800",
};

export function Badge({
  children,
  variant = "default",
  className,
}: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium border",
        badgeVariants[variant],
        className
      )}
    >
      {children}
    </span>
  );
}

// ── Status badge — maps an OrderStatus directly to colour + label ──────────────

interface StatusBadgeProps {
  status: OrderStatus;
  className?: string;
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const { bg, text, border } = getStatusColors(status);
  return (
    <span
      className={cn(
        "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium border",
        bg,
        text,
        border,
        className
      )}
    >
      {getStatusLabel(status)}
    </span>
  );
}