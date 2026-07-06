// src/features/staff/OrderQueueCard.tsx

"use client";

import { useState } from "react";
import { motion } from "framer-motion";
import { Clock, MapPin, Package, ChevronRight } from "lucide-react";
import { useRouter } from "next/navigation";
import { Order, OrderStatus } from "@/types";
import { Card, CardBody } from "@/components/ui/Card";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { formatMoney, getNextStatus } from "@/utils";
import { ROUTES } from "@/constants/routes";
import { staffService } from "@/services/staff.service";
import { toast } from "@/hooks/useToast";
import { cn } from "@/utils";

interface OrderQueueCardProps {
  order: Order;
  onAdvanced: () => void;
}

// How long ago was the order placed
function timeAgo(dateStr: string): string {
  const diff = Math.floor(
    (Date.now() - new Date(dateStr).getTime()) / 1000
  );
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  return `${Math.floor(diff / 3600)}h ago`;
}

const ADVANCE_LABELS: Partial<Record<OrderStatus, string>> = {
  paid: "Start preparing",
  preparing: "Mark ready",
  ready: "Out for delivery",
  out_for_delivery: "Mark completed",
};

export function OrderQueueCard({ order, onAdvanced }: OrderQueueCardProps) {
  const router = useRouter();
  const [isAdvancing, setIsAdvancing] = useState(false);
  const nextStatus = getNextStatus(order.status, order.fulfilment);

  const handleAdvance = async (e: React.MouseEvent) => {
    e.stopPropagation();
    if (!nextStatus) return;
    setIsAdvancing(true);
    try {
      await staffService.transitionOrder(order.id, nextStatus);
      toast.success(`Order marked as ${nextStatus.replace(/_/g, " ")}`);
      onAdvanced();
    } catch {
      toast.error("Could not update order status");
    } finally {
      setIsAdvancing(false);
    }
  };

  const isNew = order.status === "paid";

  return (
    <motion.div
      layout
      initial={{ opacity: 0, scale: 0.97 }}
      animate={{ opacity: 1, scale: 1 }}
      exit={{ opacity: 0, scale: 0.95 }}
      transition={{ duration: 0.2 }}
    >
      <Card
        className={cn(
          "transition-all duration-200",
          // New paid orders are visually loud
          isNew &&
            "border-amber-400 dark:border-amber-600 shadow-amber-100 dark:shadow-amber-900/20 shadow-md"
        )}
      >
        <CardBody className="space-y-3">
          {/* Header row */}
          <div className="flex items-start justify-between gap-2">
            <div>
              <div className="flex items-center gap-2">
                <p className="text-xs font-mono font-bold text-stone-700 dark:text-stone-300">
                  #{String(order.id).slice(-6).toUpperCase()}
                </p>
                {isNew && (
                  <motion.span
                    animate={{ opacity: [1, 0.4, 1] }}
                    transition={{ repeat: Infinity, duration: 1.5 }}
                    className="text-[10px] px-1.5 py-0.5 rounded-full bg-amber-500 text-white font-bold"
                  >
                    NEW
                  </motion.span>
                )}
              </div>
              <div className="flex items-center gap-1.5 mt-1">
                <Clock
                  size={11}
                  className="text-stone-400 dark:text-stone-500"
                />
                <p className="text-xs text-stone-400 dark:text-stone-500">
                  {timeAgo(order.created_at)}
                </p>
              </div>
            </div>

            <Badge
              variant={order.fulfilment === "delivery" ? "info" : "default"}
              className="shrink-0"
            >
              {order.fulfilment === "delivery" ? (
                <MapPin size={10} className="mr-1" />
              ) : (
                <Package size={10} className="mr-1" />
              )}
              {order.fulfilment}
            </Badge>
          </div>

          {/* Items summary */}
          <div className="space-y-1">
            {order.lines.map((line, i) => (
              <p
                key={i}
                className="text-xs text-stone-600 dark:text-stone-400 truncate"
              >
                {line.quantity}× {line.item_name}{" "}
                <span className="text-stone-400">({line.variant_name})</span>
              </p>
            ))}
          </div>

          {/* Footer */}
          <div className="flex items-center justify-between pt-2 border-t border-stone-100 dark:border-stone-800 gap-2">
            <p className="text-sm font-bold text-stone-900 dark:text-stone-100">
              {formatMoney(order.total_pesewas)}
            </p>

            <div className="flex items-center gap-2">
              {/* View detail */}
              <button
                onClick={() =>
                  router.push(ROUTES.STAFF_ORDER(order.id))
                }
                className="w-7 h-7 flex items-center justify-center rounded-lg text-stone-400 hover:text-stone-700 dark:hover:text-stone-300 hover:bg-stone-100 dark:hover:bg-stone-800 transition-colors"
                aria-label="View order detail"
              >
                <ChevronRight size={15} />
              </button>

              {/* Advance button */}
              {nextStatus && ADVANCE_LABELS[order.status] && (
                <Button
                  size="sm"
                  onClick={handleAdvance}
                  isLoading={isAdvancing}
                  className="text-xs"
                >
                  {ADVANCE_LABELS[order.status]}
                </Button>
              )}
            </div>
          </div>
        </CardBody>
      </Card>
    </motion.div>
  );
}