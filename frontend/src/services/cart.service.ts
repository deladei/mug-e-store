// src/services/cart.service.ts

import api from "./api";
import { Cart, AddToCartPayload, UpdateCartLinePayload } from "@/types";

export const cartService = {
  getCart: async (): Promise<Cart> => {
    const { data } = await api.get<Cart>("/cart");
    return data;
  },

  addItem: async (payload: AddToCartPayload): Promise<Cart> => {
    const { data } = await api.post<Cart>("/cart/items", payload);
    return data;
  },

  updateLine: async (
    lineId: string,
    payload: UpdateCartLinePayload
  ): Promise<Cart> => {
    const { data } = await api.patch<Cart>(`/cart/items/${lineId}`, payload);
    return data;
  },

  removeLine: async (lineId: string): Promise<Cart> => {
    const { data } = await api.delete<Cart>(`/cart/items/${lineId}`);
    return data;
  },
};