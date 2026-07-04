// src/features/staff/OrderQueueBoard.tsx

"use client";

import { AnimatePresence } from "framer-motion";
import { Order, OrderStatus } from "@/types";
import { OrderQueueCard } from "./OrderQueueCard";
import { Skeleton } from "@/components/ui/Skeleton";
import { cn } from "@/utils";

interface Column {
  status: OrderStatus;
  label: string;
  accent: string;
}

const COLUMNS: Column[] = [
  {
    status: "paid",
    label: "New",
    accent: "border-t-amber-500",
  },
  {
    status: "preparing",
    label: "Preparing",
    accent: "border-t-orange-500",
  },
  {
    status: "ready",
    label: "Ready",
    accent: "border-t-green-500",
  },
  {
    status: "out_for_delivery",
    label: "Out for delivery",
    accent: "border-t-purple-500",
  },
];

interface OrderQueueBoardProps {
  orders: Order[];
  isLoading: boolean;
  onAdvanced: () => void;
}

export function OrderQueueBoard({
  orders,
  isLoading,
  onAdvanced,
}: OrderQueueBoardProps) {
  if (isLoading) {
    return (
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        {COLUMNS.map((col) => (
          <div key={col.status} className="space-y-3">
            <Skeleton className="h-6 w-24" />
            <Skeleton className="h-40 w-full rounded-2xl" />
            <Skeleton className="h-40 w-full rounded-2xl" />
          </div>
        ))}
      </div>
    );
  }

  return (
    <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
      {COLUMNS.map((col) => {
        const colOrders = orders.filter((o) => o.status === col.status);
        return (
          <div key={col.status} className="space-y-3">
            {/* Column header */}
            <div
              className={cn(
                "bg-white dark:bg-stone-900",
                "border border-stone-200 dark:border-stone-800",
                "border-t-4",
                col.accent,
                "rounded-xl px-3 py-2",
                "flex items-center justify-between"
              )}
            >
              <p className="text-sm font-semibold text-stone-700 dark:text-stone-300">
                {col.label}
              </p>
              <span
                className={cn(
                  "text-xs font-bold w-5 h-5 rounded-full flex items-center justify-center",
                  colOrders.length > 0
                    ? "bg-amber-100 dark:bg-amber-900/40 text-amber-700 dark:text-amber-400"
                    : "bg-stone-100 dark:bg-stone-800 text-stone-400 dark:text-stone-500"
                )}
              >
                {colOrders.length}
              </span>
            </div>

            {/* Cards */}
            <div className="space-y-3 min-h-[120px]">
              <AnimatePresence mode="popLayout">
                {colOrders.length === 0 ? (
                  <div className="flex items-center justify-center h-24 rounded-xl border-2 border-dashed border-stone-200 dark:border-stone-800">
                    <p className="text-xs text-stone-400 dark:text-stone-500">
                      No orders
                    </p>
                  </div>
                ) : (
                  colOrders.map((order) => (
                    <OrderQueueCard
                      key={order.id}
                      order={order}
                      onAdvanced={onAdvanced}
                    />
                  ))
                )}
              </AnimatePresence>
            </div>
          </div>
        );
      })}
    </div>
  );
}