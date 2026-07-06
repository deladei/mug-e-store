// src/features/orders/OrderCard.tsx

"use client";

import { useRouter } from "next/navigation";
import { Order } from "@/types";
import { Card, CardBody } from "@/components/ui/Card";
import { StatusBadge } from "@/components/ui/Badge";
import { formatMoney } from "@/utils";
import { ROUTES } from "@/constants/routes";
import { MapPin, Package } from "lucide-react";

interface OrderCardProps {
  order: Order;
}

export function OrderCard({ order }: OrderCardProps) {
  const router = useRouter();

  const formattedDate = new Date(order.created_at).toLocaleDateString(
    "en-GH",
    {
      day: "numeric",
      month: "short",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    }
  );

  return (
    <Card
      hoverable
      onClick={() => router.push(ROUTES.ORDER(order.id))}
    >
      <CardBody className="space-y-3">
        <div className="flex items-start justify-between gap-2">
          <div>
            <p className="text-xs text-stone-400 dark:text-stone-500 font-mono">
              #{String(order.id).slice(-8).toUpperCase()}
            </p>
            <p className="text-xs text-stone-500 dark:text-stone-400 mt-0.5">
              {formattedDate}
            </p>
          </div>
          <StatusBadge status={order.status} />
        </div>

        <div className="flex items-center gap-1.5 text-xs text-stone-500 dark:text-stone-400">
          {order.fulfilment === "delivery" ? (
            <MapPin size={12} />
          ) : (
            <Package size={12} />
          )}
          <span className="capitalize">{order.fulfilment}</span>
          <span className="text-stone-300 dark:text-stone-600">·</span>
          <span>
            {order.lines.reduce((sum, l) => sum + l.quantity, 0)} item
            {order.lines.reduce((sum, l) => sum + l.quantity, 0) !== 1
              ? "s"
              : ""}
          </span>
        </div>

        <div className="flex justify-between items-center pt-1 border-t border-stone-100 dark:border-stone-800">
          <p className="text-xs text-stone-500 dark:text-stone-400 truncate pr-2">
            {order.lines
              .map((l) => `${l.item_name} (${l.variant_name})`)
              .join(", ")}
          </p>
          <p className="text-sm font-bold text-stone-900 dark:text-stone-100 shrink-0">
            {formatMoney(order.total_pesewas)}
          </p>
        </div>
      </CardBody>
    </Card>
  );
}