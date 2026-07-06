// src/app/(admin)/admin/menu/page.tsx

"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { motion, AnimatePresence } from "framer-motion";
import {
  Plus,
  Pencil,
  Trash2,
  ChevronDown,
  ChevronUp,
  X,
  Check,
} from "lucide-react";
import { adminService } from "@/services/admin.service";
import { menuService } from "@/services/menu.service";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Card, CardBody, CardHeader } from "@/components/ui/Card";
import { Toggle } from "@/components/ui/Toggle";
import { Skeleton } from "@/components/ui/Skeleton";
import { Badge } from "@/components/ui/Badge";
import { ProtectedRoute } from "@/components/layout/ProtectedRoute";
import { MenuItem, Category, Variant } from "@/types";
import { formatMoney, floatToPesewas, pesewasToFloat } from "@/utils";
import { toast } from "@/hooks/useToast";
import { cn } from "@/utils";

// ── Variant row ────────────────────────────────────────────────────────────────

function VariantRow({
  variant,
  itemId,
  onDeleted,
}: {
  variant: Variant;
  itemId: string;
  onDeleted: () => void;
}) {
  const queryClient = useQueryClient();
  const deleteMutation = useMutation({
    mutationFn: () => adminService.deleteVariant(variant.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-items"] });
      toast.success("Variant deleted");
      onDeleted();
    },
    onError: () => toast.error("Could not delete variant"),
  });

  return (
    <div className="flex items-center justify-between py-1.5">
      <div className="flex items-center gap-3">
        <span className="text-sm text-stone-700 dark:text-stone-300">
          {variant.name}
        </span>
        <span className="text-sm font-semibold text-amber-700 dark:text-amber-500">
          {formatMoney(variant.price_pesewas)}
        </span>
      </div>
      <button
        onClick={() => deleteMutation.mutate()}
        disabled={deleteMutation.isPending}
        className="w-6 h-6 flex items-center justify-center rounded-lg text-stone-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-950/30 transition-colors disabled:opacity-40"
      >
        <X size={13} />
      </button>
    </div>
  );
}

// ── Add variant form ───────────────────────────────────────────────────────────

function AddVariantForm({
  itemId,
  onAdded,
}: {
  itemId: string;
  onAdded: () => void;
}) {
  const queryClient = useQueryClient();
  const [name, setName] = useState("");
  const [price, setPrice] = useState("");

  const addMutation = useMutation({
    mutationFn: () =>
      adminService.createVariant(itemId, {
        name,
        price_pesewas: floatToPesewas(parseFloat(price)),
        sort_order: 99,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-items"] });
      setName("");
      setPrice("");
      toast.success("Variant added");
      onAdded();
    },
    onError: () => toast.error("Could not add variant"),
  });

  const canAdd = name.trim().length > 0 && parseFloat(price) > 0;

  return (
    <div className="flex items-end gap-2 pt-2 border-t border-stone-100 dark:border-stone-800">
      <div className="flex-1">
        <Input
          placeholder="Size name (e.g. Large)"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
      </div>
      <div className="w-28">
        <Input
          placeholder="Price (GHS)"
          type="number"
          step="0.01"
          min="0"
          value={price}
          onChange={(e) => setPrice(e.target.value)}
        />
      </div>
      <Button
        size="md"
        disabled={!canAdd}
        isLoading={addMutation.isPending}
        onClick={() => addMutation.mutate()}
      >
        <Check size={15} />
      </Button>
    </div>
  );
}

// ── Item row ───────────────────────────────────────────────────────────────────

function ItemRow({
  item,
  categories,
}: {
  item: MenuItem;
  categories: Category[];
}) {
  const queryClient = useQueryClient();
  const [expanded, setExpanded] = useState(false);
  const [isEditing, setIsEditing] = useState(false);
  const [editName, setEditName] = useState(item.name);
  const [editDescription, setEditDescription] = useState(item.description);
  const [editCategoryId, setEditCategoryId] = useState(item.category_id);

  const availabilityMutation = useMutation({
    mutationFn: (val: boolean) =>
      adminService.updateAvailability(item.id, { is_available: val }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-items"] });
    },
    onError: () => toast.error("Could not update availability"),
  });

  const updateMutation = useMutation({
    mutationFn: () =>
      adminService.updateItem(item.id, {
        name: editName,
        description: editDescription,
        // Select inputs yield strings; the backend expects a numeric id
        category_id: Number(editCategoryId),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-items"] });
      setIsEditing(false);
      toast.success("Item updated");
    },
    onError: () => toast.error("Could not update item"),
  });

  const deleteMutation = useMutation({
    mutationFn: () => adminService.deleteItem(item.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-items"] });
      toast.success("Item deleted");
    },
    onError: () => toast.error("Could not delete item"),
  });

  return (
    <div
      className={cn(
        "border border-stone-200 dark:border-stone-800 rounded-xl overflow-hidden",
        "transition-colors duration-150",
        !item.is_available && "opacity-60"
      )}
    >
      {/* Item header row */}
      <div className="flex items-center gap-3 px-4 py-3 bg-white dark:bg-stone-900">
        {/* Availability toggle */}
        <Toggle
          size="sm"
          checked={item.is_available}
          onChange={(val) => availabilityMutation.mutate(val)}
          disabled={availabilityMutation.isPending}
        />

        {/* Name + category */}
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-stone-800 dark:text-stone-200 truncate">
            {item.name}
          </p>
          <p className="text-xs text-stone-400 dark:text-stone-500 truncate">
            {categories.find((c) => c.id === item.category_id)?.name ?? "—"}
          </p>
        </div>

        {/* Variant count */}
        <Badge variant="default" className="shrink-0">
          {item.variants.length} variant
          {item.variants.length !== 1 ? "s" : ""}
        </Badge>

        {/* Actions */}
        <div className="flex items-center gap-1 shrink-0">
          <button
            onClick={() => {
              setIsEditing((e) => !e);
              setExpanded(true);
            }}
            className="w-7 h-7 flex items-center justify-center rounded-lg text-stone-400 hover:text-amber-700 hover:bg-amber-50 dark:hover:bg-amber-950/30 transition-colors"
          >
            <Pencil size={13} />
          </button>
          <button
            onClick={() => deleteMutation.mutate()}
            disabled={deleteMutation.isPending}
            className="w-7 h-7 flex items-center justify-center rounded-lg text-stone-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-950/30 transition-colors disabled:opacity-40"
          >
            <Trash2 size={13} />
          </button>
          <button
            onClick={() => setExpanded((e) => !e)}
            className="w-7 h-7 flex items-center justify-center rounded-lg text-stone-400 hover:text-stone-700 dark:hover:text-stone-300 hover:bg-stone-100 dark:hover:bg-stone-800 transition-colors"
          >
            {expanded ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
          </button>
        </div>
      </div>

      {/* Expanded section */}
      <AnimatePresence>
        {expanded && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.2 }}
            className="overflow-hidden border-t border-stone-100 dark:border-stone-800 bg-stone-50 dark:bg-stone-950/50 px-4 py-3 space-y-4"
          >
            {/* Edit form */}
            {isEditing && (
              <div className="space-y-3 pb-3 border-b border-stone-200 dark:border-stone-700">
                <p className="text-xs font-semibold text-stone-500 dark:text-stone-400 uppercase tracking-wide">
                  Edit item
                </p>
                <Input
                  label="Name"
                  value={editName}
                  onChange={(e) => setEditName(e.target.value)}
                />
                <Input
                  label="Description"
                  value={editDescription}
                  onChange={(e) => setEditDescription(e.target.value)}
                />
                <div className="flex flex-col gap-1.5">
                  <label className="text-sm font-medium text-stone-700 dark:text-stone-300">
                    Category
                  </label>
                  <select
                    value={editCategoryId}
                    onChange={(e) => setEditCategoryId(e.target.value)}
                    className="h-11 rounded-xl border border-stone-300 dark:border-stone-700 bg-white dark:bg-stone-900 text-stone-900 dark:text-stone-100 px-4 text-sm focus:outline-none focus:ring-2 focus:ring-amber-500"
                  >
                    {categories.map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.name}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="flex gap-2">
                  <Button
                    size="sm"
                    isLoading={updateMutation.isPending}
                    onClick={() => updateMutation.mutate()}
                  >
                    Save changes
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => setIsEditing(false)}
                  >
                    Cancel
                  </Button>
                </div>
              </div>
            )}

            {/* Variants */}
            <div>
              <p className="text-xs font-semibold text-stone-500 dark:text-stone-400 uppercase tracking-wide mb-2">
                Variants
              </p>
              {item.variants.length === 0 ? (
                <p className="text-xs text-stone-400 dark:text-stone-500 mb-2">
                  No variants yet — add one below
                </p>
              ) : (
                <div className="divide-y divide-stone-100 dark:divide-stone-800">
                  {item.variants.map((v) => (
                    <VariantRow
                      key={v.id}
                      variant={v}
                      itemId={item.id}
                      onDeleted={() => {}}
                    />
                  ))}
                </div>
              )}
              <AddVariantForm itemId={item.id} onAdded={() => {}} />
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}

// ── Add item form ──────────────────────────────────────────────────────────────

function AddItemForm({
  categories,
  onAdded,
}: {
  categories: Category[];
  onAdded: () => void;
}) {
  const queryClient = useQueryClient();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [categoryId, setCategoryId] = useState(categories[0]?.id ?? "");
  const [imageUrl, setImageUrl] = useState("");

  const addMutation = useMutation({
    mutationFn: () =>
      adminService.createItem({
        name,
        description,
        // Select inputs yield strings; the backend expects a numeric id
        category_id: Number(categoryId),
        image_url: imageUrl,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-items"] });
      setName("");
      setDescription("");
      setImageUrl("");
      toast.success("Item created");
      onAdded();
    },
    onError: () => toast.error("Could not create item"),
  });

  const canAdd =
    name.trim().length > 0 && String(categoryId).length > 0;

  return (
    <Card>
      <CardHeader>
        <p className="text-sm font-semibold text-stone-700 dark:text-stone-300">
          Add new item
        </p>
      </CardHeader>
      <CardBody className="space-y-3">
        <Input
          label="Name"
          placeholder="e.g. Iced Latte"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <Input
          label="Description"
          placeholder="Short description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
        <Input
          label="Image URL (optional)"
          placeholder="https://..."
          value={imageUrl}
          onChange={(e) => setImageUrl(e.target.value)}
        />
        <div className="flex flex-col gap-1.5">
          <label className="text-sm font-medium text-stone-700 dark:text-stone-300">
            Category
          </label>
          <select
            value={categoryId}
            onChange={(e) => setCategoryId(e.target.value)}
            className="h-11 rounded-xl border border-stone-300 dark:border-stone-700 bg-white dark:bg-stone-900 text-stone-900 dark:text-stone-100 px-4 text-sm focus:outline-none focus:ring-2 focus:ring-amber-500"
          >
            {categories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
        </div>
        <Button
          fullWidth
          disabled={!canAdd}
          isLoading={addMutation.isPending}
          onClick={() => addMutation.mutate()}
        >
          <Plus size={15} className="mr-1.5" />
          Add item
        </Button>
      </CardBody>
    </Card>
  );
}

// ── Main menu management page ──────────────────────────────────────────────────

function AdminMenuContent() {
  const queryClient = useQueryClient();
  const [showAddItem, setShowAddItem] = useState(false);

  const { data: categories = [], isLoading: categoriesLoading } = useQuery({
    queryKey: ["admin-categories"],
    queryFn: adminService.getCategories,
  });

  const { data: items = [], isLoading: itemsLoading } = useQuery({
    queryKey: ["admin-items"],
    queryFn: adminService.getItems,
  });

  const isLoading = categoriesLoading || itemsLoading;

  // Group items by category
  const itemsByCategory = categories.map((cat) => ({
    category: cat,
    items: items.filter((i) => i.category_id === cat.id),
  }));

  return (
    <div className="space-y-6 pb-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold text-stone-900 dark:text-stone-100">
            Menu Management
          </h1>
          <p className="text-sm text-stone-500 dark:text-stone-400">
            {items.length} items across {categories.length} categories
          </p>
        </div>
        <Button
          size="sm"
          onClick={() => setShowAddItem((s) => !s)}
          variant={showAddItem ? "secondary" : "primary"}
        >
          {showAddItem ? (
            <>
              <X size={14} className="mr-1.5" />
              Cancel
            </>
          ) : (
            <>
              <Plus size={14} className="mr-1.5" />
              Add item
            </>
          )}
        </Button>
      </div>

      {/* Add item form */}
      <AnimatePresence>
        {showAddItem && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "auto" }}
            exit={{ opacity: 0, height: 0 }}
            transition={{ duration: 0.2 }}
            className="overflow-hidden"
          >
            <AddItemForm
              categories={categories}
              onAdded={() => setShowAddItem(false)}
            />
          </motion.div>
        )}
      </AnimatePresence>

      {/* Items grouped by category */}
      {isLoading ? (
        <div className="space-y-6">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="space-y-3">
              <Skeleton className="h-5 w-32" />
              <Skeleton className="h-14 w-full rounded-xl" />
              <Skeleton className="h-14 w-full rounded-xl" />
            </div>
          ))}
        </div>
      ) : (
        <div className="space-y-6">
          {itemsByCategory.map(({ category, items: catItems }) => (
            <div key={category.id} className="space-y-2">
              <div className="flex items-center gap-2">
                <p className="text-sm font-semibold text-stone-600 dark:text-stone-400 uppercase tracking-wide">
                  {category.name}
                </p>
                <span className="text-xs text-stone-400 dark:text-stone-500">
                  ({catItems.length})
                </span>
              </div>

              {catItems.length === 0 ? (
                <p className="text-xs text-stone-400 dark:text-stone-500 py-2 pl-2">
                  No items in this category
                </p>
              ) : (
                <div className="space-y-2">
                  <AnimatePresence mode="popLayout">
                    {catItems.map((item) => (
                      <motion.div
                        key={item.id}
                        layout
                        initial={{ opacity: 0, y: 8 }}
                        animate={{ opacity: 1, y: 0 }}
                        exit={{ opacity: 0, height: 0 }}
                        transition={{ duration: 0.2 }}
                      >
                        <ItemRow item={item} categories={categories} />
                      </motion.div>
                    ))}
                  </AnimatePresence>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default function AdminMenuPage() {
  return (
    <ProtectedRoute requiredRole="admin">
      <AdminMenuContent />
    </ProtectedRoute>
  );
}