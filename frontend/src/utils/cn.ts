// src/utils/cn.ts

import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

// cn() combines clsx (conditional classes) with tailwind-merge (conflict resolution).
//
// Without twMerge, doing cn("px-4", "px-8") would output "px-4 px-8" and the
// browser would apply whichever appears last in the stylesheet — unpredictable.
// With twMerge, it outputs "px-8" — the last one wins, as you'd expect.
//
// Usage:
//   cn("base-class", isActive && "active-class", "always-applied")
//   cn("px-4 py-2", className)  ← safely merge external className props

export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}