// src/app/(customer)/checkout/page.tsx

"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { motion, AnimatePresence } from "framer-motion";
import { v4 as uuidv4 } from "uuid";
import { MapPin, Phone, Info } from "lucide-react";
import { useAuth } from "@/contexts/AuthContext";
import { useCart } from "@/contexts/CartContext";
import { ordersService } from "@/services/orders.service";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Card, CardBody } from "@/components/ui/Card";
import { Toggle } from "@/components/ui/Toggle";
import { ProtectedRoute } from "@/components/layout/ProtectedRoute";
import { formatMoney } from "@/utils";
import { toast } from "@/hooks/useToast";
import { API_ERROR_CODES } from "@/types";
import {
  checkoutSchema,
  CheckoutFormValues,
} from "@/lib/validators/checkout.schema";
import { ROUTES } from "@/constants/routes";
import {
  openPaystackPopup,
  stashPaymentRetry,
  clearPaymentRetry,
} from "@/lib/paystack";
import { cn } from "@/utils";
import axios from "axios";

// ── Order summary row ─────────────────────────────────────────────────────────
function SummaryRow({
  label,
  value,
  isTotal,
  isDiscount,
}: {
  label: string;
  value: string;
  isTotal?: boolean;
  isDiscount?: boolean;
}) {
  return (
    <div className="flex justify-between items-center">
      <span
        className={cn(
          "text-sm",
          isTotal
            ? "font-semibold text-stone-900 dark:text-stone-100"
            : "text-stone-500 dark:text-stone-400"
        )}
      >
        {label}
      </span>
      <span
        className={cn(
          "text-sm font-semibold",
          isTotal
            ? "text-stone-900 dark:text-stone-100 text-base"
            : isDiscount
            ? "text-emerald-600 dark:text-emerald-400"
            : "text-stone-700 dark:text-stone-300"
        )}
      >
        {value}
      </span>
    </div>
  );
}

// ── Main checkout page ────────────────────────────────────────────────────────

function CheckoutContent() {
  const router = useRouter();
  const { cart, fetchCart, clearLocalCart, openCart } = useCart();
  const { user } = useAuth();

  const [isDelivery, setIsDelivery] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [idempotencyKey] = useState(() => uuidv4());

  // Loyalty state
  const [pointsToRedeem, setPointsToRedeem] = useState(0);
  const [loyaltyBalance] = useState(340); // will come from API in real backend

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors },
  } = useForm<CheckoutFormValues>({
    resolver: zodResolver(checkoutSchema),
    defaultValues: {
      fulfilment: "pickup",
      points_to_redeem: 0,
    },
  });

  // Sync toggle state with form
  useEffect(() => {
    setValue("fulfilment", isDelivery ? "delivery" : "pickup");
  }, [isDelivery, setValue]);

  useEffect(() => {
    fetchCart();
  }, [fetchCart]);

  if (!cart || cart.lines.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-24 text-center space-y-4">
        <p className="font-semibold text-stone-800 dark:text-stone-200">
          Your cart is empty
        </p>
        <Button onClick={() => router.push(ROUTES.HOME)}>Browse Menu</Button>
      </div>
    );
  }

  // Delivery fee comes from the order response — we show 0 until confirmed
  // In mock mode we hardcode 500 pesewas (GHS 5.00) for delivery
  const deliveryFee = isDelivery ? 500 : 0;

  // Points are worth 1 pesewa each, capped at subtotal
  const maxRedeemable = Math.min(loyaltyBalance, cart.subtotal_pesewas);
  const discount = Math.min(pointsToRedeem, maxRedeemable);
  const total = cart.subtotal_pesewas + deliveryFee - discount;

  const onSubmit = async (values: CheckoutFormValues) => {
    setIsSubmitting(true);
    try {
      const response = await ordersService.checkout({
        fulfilment: values.fulfilment,
        address: values.address,
        phone: values.phone,
        idempotency_key: idempotencyKey,
        points_to_redeem: pointsToRedeem > 0 ? pointsToRedeem : undefined,
      });

      // The backend cleared the cart server-side; clearing the local copy
      // waits until we leave, so the empty-cart branch doesn't flash behind
      // the popup.
      const goToOrder = () => {
        clearLocalCart();
        router.push(ROUTES.ORDER(response.order.id));
      };

      // If the customer cancels the popup, the order page offers "Pay now"
      // for the rest of this session.
      stashPaymentRetry(String(response.order.id), {
        access_code: response.access_code,
        authorization_url: response.authorization_url,
      });

      // Collect payment in an on-page Paystack popup. `paid` still comes only
      // from Paystack's webhook — the order page tracks that live over SSE, so
      // onSuccess just navigates there. If the inline SDK can't load, fall back
      // to Paystack's hosted page.
      try {
        await openPaystackPopup(response.access_code, {
          onSuccess: () => {
            clearPaymentRetry(String(response.order.id));
            goToOrder();
          },
          onCancel: () => {
            toast.error("Payment cancelled — your order is awaiting payment");
            goToOrder();
          },
          onError: () => {
            window.location.assign(response.authorization_url);
          },
        });
      } catch {
        // SDK failed to load — use the hosted checkout page instead.
        window.location.assign(response.authorization_url);
      }
    } catch (err) {
      if (axios.isAxiosError(err)) {
        const code = err.response?.data?.error?.code;
        if (code === API_ERROR_CODES.EMPTY_CART) {
          toast.error("Your cart is empty");
          router.push(ROUTES.HOME);
        } else if (code === API_ERROR_CODES.UNAVAILABLE) {
          toast.error("Some items are unavailable — check your cart");
          openCart();
        } else if (code === API_ERROR_CODES.INSUFFICIENT_POINTS) {
          toast.error("Not enough loyalty points");
          setPointsToRedeem(0);
        } else if (code === API_ERROR_CODES.DUPLICATE_ORDER) {
          toast.error("This order was already placed");
        } else if (code === API_ERROR_CODES.PAYMENT_INIT_FAILED) {
          toast.error("Payment could not start — please try again");
        } else {
          toast.error("Something went wrong — please try again");
        }
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="space-y-5 pb-8">
      <h1 className="text-2xl font-bold text-stone-900 dark:text-stone-100">
        Checkout
      </h1>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-5">
        {/* Fulfilment toggle */}
        <Card>
          <CardBody>
            <div className="space-y-4">
              <p className="text-sm font-semibold text-stone-700 dark:text-stone-300">
                How would you like to receive your order?
              </p>

              <div className="flex gap-3">
                {(["pickup", "delivery"] as const).map((option) => (
                  <button
                    key={option}
                    type="button"
                    onClick={() => setIsDelivery(option === "delivery")}
                    className={cn(
                      "flex-1 py-3 px-4 rounded-xl border text-sm font-medium",
                      "transition-all duration-150",
                      "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500",
                      (option === "delivery") === isDelivery
                        ? "border-amber-700 bg-amber-50 dark:bg-amber-950/30 text-amber-800 dark:text-amber-300"
                        : "border-stone-200 dark:border-stone-700 text-stone-600 dark:text-stone-400"
                    )}
                  >
                    {option === "pickup" ? "🏪 Pickup" : "🛵 Delivery"}
                  </button>
                ))}
              </div>

              {/* Delivery fields */}
              <AnimatePresence>
                {isDelivery && (
                  <motion.div
                    initial={{ opacity: 0, height: 0 }}
                    animate={{ opacity: 1, height: "auto" }}
                    exit={{ opacity: 0, height: 0 }}
                    transition={{ duration: 0.2 }}
                    className="space-y-3 overflow-hidden"
                  >
                    <Input
                      label="Delivery address"
                      placeholder="e.g. 14 Ring Road, Accra"
                      error={errors.address?.message}
                      leftIcon={<MapPin size={15} />}
                      {...register("address")}
                    />
                    <Input
                      label="Phone number"
                      type="tel"
                      placeholder="0244000001"
                      error={errors.phone?.message}
                      leftIcon={<Phone size={15} />}
                      {...register("phone")}
                    />
                  </motion.div>
                )}
              </AnimatePresence>
            </div>
          </CardBody>
        </Card>

        {/* Order summary */}
        <Card>
          <CardBody className="space-y-3">
            <p className="text-sm font-semibold text-stone-700 dark:text-stone-300">
              Order summary
            </p>

            {/* Line items */}
            <div className="space-y-2 pb-3 border-b border-stone-100 dark:border-stone-800">
              {cart.lines.map((line) => (
                <div
                  key={line.line_id}
                  className="flex justify-between items-start"
                >
                  <div className="flex-1 min-w-0">
                    <p className="text-sm text-stone-700 dark:text-stone-300 truncate">
                      {line.item_name}
                    </p>
                    <p className="text-xs text-stone-400">
                      {line.variant_name} × {line.quantity}
                    </p>
                  </div>
                  <p className="text-sm font-medium text-stone-700 dark:text-stone-300 ml-3 shrink-0">
                    {formatMoney(line.unit_price_pesewas * line.quantity)}
                  </p>
                </div>
              ))}
            </div>

            {/* Totals */}
            <div className="space-y-2">
              <SummaryRow
                label="Subtotal"
                value={formatMoney(cart.subtotal_pesewas)}
              />

              {isDelivery && (
                <SummaryRow
                  label="Delivery fee"
                  value={formatMoney(deliveryFee)}
                />
              )}

              {/* Loyalty redemption */}
              <div className="pt-2 border-t border-stone-100 dark:border-stone-800">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-1.5">
                    <span className="text-sm text-stone-500 dark:text-stone-400">
                      Loyalty points
                    </span>
                    <span className="text-xs px-1.5 py-0.5 rounded-full bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400 font-medium">
                      {loyaltyBalance} pts
                    </span>
                  </div>
                  <div onClick={(e) => e.preventDefault()}>
                    <Toggle
                    size="sm"
                    checked={pointsToRedeem > 0}
                    onChange={(checked) =>
                      setPointsToRedeem(checked ? maxRedeemable : 0)
                    }
                    disabled={loyaltyBalance === 0 || maxRedeemable === 0}
                  />
                  </div>
                </div>

                {pointsToRedeem > 0 && (
                  <motion.div
                    initial={{ opacity: 0, height: 0 }}
                    animate={{ opacity: 1, height: "auto" }}
                    exit={{ opacity: 0, height: 0 }}
                  >
                    <SummaryRow
                      label={`Points discount (${pointsToRedeem} pts)`}
                      value={`−${formatMoney(discount)}`}
                      isDiscount
                    />
                  </motion.div>
                )}
              </div>

              {/* Total */}
              <div className="pt-2 border-t border-stone-100 dark:border-stone-800">
                <SummaryRow
                  label="Total"
                  value={formatMoney(total)}
                  isTotal
                />
              </div>
            </div>
          </CardBody>
        </Card>

        {/* Paystack redirect notice */}
        <div className="flex items-start gap-2.5 px-4 py-3 rounded-xl bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800">
          <Info size={15} className="text-blue-500 shrink-0 mt-0.5" />
          <p className="text-xs text-blue-700 dark:text-blue-400 leading-relaxed">
            A secure Paystack payment window opens right here on this page. Your
            order confirms automatically once payment clears.
          </p>
        </div>

        {/* CTA */}
        <Button
          type="submit"
          fullWidth
          size="lg"
          isLoading={isSubmitting}
        >
          Pay {formatMoney(total)} with Paystack
        </Button>
      </form>
    </div>
  );
}

export default function CheckoutPage() {
  return (
    <ProtectedRoute>
      <CheckoutContent />
    </ProtectedRoute>
  );
}