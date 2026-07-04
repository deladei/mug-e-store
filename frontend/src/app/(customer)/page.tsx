// src/app/(customer)/page.tsx

"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { menuService } from "@/services/menu.service";
import { CategoryTabs } from "@/features/menu/CategoryTabs";
import { ItemGrid } from "@/features/menu/ItemGrid";

export default function HomePage() {
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);

  // Fetch categories
  const { data: categories = [], isLoading: categoriesLoading } = useQuery({
    queryKey: ["categories"],
    queryFn: menuService.getCategories,
  });

  // Fetch items — refetches automatically when selectedCategory changes
  const { data: items = [], isLoading: itemsLoading } = useQuery({
    queryKey: ["items", selectedCategory],
    queryFn: () => menuService.getItems(selectedCategory ?? undefined),
  });

  return (
    <div className="space-y-5">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-stone-900 dark:text-stone-100">
          Our Menu
        </h1>
        <p className="text-stone-500 dark:text-stone-400 text-sm mt-0.5">
          Fresh coffee and pastries, made to order
        </p>
      </div>

      {/* Category tabs */}
      <CategoryTabs
        categories={categories}
        selectedId={selectedCategory}
        onSelect={setSelectedCategory}
        isLoading={categoriesLoading}
      />

      {/* Item grid */}
      <ItemGrid
        items={items}
        isLoading={itemsLoading}
      />
    </div>
  );
}