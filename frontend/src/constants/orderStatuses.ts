// constants/orderStatuses.ts

import { OrderStatus } from "@/types";

// The exact progression the API defines — do not reorder or add to this
export const ORDER_STATUS_STEPS: OrderStatus[] = [
  "pending_payment",
  "paid",
  "preparing",
  "ready",
  "out_for_delivery",
  "completed",
];

export const ORDER_STATUS_LABELS: Record<OrderStatus, string> = {
  pending_payment: "Confirming Payment",
  paid: "Payment Confirmed",
  preparing: "Preparing Your Order",
  ready: "Ready for Pickup",
  out_for_delivery: "Out for Delivery",
  completed: "Completed",
  cancelled: "Cancelled",
};

// FIX: Combined Record and opened the angle bracket properly on the same line
export const NEXT_STATUS: Record<
  string,
  Partial<Record<OrderStatus, OrderStatus>>
> = {
  pickup: {
    paid: "preparing",
    preparing: "ready",
    ready: "completed",
  },
  delivery: {
    paid: "preparing",
    preparing: "ready",
    ready: "out_for_delivery",
    out_for_delivery: "completed",
  },
};

export const ADVANCE_LABELS: Partial<Record<OrderStatus, string>> = {
  paid: "Start preparing",
  preparing: "Mark ready",
  ready: "Out for delivery",
  out_for_delivery: "Mark completed",
};