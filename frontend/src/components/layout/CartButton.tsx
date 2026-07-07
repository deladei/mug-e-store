// src/components/layout/CartButton.tsx

"use client";

import { useEffect } from "react";
import { ShoppingBag } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { useCart } from "@/contexts/CartContext";
import { useAuth } from "@/contexts/AuthContext";
import { cn } from "@/utils";

export function CartButton() {
  const { totalItems, fetchCart, openCart } = useCart();
  const { isAuthenticated } = useAuth();

  // Fetch cart on mount when authenticated
  useEffect(() => {
    if (isAuthenticated) {
      fetchCart();
    }
  }, [isAuthenticated, fetchCart]);

  return (
    <button
      onClick={openCart}
      aria-label={`Cart — ${totalItems} item${totalItems !== 1 ? "s" : ""}`}
      className={cn(
        "relative flex items-center justify-center",
        "w-10 h-10 rounded-xl",
        "text-stone-600 dark:text-stone-400",
        "hover:bg-stone-100 dark:hover:bg-stone-800",
        "hover:text-stone-900 dark:hover:text-stone-100",
        "transition-colors duration-150",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500"
      )}
    >
      <ShoppingBag size={20} />

      {/* Item count badge */}
      <AnimatePresence>
        {totalItems > 0 && (
          <motion.span
            key="cart-count"
            initial={{ scale: 0, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            exit={{ scale: 0, opacity: 0 }}
            transition={{ type: "spring", stiffness: 500, damping: 30 }}
            className={cn(
              "absolute -top-1 -right-1",
              "min-w-[18px] h-[18px] px-1",
              "bg-amber-700 text-white",
              "text-[10px] font-bold rounded-full",
              "flex items-center justify-center",
              "pointer-events-none"
            )}
          >
            {totalItems > 99 ? "99+" : totalItems}
          </motion.span>
        )}
      </AnimatePresence>
    </button>
  );
}