// src/hooks/useToast.ts

"use client";

import { useState, useCallback } from "react";

export type ToastType = "success" | "error" | "info" | "warning";

export interface Toast {
  id: string;
  message: string;
  type: ToastType;
}

// Module-level singleton so any component can trigger toasts
// without prop drilling
type ToastListener = (toast: Toast) => void;
let listener: ToastListener | null = null;

export const toast = {
  show: (message: string, type: ToastType = "info") => {
    listener?.({
      id: Math.random().toString(36).slice(2),
      message,
      type,
    });
  },
  success: (message: string) => toast.show(message, "success"),
  error: (message: string) => toast.show(message, "error"),
  warning: (message: string) => toast.show(message, "warning"),
  info: (message: string) => toast.show(message, "info"),
};

// Used by the ToastContainer component to register itself as the listener
export function useToastListener() {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const register = useCallback(() => {
    listener = (t: Toast) => {
      setToasts((prev) => [...prev, t]);
      setTimeout(() => {
        setToasts((prev) => prev.filter((x) => x.id !== t.id));
      }, 3500);
    };
    return () => {
      listener = null;
    };
  }, []);

  return { toasts, register };
}