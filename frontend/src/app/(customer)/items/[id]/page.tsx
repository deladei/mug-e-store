// src/app/(customer)/items/[id]/page.tsx

"use client";

import { useState, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import Image from "next/image";
import { ArrowLeft, Minus, Plus } from "lucide-react";
import { menuService } from "@/services/menu.service";
import { useCart } from "@/contexts/CartContext";
import { useAuth } from "@/contexts/AuthContext";
import { Button } from "@/components/ui/Button";
import { Skeleton } from "@/components/ui/Skeleton";
import { Variant } from "@/types";
import { formatMoney } from "@/utils";
import { toast } from "@/hooks/useToast";
import { ROUTES } from "@/constants/routes";
import { cn } from "@/utils";

export default function ItemDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { isAuthenticated } = useAuth();
  const { addItem } = useCart();

  const [selectedVariant, setSelectedVariant] = useState<Variant | null>(null);
  const [quantity, setQuantity] = useState(1);
  const [isAdding, setIsAdding] = useState(false);

  const { data: item, isLoading, isError } = useQuery({
    queryKey: ["item", id],
    queryFn: () => menuService.getItem(id),
  });

  // OPTIMIZATION: Automatically pre-select the first option once the data loads
  useEffect(() => {
    if (item?.variants && item.variants.length > 0 && !selectedVariant) {
      const sorted = [...item.variants].sort((a, b) => a.sort_order - b.sort_order);
      setSelectedVariant(sorted[0]);
    }
  }, [item, selectedVariant]);

  const handleAddToCart = async () => {
  if (!selectedVariant) return;

  if (!isAuthenticated) {
    router.push(ROUTES.AUTH);
    return;
  }

  setIsAdding(true);
  try {
    await addItem({
      item_variant_id: selectedVariant.id,
      quantity,
    });
    toast.success(`${item?.name} added to cart`);
    router.back();
  } catch (err) {
    console.error("ADD TO CART ERROR:", err);
    toast.error("Could not add item — please try again");
  } finally {
    setIsAdding(false);
  }
};

  // ── Error state ────────────────────────────────────────────────────────────
  if (isError) {
    return (
      <div className="flex flex-col items-center justify-center py-24 text-center space-y-4">
        <span className="text-5xl">☕</span>
        <p className="font-semibold text-stone-800 dark:text-stone-200">
          Item no longer available
        </p>
        <p className="text-sm text-stone-500">
          This item may have been removed from the menu.
        </p>
        <Button variant="outline" onClick={() => router.push(ROUTES.HOME)}>
          Back to menu
        </Button>
      </div>
    );
  }

  // ── Loading state ──────────────────────────────────────────────────────────
  if (isLoading || !item) { // FIXED: Keep loading visible if item data hasn't fully hydrated yet
    return (
      <div className="space-y-5">
        <Skeleton className="h-64 w-full rounded-2xl" />
        <Skeleton className="h-6 w-2/3" />
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-4/5" />
        <div className="flex gap-2 mt-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-10 flex-1 rounded-xl" />
          ))}
        </div>
        <Skeleton className="h-12 w-full rounded-xl mt-4" />
      </div>
    );
  }

  // FIXED: Evaluated down here safely AFTER the loading and existence validations are complete
  const totalPrice = selectedVariant
    ? formatMoney(selectedVariant.price_pesewas * quantity)
    : null;

  return (
    <div className="space-y-6 pb-8">
      {/* Back button */}
      <button
        onClick={() => router.back()}
        className="flex items-center gap-1.5 text-sm text-stone-500 hover:text-stone-800 dark:hover:text-stone-200 transition-colors cursor-pointer"
      >
        <ArrowLeft size={16} />
        Back
      </button>

      {/* Image */}
      <div className="relative h-64 rounded-2xl overflow-hidden bg-stone-100 dark:bg-stone-800">
        {item.image_url ? (
          <Image
            src={item.image_url}
            alt={item.name}
            fill
            sizes="(max-width: 672px) 100vw, 672px"
            className="object-cover"
            priority
          />
        ) : (
          <div className="w-full h-full flex items-center justify-center text-6xl">
            ☕
          </div>
        )}
      </div>

      {/* Name + description */}
      <div className="space-y-1.5">
        <h1 className="text-2xl font-bold text-stone-900 dark:text-stone-100">
          {item.name}
        </h1>
        <p className="text-stone-500 dark:text-stone-400 text-sm leading-relaxed">
          {item.description}
        </p>
      </div>

      {/* Variant selector */}
      <div className="space-y-2.5">
        <p className="text-sm font-semibold text-stone-700 dark:text-stone-300">
          Choose size
        </p>
        <div className="flex flex-col gap-2">
          {(item.variants || [])
            .slice()
            .sort((a, b) => a.sort_order - b.sort_order)
            .map((variant) => {
              const isSelected = selectedVariant?.id === variant.id;
              return (
                <button
                  key={variant.id}
                  onClick={() => setSelectedVariant(variant)}
                  className={cn(
                    "flex items-center justify-between px-4 py-3 rounded-xl border text-sm cursor-pointer",
                    "transition-all duration-150",
                    "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-500",
                    isSelected
                      ? "border-amber-700 bg-amber-50 dark:bg-amber-950/30 text-amber-800 dark:text-amber-300"
                      : "border-stone-200 dark:border-stone-700 text-stone-700 dark:text-stone-300 hover:border-stone-300 dark:hover:border-stone-600"
                  )}
                >
                  <span className="font-medium">{variant.name}</span>
                  <span
                    className={cn(
                      "font-semibold",
                      isSelected
                        ? "text-amber-700 dark:text-amber-400"
                        : "text-stone-600 dark:text-stone-400"
                    )}
                  >
                    {formatMoney(variant.price_pesewas)}
                  </span>
                </button>
              );
            })}
        </div>

        {/* Variant required hint */}
        {!selectedVariant && (
          <p className="text-xs text-stone-400 dark:text-stone-500">
            Select a size to continue
          </p>
        )}
      </div>

      {/* Quantity stepper */}
      <div className="space-y-2.5">
        <p className="text-sm font-semibold text-stone-700 dark:text-stone-300">
          Quantity
        </p>
        <div className="flex items-center gap-4">
          <button
            onClick={() => setQuantity((q) => Math.max(1, q - 1))}
            disabled={quantity <= 1}
            aria-label="Decrease quantity"
            className={cn(
              "w-10 h-10 rounded-xl border border-stone-200 dark:border-stone-700 cursor-pointer",
              "flex items-center justify-center",
              "text-stone-600 dark:text-stone-400",
              "hover:bg-stone-100 dark:hover:bg-stone-800",
              "disabled:opacity-40 disabled:cursor-not-allowed disabled:cursor-default",
              "transition-colors"
            )}
          >
            <Minus size={16} />
          </button>
          <span className="text-lg font-semibold text-stone-900 dark:text-stone-100 w-6 text-center">
            {quantity}
          </span>
          <button
            onClick={() => setQuantity((q) => Math.min(20, q + 1))}
            disabled={quantity >= 20}
            aria-label="Increase quantity"
            className={cn(
              "w-10 h-10 rounded-xl border border-stone-200 dark:border-stone-700 cursor-pointer",
              "flex items-center justify-center",
              "text-stone-600 dark:text-stone-400",
              "hover:bg-stone-100 dark:hover:bg-stone-800",
              "disabled:opacity-40 disabled:cursor-not-allowed disabled:cursor-default",
              "transition-colors"
            )}
          >
            <Plus size={16} />
          </button>
        </div>
      </div>

      {/* Add to cart CTA */}
      <Button
        fullWidth
        size="lg"
        onClick={handleAddToCart}
        disabled={!selectedVariant}
        isLoading={isAdding}
      >
        {totalPrice ? `Add to Cart — ${totalPrice}` : "Select a size"}
      </Button>
    </div>
  );
}