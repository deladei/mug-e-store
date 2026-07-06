// src/services/admin.service.ts

import api from "./api";
import {
  Category,
  MenuItem,
  CreateCategoryPayload,
  UpdateCategoryPayload,
  CreateItemPayload,
  UpdateItemPayload,
  CreateVariantPayload,
  UpdateAvailabilityPayload,
} from "@/types";

export const adminService = {
  // ── Categories ─────────────────────────────────────────────────────────────
  getCategories: async (): Promise<Category[]> => {
    const { data } = await api.get<Category[]>("/categories");
    return data;
  },

  createCategory: async (
    payload: CreateCategoryPayload
  ): Promise<Category> => {
    const { data } = await api.post<Category>(
      "/admin/categories",
      payload
    );
    return data;
  },

  updateCategory: async (
    id: string,
    payload: UpdateCategoryPayload
  ): Promise<Category> => {
    const { data } = await api.patch<Category>(
      `/admin/categories/${id}`,
      payload
    );
    return data;
  },

  deleteCategory: async (id: string): Promise<void> => {
    await api.delete(`/admin/categories/${id}`);
  },

  // ── Items ──────────────────────────────────────────────────────────────────
  getItems: async (): Promise<MenuItem[]> => {
    // There is no GET /admin/items route — the public /items endpoint
    // returns unavailable items too when called with a staff/admin token.
    const { data } = await api.get<MenuItem[]>("/items");
    return data;
  },

  createItem: async (payload: CreateItemPayload): Promise<MenuItem> => {
    const { data } = await api.post<MenuItem>("/admin/items", payload);
    return data;
  },

  updateItem: async (
    id: string,
    payload: UpdateItemPayload
  ): Promise<MenuItem> => {
    const { data } = await api.patch<MenuItem>(
      `/admin/items/${id}`,
      payload
    );
    return data;
  },

  deleteItem: async (id: string): Promise<void> => {
    await api.delete(`/admin/items/${id}`);
  },

  updateAvailability: async (
    id: string,
    payload: UpdateAvailabilityPayload
  ): Promise<MenuItem> => {
    const { data } = await api.patch<MenuItem>(
      `/admin/items/${id}/availability`,
      payload
    );
    return data;
  },

  // ── Variants ───────────────────────────────────────────────────────────────
  createVariant: async (
    itemId: string,
    payload: CreateVariantPayload
  ): Promise<MenuItem> => {
    const { data } = await api.post<MenuItem>(
      `/admin/items/${itemId}/variants`,
      payload
    );
    return data;
  },

  deleteVariant: async (variantId: string): Promise<void> => {
    await api.delete(`/admin/variants/${variantId}`);
  },
};