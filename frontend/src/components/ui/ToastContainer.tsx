// src/components/ui/ToastContainer.tsx

"use client";

import { useEffect } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { CheckCircle, XCircle, AlertCircle, Info, X } from "lucide-react";
import { useToastListener, Toast, ToastType } from "@/hooks/useToast";
import { cn } from "@/utils";

const icons: Record<ToastType, React.ReactNode> = {
  success: <CheckCircle size={16} className="text-emerald-500 shrink-0" />,
  error: <XCircle size={16} className="text-red-500 shrink-0" />,
  warning: <AlertCircle size={16} className="text-yellow-500 shrink-0" />,
  info: <Info size={16} className="text-blue-500 shrink-0" />,
};

const borderColors: Record<ToastType, string> = {
  success: "border-l-emerald-500",
  error: "border-l-red-500",
  warning: "border-l-yellow-500",
  info: "border-l-blue-500",
};

function ToastItem({ toast }: { toast: Toast }) {
  return (
    <motion.div
      layout
      initial={{ opacity: 0, y: 32, scale: 0.95 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      exit={{ opacity: 0, y: 8, scale: 0.95 }}
      transition={{ type: "spring", stiffness: 400, damping: 30 }}
      className={cn(
        "flex items-start gap-3 px-4 py-3 rounded-xl shadow-lg",
        "bg-white dark:bg-stone-900",
        "border border-stone-200 dark:border-stone-700",
        "border-l-4",
        borderColors[toast.type],
        "max-w-sm w-full"
      )}
    >
      {icons[toast.type]}
      <p className="text-sm text-stone-800 dark:text-stone-200 flex-1 leading-snug">
        {toast.message}
      </p>
    </motion.div>
  );
}

export function ToastContainer() {
  const { toasts, register } = useToastListener();

  useEffect(() => {
    const unregister = register();
    return unregister;
  }, [register]);

  return (
    <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 flex flex-col gap-2 items-center pointer-events-none">
      <AnimatePresence mode="popLayout">
        {toasts.map((t) => (
          <ToastItem key={t.id} toast={t} />
        ))}
      </AnimatePresence>
    </div>
  );
}