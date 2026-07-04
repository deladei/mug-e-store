// src/components/ui/Input.tsx

"use client";

import { forwardRef, InputHTMLAttributes } from "react";
import { cn } from "@/utils";

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  hint?: string;
  leftIcon?: React.ReactNode;
  rightIcon?: React.ReactNode;
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ label, error, hint, leftIcon, rightIcon, className, id, ...props }, ref) => {
    const inputId = id ?? label?.toLowerCase().replace(/\s+/g, "-");

    return (
      <div className="flex flex-col gap-1.5">
        {label && (
          <label
            htmlFor={inputId}
            className="text-sm font-medium text-stone-700 dark:text-stone-300"
          >
            {label}
          </label>
        )}

        <div className="relative flex items-center">
          {leftIcon && (
            <span className="absolute left-3 text-stone-400 dark:text-stone-500 pointer-events-none">
              {leftIcon}
            </span>
          )}

          <input
            ref={ref}
            id={inputId}
            className={cn(
              // Base
              "w-full h-11 rounded-xl border bg-white dark:bg-stone-900",
              "text-stone-900 dark:text-stone-100 placeholder:text-stone-400",
              "text-sm transition-colors duration-150",
              // Border
              error
                ? "border-red-400 dark:border-red-600 focus:ring-red-400"
                : "border-stone-300 dark:border-stone-700 focus:ring-amber-500",
              // Focus
              "focus:outline-none focus:ring-2 focus:ring-offset-0 focus:border-transparent",
              // Padding — adjust for icons
              leftIcon ? "pl-10" : "pl-4",
              rightIcon ? "pr-10" : "pr-4",
              // Disabled
              "disabled:opacity-50 disabled:cursor-not-allowed",
              className
            )}
            {...props}
          />

          {rightIcon && (
            <span className="absolute right-3 text-stone-400 dark:text-stone-500">
              {rightIcon}
            </span>
          )}
        </div>

        {error && (
          <p className="text-xs text-red-600 dark:text-red-400">{error}</p>
        )}
        {hint && !error && (
          <p className="text-xs text-stone-500 dark:text-stone-400">{hint}</p>
        )}
      </div>
    );
  }
);

Input.displayName = "Input";