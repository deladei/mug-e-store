// src/services/menu.service.ts

import api from "./api";
import { Category, MenuItem } from "@/types";

export const menuService = {
  getCategories: async (): Promise<Category[]> => {
    const { data } = await api.get<Category[]>("/categories");
    return data;
  },

  getItems: async (categoryId?: string): Promise<MenuItem[]> => {
    const { data } = await api.get<MenuItem[]>("/items", {
      params: categoryId ? { category: categoryId } : undefined,
    });
    return data;
  },

  getItem: async (id: string): Promise<MenuItem> => {
    const { data } = await api.get<MenuItem>(`/items/${id}`);
    return data;
  },
};