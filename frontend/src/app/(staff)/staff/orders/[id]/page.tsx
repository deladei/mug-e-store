// src/app/(staff)/staff/orders/[id]/page.tsx

"use client";

import { useParams, useRouter } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { motion } from "framer-motion"; // [cite: 2]
import {
  ArrowLeft,
  MapPin,
  Package,
  Phone,
  Clock,
  AlertTriangle,
} from "lucide-react";
import { staffService } from "@/services/staff.service"; // [cite: 3]
import { Button } from "@/components/ui/Button";
import { Card, CardBody, CardHeader } from "@/components/ui/Card";
import { StatusBadge } from "@/components/ui/Badge"; // [cite: 4]
import { Skeleton } from "@/components/ui/Skeleton";
import { ProtectedRoute } from "@/components/layout/ProtectedRoute";
import { formatMoney, getNextStatus, isTerminalStatus } from "@/utils"; // [cite: 5]
import { ADVANCE_LABELS } from "@/constants/orderStatuses";
import { toast } from "@/hooks/useToast";
import { OrderStatus } from "@/types"; // [cite: 6]
import { cn } from "@/utils";

function StaffOrderDetailContent() { // [cite: 7]
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const queryClient = useQueryClient(); // [cite: 8]

  const {
    data: order,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["staff-order", id],
    queryFn: () => staffService.getOrder(id),
  });

  const { data: history = [], isLoading: historyLoading } = useQuery({ // [cite: 9]
    queryKey: ["staff-order-history", id],
    queryFn: () => staffService.getOrderHistory(id),
    enabled: !!order,
  });

  const transitionMutation = useMutation({ // [cite: 10]
    mutationFn: (to: OrderStatus) => staffService.transitionOrder(id, to),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["staff-order", id] });
      queryClient.invalidateQueries({ queryKey: ["staff-orders"] });
      queryClient.invalidateQueries({
        queryKey: ["staff-order-history", id],
      });
      toast.success("Order status updated");
    },
    onError: () => {
      toast.error("Could not update order status");
    },
  });

  const cancelMutation = useMutation({ // [cite: 11]
    mutationFn: () => staffService.transitionOrder(id, "cancelled"),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["staff-order", id] });
      toast.success("Order cancelled");
    },
    onError: () => {
      toast.error("Could not cancel order");
    },
  });

  if (isLoading) { // [cite: 12]
    return (
      <div className="space-y-4 max-w-2xl">
        <Skeleton className="h-6 w-32" />
        <Skeleton className="h-48 w-full rounded-2xl" />
        <Skeleton className="h-48 w-full rounded-2xl" />
      </div>
    );
  } // [cite: 13]

  if (error || !order) {
    return (
      <div className="flex flex-col items-center justify-center py-24 space-y-4">
        <p className="text-stone-600 dark:text-stone-400">Order not found</p>
        <Button variant="outline" onClick={() => router.back()}>
          Go back
        </Button>
      </div>
    );
  } // [cite: 14]

  const nextStatus = getNextStatus(order.status, order.fulfilment);
  const canCancel = order.status === "paid" || order.status === "preparing";

  return ( // 
    <div className="space-y-5 pb-8 max-w-2xl">
      {/* Header */}
      <div className="flex items-center justify-between">
        <button
          onClick={() => router.back()}
          className="flex items-center gap-1.5 text-sm text-stone-500 hover:text-stone-800 dark:hover:text-stone-200 transition-colors cursor-pointer"
        >
          <ArrowLeft size={16} />
          Queue
        </button> {/* [cite: 16] */}
        <StatusBadge status={order.status} />
      </div>

      {/* Order ID */}
      <div>
        <p className="text-xs text-stone-400 dark:text-stone-500 font-mono">
          Order #{order.id.slice(-8).toUpperCase()}
        </p>
        <p className="text-xs text-stone-500 dark:text-stone-400 mt-0.5">
          {new Date(order.created_at).toLocaleDateString("en-GH", {
            day: "numeric", // [cite: 16]
            month: "long", // [cite: 17]
            year: "numeric",
            hour: "2-digit",
            minute: "2-digit",
          })}
        </p>
      </div>

      {/* Customer + fulfilment */}
      <Card>
        <CardHeader> // [cite: 17]
          <p className="text-sm font-semibold text-stone-700 dark:text-stone-300"> // [cite: 18]
            Fulfilment
          </p>
        </CardHeader>
        <CardBody className="space-y-3">
          <div className="flex items-center gap-2 text-sm text-stone-700 dark:text-stone-300">
            {order.fulfilment === "delivery" ? ( // [cite: 18, 19]
              <MapPin size={15} className="text-stone-400 shrink-0" />
            ) : (
              <Package size={15} className="text-stone-400 shrink-0" />
            )}
            <span className="font-medium capitalize">{order.fulfilment}</span>
          </div>

          {order.address && ( // [cite: 19]
            <p className="text-sm text-stone-500 dark:text-stone-400 pl-6"> // [cite: 20]
              {order.address}
            </p>
          )}

          {order.phone && (
            <div className="flex items-center gap-2 pl-6">
              <Phone size={13} className="text-stone-400" />
              {/* FIXED: Restored truncated link selector tags here */}
              <a // [cite: 20, 21]
                href={`tel:${order.phone}`}
                className="text-sm text-amber-700 dark:text-amber-500 hover:underline"
              >
                {order.phone}
              </a>
            </div> // [cite: 22]
          )}
        </CardBody>
      </Card>

      {/* Line items */}
      <Card>
        <CardHeader>
          <p className="text-sm font-semibold text-stone-700 dark:text-stone-300">
            Items
          </p>
        </CardHeader>
        <CardBody className="space-y-3"> // [cite: 22, 23]
          {order.lines.map((line, i) => (
            <div key={i} className="flex justify-between items-start">
              <div>
                <p className="text-sm font-medium text-stone-800 dark:text-stone-200">
                  {line.quantity}× {line.item_name}
                </p>
                <p className="text-xs text-stone-400 dark:text-stone-500"> // [cite: 24]
                  {line.variant_name}
                </p>
              </div>
              <p className="text-sm font-semibold text-stone-700 dark:text-stone-300">
                {formatMoney(line.unit_price_pesewas * line.quantity)} // [cite: 24, 25]
              </p>
            </div>
          ))}

          <div className="pt-3 border-t border-stone-100 dark:border-stone-800 space-y-1.5">
            <div className="flex justify-between text-sm">
              <span className="text-stone-500 dark:text-stone-400">
                Subtotal
              </span> {/* [cite: 25, 26] */}
              <span>{formatMoney(order.subtotal_pesewas)}</span>
            </div>
            {order.delivery_fee_pesewas > 0 && (
              <div className="flex justify-between text-sm">
                <span className="text-stone-500 dark:text-stone-400"> // [cite: 26]
                  Delivery fee
                </span> {/* [cite: 27] */}
                <span>{formatMoney(order.delivery_fee_pesewas)}</span>
              </div>
            )}
            {order.discount_pesewas > 0 && (
              <div className="flex justify-between text-sm"> {/* [cite: 27, 28] */}
                <span className="text-stone-500 dark:text-stone-400">
                  Discount
                </span>
                <span className="text-emerald-600 dark:text-emerald-400">
                  −{formatMoney(order.discount_pesewas)}
                </span> {/* [cite: 29] */}
              </div>
            )}
            <div className="flex justify-between font-bold text-base pt-1 border-t border-stone-100 dark:border-stone-800">
              <span>Total</span>
              <span>{formatMoney(order.total_pesewas)}</span>
            </div>
          </div> // [cite: 30]
        </CardBody>
      </Card>

      {/* Status history timeline */}
      <Card>
        <CardHeader>
          <p className="text-sm font-semibold text-stone-700 dark:text-stone-300">
            Status history
          </p>
        </CardHeader>
        <CardBody className="space-y-3">
          {historyLoading ? ( // [cite: 30, 31]
            <div className="space-y-2">
              {Array.from({ length: 2 }).map((_, i) => (
                <Skeleton key={i} className="h-10 w-full" />
              ))}
            </div>
          ) : history.length === 0 ? ( // [cite: 31, 32]
            <p className="text-sm text-stone-400 dark:text-stone-500">
              No history yet
            </p>
          ) : (
            <div className="space-y-3">
              {history.map((entry, i) => (
                <motion.div // [cite: 32, 33]
                  key={i}
                  initial={{ opacity: 0, x: -8 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ delay: i * 0.05 }}
                  className="flex items-start gap-3" // [cite: 34]
                >
                  <div className="flex flex-col items-center">
                    <div className="w-2 h-2 rounded-full bg-amber-500 mt-1.5 shrink-0" />
                    {i < history.length - 1 && (
                      <div className="w-px flex-1 bg-stone-200 dark:bg-stone-700 mt-1 mb-0 min-h-[20px]" /> // [cite: 34, 35]
                    )}
                  </div>
                  <div className="flex-1 pb-3">
                    <p className="text-sm text-stone-700 dark:text-stone-300"> // [cite: 35, 36]
                      {entry.from_status ? ( // [cite: 36, 37]
                        <>
                          <span className="font-medium capitalize">
                            {entry.from_status.replace(/_/g, " ")}
                          </span> {/* [cite: 37, 38] */}
                          {" → "}
                          <span className="font-medium capitalize">
                            {entry.to_status.replace(/_/g, " ")}
                          </span> {/* [cite: 38, 39] */}
                        </>
                      ) : (
                        <span className="font-medium capitalize"> // [cite: 39]
                          {entry.to_status.replace(/_/g, " ")} // [cite: 40]
                        </span>
                      )}
                    </p>
                    <div className="flex items-center gap-2 mt-0.5"> // [cite: 40, 41]
                      <Clock
                        size={11}
                        className="text-stone-400 dark:text-stone-500"
                      /> {/* [cite: 41, 42] */}
                      <p className="text-xs text-stone-400 dark:text-stone-500">
                        {new Date(entry.created_at).toLocaleTimeString(
                          "en-GH",
                          { hour: "2-digit", minute: "2-digit" } // [cite: 42, 43]
                        )}
                        {" · "}
                        {entry.actor ?? "System (payment webhook)"} // [cite: 43, 44]
                      </p>
                    </div>
                  </div>
                </motion.div>
              ))}
            </div> // [cite: 44, 45]
          )}
        </CardBody>
      </Card>

      {/* Actions */}
      {!isTerminalStatus(order.status) && ( // [cite: 45]
        <div className="space-y-3">
          {nextStatus && ADVANCE_LABELS[order.status] && (
            <Button
              fullWidth
              size="lg" // [cite: 45, 46]
              isLoading={transitionMutation.isPending}
              onClick={() => transitionMutation.mutate(nextStatus)}
            >
              {ADVANCE_LABELS[order.status]}
            </Button>
          )}

          {canCancel && (
            <Button // [cite: 46, 47]
              fullWidth
              variant="outline"
              isLoading={cancelMutation.isPending}
              onClick={() => cancelMutation.mutate()}
              className={cn(
                "text-red-600 dark:text-red-400",
                "border-red-200 dark:border-red-800", // [cite: 47, 48]
                "hover:bg-red-50 dark:hover:bg-red-950/30"
              )}
            >
              <AlertTriangle size={15} className="mr-2" />
              Cancel order
            </Button> // [cite: 49]
          )}
        </div>
      )}
    </div>
  );
} // [cite: 50]

export default function StaffOrderDetailPage() {
  return (
    <ProtectedRoute requiredRole="staff">
      <StaffOrderDetailContent />
    </ProtectedRoute>
  );
} // [cite: 51]