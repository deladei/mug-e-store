// src/utils/orderStatus.ts

import { OrderStatus, Fulfilment } from "@/types";
import {
  ORDER_STATUS_STEPS,
  ORDER_STATUS_LABELS,
  NEXT_STATUS,
} from "@/constants/orderStatuses";

// Returns the human-readable label for a status
// "preparing" → "Preparing Your Order"
export function getStatusLabel(status: OrderStatus): string {
  return ORDER_STATUS_LABELS[status];
}

// Returns the index of a status in the progression steps
// Used by the stepper component to know how far along an order is
// "ready" → 3
export function getStatusIndex(status: OrderStatus): number {
  return ORDER_STATUS_STEPS.indexOf(status);
}

// Returns the next legal status for an order given its fulfilment type
// Used to determine which advance button to show on the staff screen
export function getNextStatus(
  current: OrderStatus,
  fulfilment: Fulfilment
): OrderStatus | null {
  return NEXT_STATUS[fulfilment][current] ?? null;
}

// Returns true if the order has reached a terminal state
export function isTerminalStatus(status: OrderStatus): boolean {
  return status === "completed" || status === "cancelled";
}

// Returns the appropriate Tailwind color classes for a status badge
// Returns an object with bg and text classes for flexibility
export function getStatusColors(status: OrderStatus): {
  bg: string;
  text: string;
  border: string;
} {
  const map: Record<OrderStatus, { bg: string; text: string; border: string }> = {
    pending_payment: {
      bg: "bg-yellow-50 dark:bg-yellow-950/30",
      text: "text-yellow-700 dark:text-yellow-400",
      border: "border-yellow-200 dark:border-yellow-800",
    },
    paid: {
      bg: "bg-blue-50 dark:bg-blue-950/30",
      text: "text-blue-700 dark:text-blue-400",
      border: "border-blue-200 dark:border-blue-800",
    },
    preparing: {
      bg: "bg-orange-50 dark:bg-orange-950/30",
      text: "text-orange-700 dark:text-orange-400",
      border: "border-orange-200 dark:border-orange-800",
    },
    ready: {
      bg: "bg-green-50 dark:bg-green-950/30",
      text: "text-green-700 dark:text-green-400",
      border: "border-green-200 dark:border-green-800",
    },
    out_for_delivery: {
      bg: "bg-purple-50 dark:bg-purple-950/30",
      text: "text-purple-700 dark:text-purple-400",
      border: "border-purple-200 dark:border-purple-800",
    },
    completed: {
      bg: "bg-emerald-50 dark:bg-emerald-950/30",
      text: "text-emerald-700 dark:text-emerald-400",
      border: "border-emerald-200 dark:border-emerald-800",
    },
    cancelled: {
      bg: "bg-red-50 dark:bg-red-950/30",
      text: "text-red-700 dark:text-red-400",
      border: "border-red-200 dark:border-red-800",
    },
  };

  return map[status];
}
// Returns the Tailwind color for the stepper dot at a given step index
// relative to the current order step
export function getStepColor(
  stepIndex: number,
  currentIndex: number
): "complete" | "current" | "upcoming" {
  if (stepIndex < currentIndex) return "complete";
  if (stepIndex === currentIndex) return "current";
  return "upcoming";
}