// src/features/menu/CategoryTabs.tsx

"use client";

import { useRef, useEffect } from "react";
import { motion } from "framer-motion";
import { Category } from "@/types";
import { cn } from "@/utils";
import { Skeleton } from "@/components/ui/Skeleton";

interface CategoryTabsProps {
  categories: Category[];
  selectedId: string | null;
  onSelect: (id: string | null) => void;
  isLoading: boolean;
}

export function CategoryTabs({
  categories,
  selectedId,
  onSelect,
  isLoading,
}: CategoryTabsProps) {
  const scrollRef = useRef<HTMLDivElement>(null);

  // Scroll active tab into view when selection changes
  useEffect(() => {
    if (!scrollRef.current) return;
    const active = scrollRef.current.querySelector("[data-active='true']");
    active?.scrollIntoView({ behavior: "smooth", block: "nearest", inline: "center" });
  }, [selectedId]);

  if (isLoading) {
    return (
      <div className="flex gap-2 overflow-hidden">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-9 w-24 rounded-full shrink-0" />
        ))}
      </div>
    );
  }

  const allTab = { id: null, label: "All" };
  const tabs = [allTab, ...categories.map((c) => ({ id: c.id, label: c.name }))];

  return (
    <div
      ref={scrollRef}
      className="flex gap-2 overflow-x-auto scrollbar-none pb-1"
      style={{ scrollbarWidth: "none" }}
    >
      {tabs.map((tab) => {
        const isActive = tab.id === selectedId;
        return (
          <button
            key={tab.id ?? "all"}
            data-active={isActive}
            onClick={() => onSelect(tab.id)}
            className={cn(
              "relative shrink-0 h-9 px-4 rounded-full text-sm font-medium",
              "transition-colors duration-150",
              "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500",
              isActive
                ? "text-white"
                : "text-stone-600 dark:text-stone-400 hover:text-stone-900 dark:hover:text-stone-100 hover:bg-stone-100 dark:hover:bg-stone-800"
            )}
          >
            {/* Animated pill background */}
            {isActive && (
              <motion.span
                layoutId="category-pill"
                className="absolute inset-0 bg-amber-700 rounded-full"
                transition={{ type: "spring", stiffness: 400, damping: 35 }}
              />
            )}
            <span className="relative z-10">{tab.label}</span>
          </button>
        );
      })}
    </div>
  );
}