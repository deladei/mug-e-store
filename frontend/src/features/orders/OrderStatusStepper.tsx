// src/features/orders/OrderStatusStepper.tsx

"use client";

import { motion } from "framer-motion";
import { Check, X } from "lucide-react";
import { OrderStatus, Fulfilment } from "@/types";
import { ORDER_STATUS_STEPS, ORDER_STATUS_LABELS } from "@/constants/orderStatuses";
import { getStepColor } from "@/utils";
import { cn } from "@/utils";

interface OrderStatusStepperProps {
  status: OrderStatus;
  fulfilment: Fulfilment;
}

export function OrderStatusStepper({
  status,
  fulfilment,
}: OrderStatusStepperProps) {
  // For pickup, skip out_for_delivery
  const steps = ORDER_STATUS_STEPS.filter(
    (s) =>
      s !== "pending_payment" &&
      s !== "cancelled" &&
      (fulfilment === "delivery" || s !== "out_for_delivery")
  );

  const currentIndex = steps.indexOf(status);

  if (status === "cancelled") {
    return (
      <div className="flex flex-col items-center py-6 space-y-3">
        <div className="w-14 h-14 rounded-full bg-red-100 dark:bg-red-950/30 flex items-center justify-center">
          <X size={24} className="text-red-500" />
        </div>
        <p className="font-semibold text-red-600 dark:text-red-400">
          Order Cancelled
        </p>
        <p className="text-sm text-stone-500 dark:text-stone-400 text-center">
          This order has been cancelled. If you were charged, a refund will be
          processed.
        </p>
      </div>
    );
  }

  if (status === "pending_payment") {
    return (
      <div className="flex flex-col items-center py-6 space-y-3">
        <div className="w-14 h-14 rounded-full bg-amber-100 dark:bg-amber-950/30 flex items-center justify-center">
          <motion.div
            animate={{ rotate: 360 }}
            transition={{ repeat: Infinity, duration: 1.2, ease: "linear" }}
            className="w-6 h-6 border-2 border-amber-600 border-t-transparent rounded-full"
          />
        </div>
        <p className="font-semibold text-stone-800 dark:text-stone-200">
          Confirming your payment…
        </p>
        <p className="text-sm text-stone-500 dark:text-stone-400 text-center">
          Please wait while we confirm your payment. This usually takes a few
          seconds.
        </p>
      </div>
    );
  }

  return (
    <div className="py-4">
      <div className="flex items-start">
        {steps.map((step, index) => {
          const state = getStepColor(index, currentIndex);
          const isLast = index === steps.length - 1;

          return (
            <div
              key={step}
              className={cn(
                "flex flex-col items-center",
                isLast ? "flex-none" : "flex-1"
              )}
            >
              {/* Dot + connector row */}
              <div className="flex items-center w-full">
                {/* Step dot */}
                <motion.div
                  initial={false}
                  animate={{
                    scale: state === "current" ? 1.15 : 1,
                  }}
                  className={cn(
                    "w-8 h-8 rounded-full flex items-center justify-center shrink-0",
                    "border-2 transition-colors duration-300",
                    state === "complete"
                      ? "bg-amber-700 border-amber-700"
                      : state === "current"
                      ? "bg-white dark:bg-stone-900 border-amber-700"
                      : "bg-white dark:bg-stone-900 border-stone-300 dark:border-stone-600"
                  )}
                >
                  {state === "complete" ? (
                    <Check size={14} className="text-white" />
                  ) : state === "current" ? (
                    <motion.div
                      animate={{ scale: [1, 1.3, 1] }}
                      transition={{ repeat: Infinity, duration: 1.5 }}
                      className="w-2.5 h-2.5 rounded-full bg-amber-700"
                    />
                  ) : (
                    <div className="w-2 h-2 rounded-full bg-stone-300 dark:bg-stone-600" />
                  )}
                </motion.div>

                {/* Connector line */}
                {!isLast && (
                  <div className="flex-1 h-0.5 mx-1 relative overflow-hidden bg-stone-200 dark:bg-stone-700">
                    <motion.div
                      initial={false}
                      animate={{
                        width: state === "complete" ? "100%" : "0%",
                      }}
                      transition={{ duration: 0.4 }}
                      className="absolute inset-y-0 left-0 bg-amber-700"
                    />
                  </div>
                )}
              </div>

              {/* Label */}
              <p
                className={cn(
                  "text-[10px] mt-1.5 text-center leading-tight max-w-[52px]",
                  state === "complete" || state === "current"
                    ? "text-amber-700 dark:text-amber-500 font-medium"
                    : "text-stone-400 dark:text-stone-500"
                )}
              >
                {ORDER_STATUS_LABELS[step]}
              </p>
            </div>
          );
        })}
      </div>
    </div>
  );
}