// src/features/menu/ItemCard.tsx

"use client";

import Image from "next/image";
import { useRouter } from "next/navigation";
import { motion } from "framer-motion";
import { MenuItem } from "@/types";
import { formatFromPrice } from "@/utils";
import { ROUTES } from "@/constants/routes";
import { cn } from "@/utils";

interface ItemCardProps {
  item: MenuItem;
  index: number; // for staggered animation
}

export function ItemCard({ item, index }: ItemCardProps) {
  const router = useRouter();
  const prices = item.variants.map((v) => v.price_pesewas);

  return (
    <motion.div
      initial={{ opacity: 0, y: 16 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay: index * 0.05, duration: 0.3 }}
      onClick={() => item.is_available && router.push(ROUTES.ITEM(item.id))}
      className={cn(
        "group bg-white dark:bg-stone-900",
        "border border-stone-200 dark:border-stone-800",
        "rounded-2xl overflow-hidden",
        "transition-all duration-200",
        item.is_available
          ? "cursor-pointer hover:shadow-md hover:-translate-y-0.5 active:translate-y-0"
          : "opacity-60 cursor-not-allowed"
      )}
    >
      {/* Image */}
      <div className="relative h-44 bg-stone-100 dark:bg-stone-800 overflow-hidden">
        {item.image_url ? (
          <Image
            src={item.image_url}
            alt={item.name}
            fill
            sizes="(max-width: 672px) 50vw, 320px"
            className={cn(
              "object-cover transition-transform duration-300",
              item.is_available && "group-hover:scale-105"
            )}
          />
        ) : (
          <div className="w-full h-full flex items-center justify-center text-4xl">
            ☕
          </div>
        )}

        {/* Unavailable overlay */}
        {!item.is_available && (
          <div className="absolute inset-0 bg-stone-900/50 flex items-center justify-center">
            <span className="text-white text-xs font-semibold px-3 py-1 bg-stone-900/70 rounded-full">
              Unavailable
            </span>
          </div>
        )}
      </div>

      {/* Content */}
      <div className="p-3 space-y-0.5">
        <h3 className="font-semibold text-stone-900 dark:text-stone-100 text-sm leading-snug">
          {item.name}
        </h3>
        <p className="text-xs text-stone-500 dark:text-stone-400 line-clamp-1">
          {item.description}
        </p>
        <p className="text-sm font-semibold text-amber-700 dark:text-amber-500 pt-1">
          {formatFromPrice(prices)}
        </p>
      </div>
    </motion.div>
  );
}