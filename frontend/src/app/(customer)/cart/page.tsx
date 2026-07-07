// src/app/(customer)/cart/page.tsx

"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { motion, AnimatePresence } from "framer-motion";
import { ShoppingBag, AlertTriangle } from "lucide-react";
import { useCart } from "@/contexts/CartContext";
import { useAuth } from "@/contexts/AuthContext";
import { Button } from "@/components/ui/Button";
import { Card, CardBody, CardFooter } from "@/components/ui/Card";
import { OrderCardSkeleton } from "@/components/ui/Skeleton";
import { CartLineRow } from "@/components/cart/CartLineRow";
import { formatMoney } from "@/utils";
import { ROUTES } from "@/constants/routes";
import { toast } from "@/hooks/useToast";

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
              <CartLineRow
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