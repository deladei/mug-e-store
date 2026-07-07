// src/components/cart/CartDrawer.tsx
//
// The cart as a right-side slide-in drawer. Opening it overlays the storefront
// (a dimmed backdrop) instead of navigating away, so the menu stays put behind
// it. Driven by CartContext.isOpen / openCart / closeCart.

"use client";

import { useEffect, useRef } from "react";
import { useRouter } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import { X, ShoppingBag, AlertTriangle } from "lucide-react";
import { useCart } from "@/contexts/CartContext";
import { useAuth } from "@/contexts/AuthContext";
import { Button } from "@/components/ui/Button";
import { CartLineRow } from "./CartLineRow";
import { formatMoney } from "@/utils";
import { ROUTES } from "@/constants/routes";
import { toast } from "@/hooks/useToast";

export function CartDrawer() {
  const router = useRouter();
  const { isAuthenticated } = useAuth();
  const {
    cart,
    isLoading,
    isOpen,
    closeCart,
    fetchCart,
    updateLine,
    removeLine,
  } = useCart();

  // Refresh the cart each time the drawer opens so it reflects server truth.
  useEffect(() => {
    if (isOpen && isAuthenticated) fetchCart();
  }, [isOpen, isAuthenticated, fetchCart]);

  // While open: close on Escape, trap Tab inside the panel, lock body scroll,
  // and move focus into the drawer — restoring it to the opener on close.
  const panelRef = useRef<HTMLElement>(null);
  useEffect(() => {
    if (!isOpen) return;

    const opener =
      document.activeElement instanceof HTMLElement
        ? document.activeElement
        : null;
    panelRef.current?.focus();

    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        closeCart();
        return;
      }
      if (e.key !== "Tab") return;
      const panel = panelRef.current;
      if (!panel) return;
      const focusables = panel.querySelectorAll<HTMLElement>(
        'button:not([disabled]), [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
      );
      if (focusables.length === 0) return;
      const first = focusables[0];
      const last = focusables[focusables.length - 1];
      const active = document.activeElement;
      if (!panel.contains(active)) {
        e.preventDefault();
        first.focus();
      } else if (e.shiftKey && active === first) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && active === last) {
        e.preventDefault();
        first.focus();
      }
    };
    document.addEventListener("keydown", onKey);
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.removeEventListener("keydown", onKey);
      document.body.style.overflow = prevOverflow;
      opener?.focus();
    };
  }, [isOpen, closeCart]);

  const handleUpdate = async (lineId: string, qty: number) => {
    try {
      await updateLine(lineId, qty);
    } catch {
      toast.error("Could not update quantity");
    }
  };

  const handleRemove = async (lineId: string) => {
    try {
      await removeLine(lineId);
      toast.success("Item removed");
    } catch {
      toast.error("Could not remove item");
    }
  };

  const goToCheckout = () => {
    closeCart();
    router.push(ROUTES.CHECKOUT);
  };

  const lines = cart?.lines ?? [];
  const hasUnavailableLines = lines.some((l) => !l.available);
  const canCheckout = lines.length > 0 && !hasUnavailableLines;

  return (
    <AnimatePresence>
      {isOpen && (
        <div className="fixed inset-0 z-50">
          {/* Backdrop */}
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2 }}
            onClick={closeCart}
            className="absolute inset-0 bg-stone-900/40 backdrop-blur-sm"
            aria-hidden
          />

          {/* Panel */}
          <motion.aside
            ref={panelRef}
            tabIndex={-1}
            role="dialog"
            aria-modal="true"
            aria-label="Your cart"
            initial={{ x: "100%" }}
            animate={{ x: 0 }}
            exit={{ x: "100%" }}
            transition={{ type: "tween", duration: 0.28, ease: [0.32, 0.72, 0, 1] }}
            className="absolute top-0 right-0 h-full w-full max-w-md flex flex-col bg-white dark:bg-stone-950 shadow-2xl"
          >
            {/* Header */}
            <div className="flex items-center justify-between px-5 h-14 border-b border-stone-200 dark:border-stone-800 shrink-0">
              <h2 className="font-bold text-stone-900 dark:text-stone-100">
                Your Cart
              </h2>
              <button
                onClick={closeCart}
                aria-label="Close cart"
                className="w-9 h-9 flex items-center justify-center rounded-xl text-stone-500 hover:text-stone-800 dark:hover:text-stone-200 hover:bg-stone-100 dark:hover:bg-stone-800 transition-colors"
              >
                <X size={18} />
              </button>
            </div>

            {/* Body */}
            <div className="flex-1 overflow-y-auto px-5">
              {lines.length === 0 ? (
                <div className="flex flex-col items-center justify-center h-full text-center space-y-4 py-16">
                  <div className="w-16 h-16 rounded-full bg-stone-100 dark:bg-stone-800 flex items-center justify-center">
                    <ShoppingBag size={26} className="text-stone-400" />
                  </div>
                  <div>
                    <p className="font-semibold text-stone-800 dark:text-stone-200">
                      Your cart is empty
                    </p>
                    <p className="text-sm text-stone-500 dark:text-stone-400 mt-1">
                      Add something from the menu to get started
                    </p>
                  </div>
                  <Button onClick={closeCart}>Keep browsing</Button>
                </div>
              ) : (
                <>
                  {hasUnavailableLines && (
                    <div className="flex items-start gap-3 px-4 py-3 mt-4 rounded-xl bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-800">
                      <AlertTriangle size={16} className="text-red-500 shrink-0 mt-0.5" />
                      <p className="text-sm text-red-700 dark:text-red-400">
                        Some items are no longer available. Remove them to check out.
                      </p>
                    </div>
                  )}
                  <AnimatePresence mode="popLayout">
                    {lines.map((line) => (
                      <CartLineRow
                        key={line.line_id}
                        line={line}
                        onUpdate={handleUpdate}
                        onRemove={handleRemove}
                        isUpdating={isLoading}
                      />
                    ))}
                  </AnimatePresence>
                </>
              )}
            </div>

            {/* Footer */}
            {lines.length > 0 && (
              <div className="border-t border-stone-200 dark:border-stone-800 px-5 py-4 space-y-3 shrink-0">
                <div className="flex justify-between items-center">
                  <span className="text-sm text-stone-500 dark:text-stone-400">
                    Subtotal
                  </span>
                  <span className="font-bold text-lg text-stone-900 dark:text-stone-100">
                    {formatMoney(cart?.subtotal_pesewas ?? 0)}
                  </span>
                </div>
                <Button
                  fullWidth
                  size="lg"
                  disabled={!canCheckout}
                  onClick={goToCheckout}
                >
                  Checkout
                </Button>
                <Button fullWidth variant="ghost" onClick={closeCart}>
                  Continue Shopping
                </Button>
              </div>
            )}
          </motion.aside>
        </div>
      )}
    </AnimatePresence>
  );
}
