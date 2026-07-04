// src/app/(customer)/cart/page.tsx

"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import { Minus, Plus, Trash2, ShoppingBag, AlertTriangle } from "lucide-react";
import { useCart } from "@/contexts/CartContext";
import { useAuth } from "@/contexts/AuthContext";
import { Button } from "@/components/ui/Button";
import { Card, CardBody, CardFooter } from "@/components/ui/Card";
import { OrderCardSkeleton } from "@/components/ui/Skeleton";
import { formatMoney } from "@/utils";
import { ROUTES } from "@/constants/routes";
import { CartLine } from "@/types";
import { cn } from "@/utils";
import { toast } from "@/hooks/useToast";

// ── Line item component ────────────────────────────────────────────────────────

function CartLineItem({
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

// ── Empty cart state ───────────────────────────────────────────────────────────

function EmptyCart() {
  const router = useRouter();
  return (
    <div className="flex flex-col items-center justify-center py-24 text-center space-y-4">
      <div className="w-20 h-20 rounded-full bg-stone-100 dark:bg-stone-800 flex items-center justify-center">
        <ShoppingBag size={32} className="text-stone-400" />
      </div>
      <div>
        <p className="font-semibold text-stone-800 dark:text-stone-200 text-lg">
          Your cart is empty
        </p>
        <p className="text-sm text-stone-500 dark:text-stone-400 mt-1">
          Add something from the menu to get started
        </p>
      </div>
      <Button onClick={() => router.push(ROUTES.HOME)}>
        Browse Menu
      </Button>
    </div>
  );
}

// ── Main cart page ─────────────────────────────────────────────────────────────

export default function CartPage() {
  const router = useRouter();
  const { isAuthenticated } = useAuth();
  const {
    cart,
    isLoading,
    fetchCart,
    updateLine,
    removeLine,
  } = useCart();

  // Fetch fresh cart on mount
  useEffect(() => {
    if (isAuthenticated) fetchCart();
  }, [isAuthenticated, fetchCart]);

  // Redirect unauthenticated users to auth
  useEffect(() => {
    if (!isAuthenticated) {
      router.replace(ROUTES.AUTH);
    }
  }, [isAuthenticated, router]);

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

  // ── Loading state ────────────────────────────────────────────────────────────
  if (isLoading) {
    return (
      <div className="space-y-4">
        <h1 className="text-2xl font-bold text-stone-900 dark:text-stone-100">
          Your Cart
        </h1>
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <OrderCardSkeleton key={i} />
          ))}
        </div>
      </div>
    );
  }

  // ── Empty state ──────────────────────────────────────────────────────────────
  if (!cart || cart.lines.length === 0) {
    return <EmptyCart />;
  }

  const hasUnavailableLines = cart.lines.some((l) => !l.available);
  const canCheckout = !hasUnavailableLines && cart.lines.length > 0;

  return (
    <div className="space-y-5 pb-8">
      <h1 className="text-2xl font-bold text-stone-900 dark:text-stone-100">
        Your Cart
      </h1>

      {/* Unavailable warning banner */}
      {hasUnavailableLines && (
        <motion.div
          initial={{ opacity: 0, y: -8 }}
          animate={{ opacity: 1, y: 0 }}
          className="flex items-start gap-3 px-4 py-3 rounded-xl bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-800"
        >
          <AlertTriangle
            size={16}
            className="text-red-500 shrink-0 mt-0.5"
          />
          <p className="text-sm text-red-700 dark:text-red-400">
            Some items are no longer available. Remove them to proceed to
            checkout.
          </p>
        </motion.div>
      )}

      {/* Line items */}
      <Card>
        <CardBody className="py-0 px-5">
          <AnimatePresence mode="popLayout">
            {cart.lines.map((line) => (
              <CartLineItem
                key={line.line_id}
                line={line}
                onUpdate={handleUpdate}
                onRemove={handleRemove}
                isUpdating={isLoading}
              />
            ))}
          </AnimatePresence>
        </CardBody>

        <CardFooter>
          <div className="flex justify-between items-center">
            <span className="text-sm text-stone-500 dark:text-stone-400">
              Subtotal
            </span>
            <span className="font-bold text-lg text-stone-900 dark:text-stone-100">
              {formatMoney(cart.subtotal_pesewas)}
            </span>
          </div>
        </CardFooter>
      </Card>

      {/* Checkout CTA */}
      <div className="space-y-3">
        <Button
          fullWidth
          size="lg"
          disabled={!canCheckout}
          onClick={() => router.push(ROUTES.CHECKOUT)}
        >
          Checkout
        </Button>

        <Button
          fullWidth
          variant="ghost"
          onClick={() => router.push(ROUTES.HOME)}
        >
          Continue Shopping
        </Button>
      </div>

      {/* Phase 2 loyalty placeholder */}
      <div className="rounded-xl border border-dashed border-stone-300 dark:border-stone-700 px-4 py-3 flex items-center justify-between">
        <div>
          <p className="text-sm font-medium text-stone-500 dark:text-stone-400">
            Loyalty Points
          </p>
          <p className="text-xs text-stone-400 dark:text-stone-500">
            Earn points on every order
          </p>
        </div>
        <span className="text-xs px-2 py-1 rounded-full bg-stone-100 dark:bg-stone-800 text-stone-400 dark:text-stone-500 font-medium">
          Phase 2
        </span>
      </div>
    </div>
  );
}