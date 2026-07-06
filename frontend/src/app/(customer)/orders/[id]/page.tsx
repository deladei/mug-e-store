// src/app/(customer)/orders/[id]/page.tsx

"use client";

import { useParams, useRouter } from "next/navigation";
import { motion } from "framer-motion";
import { RefreshCw, ArrowLeft, MapPin, Package } from "lucide-react";
import { useLiveOrder } from "@/features/orders/useLiveOrder";
import { OrderStatusStepper } from "@/features/orders/OrderStatusStepper";
import { Button } from "@/components/ui/Button";
import { Card, CardBody, CardHeader } from "@/components/ui/Card";
import { StatusBadge } from "@/components/ui/Badge";
import { Skeleton } from "@/components/ui/Skeleton";
import { ProtectedRoute } from "@/components/layout/ProtectedRoute";
import { formatMoney } from "@/utils";
import { ROUTES } from "@/constants/routes";

function OrderTrackingContent() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { order, isLoading, error } = useLiveOrder(id);

  // ── Error state ────────────────────────────────────────────────────────────
  if (error) {
    return (
      <div className="flex flex-col items-center justify-center py-24 text-center space-y-4">
        <span className="text-5xl">⚠️</span>
        <p className="font-semibold text-stone-800 dark:text-stone-200">
          {error}
        </p>
        <Button
          variant="outline"
          onClick={() => window.location.reload()}
          className="gap-2"
        >
          <RefreshCw size={15} />
          Retry
        </Button>
      </div>
    );
  }

  // ── Loading state ──────────────────────────────────────────────────────────
  if (isLoading || !order) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-6 w-32" />
        <Skeleton className="h-36 w-full rounded-2xl" />
        <Skeleton className="h-48 w-full rounded-2xl" />
      </div>
    );
  }

  const formattedDate = new Date(order.created_at).toLocaleDateString(
    "en-GH",
    {
      day: "numeric",
      month: "long",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    }
  );

  // ── Payment failed state ───────────────────────────────────────────────────
  if (order.status === "cancelled") {
    return (
      <div className="space-y-5 pb-8">
        <button
          onClick={() => router.push(ROUTES.HOME)}
          className="flex items-center gap-1.5 text-sm text-stone-500 hover:text-stone-800 dark:hover:text-stone-200 transition-colors"
        >
          <ArrowLeft size={16} />
          Back to menu
        </button>

        <Card>
          <CardBody>
            <OrderStatusStepper
              status="cancelled"
              fulfilment={order.fulfilment}
            />
          </CardBody>
        </Card>

        <div className="space-y-3">
          <Button
            fullWidth
            size="lg"
            onClick={() => router.push(ROUTES.CHECKOUT)}
          >
            Try again
          </Button>
          <Button
            fullWidth
            variant="ghost"
            onClick={() => router.push(ROUTES.HOME)}
          >
            Back to menu
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-5 pb-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <button
          onClick={() => router.push(ROUTES.ORDER_HISTORY)}
          className="flex items-center gap-1.5 text-sm text-stone-500 hover:text-stone-800 dark:hover:text-stone-200 transition-colors"
        >
          <ArrowLeft size={16} />
          Orders
        </button>
        <StatusBadge status={order.status} />
      </div>

      {/* Order number + date */}
      <div>
        <p className="text-xs text-stone-400 dark:text-stone-500 font-mono">
          Order #{String(order.id).slice(-8).toUpperCase()}
        </p>
        <p className="text-xs text-stone-500 dark:text-stone-400 mt-0.5">
          {formattedDate}
        </p>
      </div>

      {/* Status stepper */}
      <motion.div
        key={order.status}
        initial={{ opacity: 0.8 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.3 }}
      >
        <Card>
          <CardBody>
            <OrderStatusStepper
              status={order.status}
              fulfilment={order.fulfilment}
            />

            {/* Live update notice */}
            {order.status !== "completed" && (
              <p className="text-center text-xs text-stone-400 dark:text-stone-500 mt-2">
                This page updates automatically
              </p>
            )}
          </CardBody>
        </Card>
      </motion.div>

      {/* Fulfilment details */}
      <Card>
        <CardBody className="space-y-3">
          <div className="flex items-center gap-2">
            {order.fulfilment === "delivery" ? (
              <MapPin size={15} className="text-stone-400" />
            ) : (
              <Package size={15} className="text-stone-400" />
            )}
            <p className="text-sm font-medium text-stone-700 dark:text-stone-300 capitalize">
              {order.fulfilment}
            </p>
          </div>

          {order.fulfilment === "delivery" && order.address && (
            <p className="text-sm text-stone-500 dark:text-stone-400 pl-6">
              {order.address}
            </p>
          )}

          {order.fulfilment === "delivery" && order.phone && (
            <p className="text-sm text-stone-500 dark:text-stone-400 pl-6">
              {order.phone}
            </p>
          )}
        </CardBody>
      </Card>

      {/* Order items */}
      <Card>
        <CardHeader>
          <p className="text-sm font-semibold text-stone-700 dark:text-stone-300">
            Items
          </p>
        </CardHeader>
        <CardBody className="space-y-3 py-3">
          {order.lines.map((line, i) => (
            <div key={i} className="flex justify-between items-start">
              <div className="flex-1 min-w-0">
                <p className="text-sm text-stone-800 dark:text-stone-200">
                  {line.item_name}
                </p>
                <p className="text-xs text-stone-400 dark:text-stone-500">
                  {line.variant_name} × {line.quantity}
                </p>
              </div>
              <p className="text-sm font-medium text-stone-700 dark:text-stone-300 ml-3 shrink-0">
                {formatMoney(line.unit_price_pesewas * line.quantity)}
              </p>
            </div>
          ))}
        </CardBody>

        {/* Totals */}
        <CardBody className="border-t border-stone-100 dark:border-stone-800 space-y-2 pt-3">
          <div className="flex justify-between text-sm">
            <span className="text-stone-500 dark:text-stone-400">Subtotal</span>
            <span className="text-stone-700 dark:text-stone-300">
              {formatMoney(order.subtotal_pesewas)}
            </span>
          </div>

          {order.delivery_fee_pesewas > 0 && (
            <div className="flex justify-between text-sm">
              <span className="text-stone-500 dark:text-stone-400">
                Delivery fee
              </span>
              <span className="text-stone-700 dark:text-stone-300">
                {formatMoney(order.delivery_fee_pesewas)}
              </span>
            </div>
          )}

          {order.discount_pesewas > 0 && (
            <div className="flex justify-between text-sm">
              <span className="text-stone-500 dark:text-stone-400">
                Points discount
              </span>
              <span className="text-emerald-600 dark:text-emerald-400">
                −{formatMoney(order.discount_pesewas)}
              </span>
            </div>
          )}

          <div className="flex justify-between font-bold text-base pt-1 border-t border-stone-100 dark:border-stone-800">
            <span className="text-stone-900 dark:text-stone-100">Total</span>
            <span className="text-stone-900 dark:text-stone-100">
              {formatMoney(order.total_pesewas)}
            </span>
          </div>
        </CardBody>
      </Card>

      {/* Actions */}
      {order.status === "completed" && (
        <Button
          fullWidth
          variant="outline"
          onClick={() => router.push(ROUTES.HOME)}
        >
          Order again
        </Button>
      )}
    </div>
  );
}

export default function OrderTrackingPage() {
  return (
    <ProtectedRoute>
      <OrderTrackingContent />
    </ProtectedRoute>
  );
}