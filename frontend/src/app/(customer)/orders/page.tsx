// src/app/(customer)/orders/page.tsx

"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { motion } from "framer-motion";
import { ShoppingBag } from "lucide-react";
import { ordersService } from "@/services/orders.service";
import { OrderCard } from "@/features/orders/OrderCard";
import { OrderCardSkeleton } from "@/components/ui/Skeleton";
import { Button } from "@/components/ui/Button";
import { ProtectedRoute } from "@/components/layout/ProtectedRoute";

function OrderHistoryContent() {
  const [page, setPage] = useState(1);

  const { data, isLoading, isFetching } = useQuery({
    queryKey: ["orders", page],
    queryFn: () => ordersService.getOrderHistory(page),
  });

  const orders = data?.orders ?? [];
  const hasMore = orders.length === 20;

  // ── Loading state ──────────────────────────────────────────────────────────
  if (isLoading) {
    return (
      <div className="space-y-4">
        <h1 className="text-2xl font-bold text-stone-900 dark:text-stone-100">
          Your Orders
        </h1>
        <div className="space-y-3">
          {Array.from({ length: 4 }).map((_, i) => (
            <OrderCardSkeleton key={i} />
          ))}
        </div>
      </div>
    );
  }

  // ── Empty state ────────────────────────────────────────────────────────────
  if (orders.length === 0) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-bold text-stone-900 dark:text-stone-100">
          Your Orders
        </h1>
        <div className="flex flex-col items-center justify-center py-24 text-center space-y-4">
          <div className="w-20 h-20 rounded-full bg-stone-100 dark:bg-stone-800 flex items-center justify-center">
            <ShoppingBag
              size={32}
              className="text-stone-400 dark:text-stone-500"
            />
          </div>
          <div>
            <p className="font-semibold text-stone-800 dark:text-stone-200 text-lg">
              No orders yet
            </p>
            <p className="text-sm text-stone-500 dark:text-stone-400 mt-1">
              Your order history will appear here
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-5 pb-8">
      <h1 className="text-2xl font-bold text-stone-900 dark:text-stone-100">
        Your Orders
      </h1>

      <div className="space-y-3">
        {orders.map((order, index) => (
          <motion.div
            key={order.id}
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: index * 0.05, duration: 0.25 }}
          >
            <OrderCard order={order} />
          </motion.div>
        ))}
      </div>

      {/* Pagination */}
      <div className="flex gap-3">
        {page > 1 && (
          <Button
            variant="outline"
            fullWidth
            onClick={() => setPage((p) => p - 1)}
            disabled={isFetching}
          >
            Previous
          </Button>
        )}
        {hasMore && (
          <Button
            variant="outline"
            fullWidth
            onClick={() => setPage((p) => p + 1)}
            isLoading={isFetching}
          >
            Load more
          </Button>
        )}
      </div>
    </div>
  );
}

export default function OrderHistoryPage() {
  return (
    <ProtectedRoute>
      <OrderHistoryContent />
    </ProtectedRoute>
  );
}