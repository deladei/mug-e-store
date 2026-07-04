// src/features/menu/ItemGrid.tsx

import { MenuItem } from "@/types";
import { ItemCard } from "./ItemCard";
import { ItemCardSkeleton } from "@/components/ui/Skeleton";
import { ShoppingBag } from "lucide-react";

interface ItemGridProps {
  items: MenuItem[];
  isLoading: boolean;
}

export function ItemGrid({ items, isLoading }: ItemGridProps) {
  if (isLoading) {
    return (
      <div className="grid grid-cols-2 gap-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <ItemCardSkeleton key={i} />
        ))}
      </div>
    );
  }

  if (items.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-20 text-center">
        <div className="w-16 h-16 rounded-full bg-stone-100 dark:bg-stone-800 flex items-center justify-center mb-4">
          <ShoppingBag size={28} className="text-stone-400" />
        </div>
        <p className="font-medium text-stone-700 dark:text-stone-300">
          Nothing here yet
        </p>
        <p className="text-sm text-stone-500 dark:text-stone-400 mt-1">
          This category has no available items
        </p>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-2 gap-3">
      {items.map((item, index) => (
        <ItemCard key={item.id} item={item} index={index} />
      ))}
    </div>
  );
}