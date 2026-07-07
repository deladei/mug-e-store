// src/components/cart/CartLineRow.tsx
//
// One cart line with a quantity stepper and remove button. Shared by the cart
// drawer and the full cart page so both render lines identically.

"use client";

import { motion } from "framer-motion";
import { Minus, Plus, Trash2, AlertTriangle } from "lucide-react";
import { CartLine } from "@/types";
import { formatMoney, cn } from "@/utils";

export function CartLineRow({
  line,
  onUpdate,
  onRemove,
  isUpdating,
}: {
  line: CartLine;
  onUpdate: (lineId: string, qty: number) => void;
  onRemove: (lineId: string) => void;
  isUpdating: boolean;
}) {
  return (
    <motion.div
      layout
      initial={{ opacity: 0, x: -16 }}
      animate={{ opacity: 1, x: 0 }}
      exit={{ opacity: 0, x: 16, height: 0, marginBottom: 0 }}
      transition={{ duration: 0.2 }}
      className={cn(
        "flex flex-col gap-3 py-4",
        "border-b border-stone-100 dark:border-stone-800 last:border-0",
        !line.available && "opacity-60"
      )}
    >
      {/* Unavailable warning */}
      {!line.available && (
        <div className="flex items-center gap-2 px-3 py-2 rounded-xl bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-800">
          <AlertTriangle size={14} className="text-red-500 shrink-0" />
          <p className="text-xs text-red-700 dark:text-red-400">
            This item is no longer available. Remove it to continue.
          </p>
        </div>
      )}

      <div className="flex items-start justify-between gap-3">
        {/* Item info */}
        <div className="flex-1 min-w-0">
          <p className="font-medium text-stone-900 dark:text-stone-100 text-sm leading-snug">
            {line.item_name}
          </p>
          <p className="text-xs text-stone-500 dark:text-stone-400 mt-0.5">
            {line.variant_name}
          </p>
          <p className="text-sm font-semibold text-amber-700 dark:text-amber-500 mt-1">
            {formatMoney(line.unit_price_pesewas * line.quantity)}
          </p>
        </div>

        {/* Quantity stepper + remove */}
        <div className="flex items-center gap-2">
          <div className="flex items-center gap-1 border border-stone-200 dark:border-stone-700 rounded-xl overflow-hidden">
            <button
              onClick={() =>
                line.quantity > 1
                  ? onUpdate(line.line_id, line.quantity - 1)
                  : onRemove(line.line_id)
              }
              disabled={isUpdating}
              aria-label="Decrease quantity"
              className={cn(
                "w-8 h-8 flex items-center justify-center",
                "text-stone-500 hover:text-stone-800 dark:hover:text-stone-200",
                "hover:bg-stone-100 dark:hover:bg-stone-800",
                "transition-colors disabled:opacity-40"
              )}
            >
              <Minus size={13} />
            </button>

            <span className="w-6 text-center text-sm font-semibold text-stone-900 dark:text-stone-100">
              {line.quantity}
            </span>

            <button
              onClick={() => onUpdate(line.line_id, line.quantity + 1)}
              disabled={isUpdating || line.quantity >= 20}
              aria-label="Increase quantity"
              className={cn(
                "w-8 h-8 flex items-center justify-center",
                "text-stone-500 hover:text-stone-800 dark:hover:text-stone-200",
                "hover:bg-stone-100 dark:hover:bg-stone-800",
                "transition-colors disabled:opacity-40"
              )}
            >
              <Plus size={13} />
            </button>
          </div>

          {/* Remove button */}
          <button
            onClick={() => onRemove(line.line_id)}
            disabled={isUpdating}
            aria-label="Remove item"
            className={cn(
              "w-8 h-8 flex items-center justify-center rounded-xl",
              "text-stone-400 hover:text-red-500",
              "hover:bg-red-50 dark:hover:bg-red-950/30",
              "transition-colors disabled:opacity-40"
            )}
          >
            <Trash2 size={14} />
          </button>
        </div>
      </div>
    </motion.div>
  );
}
