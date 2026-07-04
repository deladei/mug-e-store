// src/components/ui/Toggle.tsx

"use client";

import { cn } from "@/utils";

interface ToggleProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label?: string;
  description?: string;
  disabled?: boolean;
  size?: "sm" | "md";
}

export function Toggle({
  checked,
  onChange,
  label,
  description,
  disabled = false,
  size = "md",
}: ToggleProps) {
  const trackSize =
    size === "sm" ? "w-8 h-4" : "w-11 h-6";
  const thumbSize =
    size === "sm" ? "w-3 h-3" : "w-4 h-4";
  const thumbTranslate =
    size === "sm"
      ? checked ? "translate-x-4" : "translate-x-0.5"
      : checked ? "translate-x-6" : "translate-x-1";

  return (
    <label
      className={cn(
        "flex items-center gap-3",
        disabled ? "cursor-not-allowed opacity-50" : "cursor-pointer"
      )}
    >
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        disabled={disabled}
        onClick={() => onChange(!checked)}
        className={cn(
          "relative inline-flex shrink-0 items-center rounded-full",
          "transition-colors duration-200 ease-in-out",
          "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500 focus-visible:ring-offset-2",
          trackSize,
          checked
            ? "bg-amber-700"
            : "bg-stone-300 dark:bg-stone-600"
        )}
      >
        <span
          className={cn(
            "inline-block rounded-full bg-white shadow-sm",
            "transition-transform duration-200 ease-in-out",
            thumbSize,
            thumbTranslate
          )}
        />
      </button>

      {(label || description) && (
        <div className="flex flex-col">
          {label && (
            <span className="text-sm font-medium text-stone-800 dark:text-stone-200">
              {label}
            </span>
          )}
          {description && (
            <span className="text-xs text-stone-500 dark:text-stone-400">
              {description}
            </span>
          )}
        </div>
      )}
    </label>
  );
}