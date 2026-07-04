// src/components/ui/Button.tsx

"use client";

import { forwardRef, ButtonHTMLAttributes } from "react";
import { cn } from "@/utils";

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "primary" | "secondary" | "ghost" | "danger" | "outline";
  size?: "sm" | "md" | "lg";
  isLoading?: boolean;
  fullWidth?: boolean;
}

const variantClasses: Record<NonNullable<ButtonProps["variant"]>, string> = {
  primary:
    "bg-amber-700 hover:bg-amber-800 text-white shadow-sm active:scale-[0.98]",
  secondary:
    "bg-stone-100 hover:bg-stone-200 text-stone-800 dark:bg-stone-800 dark:hover:bg-stone-700 dark:text-stone-100",
  ghost:
    "bg-transparent hover:bg-stone-100 text-stone-700 dark:hover:bg-stone-800 dark:text-stone-300",
  danger:
    "bg-red-600 hover:bg-red-700 text-white shadow-sm active:scale-[0.98]",
  outline:
    "border border-stone-300 dark:border-stone-600 bg-transparent hover:bg-stone-50 dark:hover:bg-stone-800 text-stone-700 dark:text-stone-300",
};

const sizeClasses: Record<NonNullable<ButtonProps["size"]>, string> = {
  sm: "h-8 px-3 text-sm rounded-lg gap-1.5",
  md: "h-10 px-4 text-sm rounded-xl gap-2",
  lg: "h-12 px-6 text-base rounded-xl gap-2",
};

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      variant = "primary",
      size = "md",
      isLoading = false,
      fullWidth = false,
      className,
      children,
      disabled,
      ...props
    },
    ref
  ) => {
    return (
      <button
        ref={ref}
        disabled={disabled || isLoading}
        className={cn(
          // Base
          "inline-flex items-center justify-center font-medium",
          "transition-all duration-150 ease-in-out",
          "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500 focus-visible:ring-offset-2",
          "disabled:opacity-50 disabled:cursor-not-allowed disabled:active:scale-100",
          // Variant + size
          variantClasses[variant],
          sizeClasses[size],
          // Optional
          fullWidth && "w-full",
          className
        )}
        {...props}
      >
        {isLoading ? (
          <>
            <span className="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
            <span>Loading…</span>
          </>
        ) : (
          children
        )}
      </button>
    );
  }
);

Button.displayName = "Button";