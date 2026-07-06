// src/services/orders.service.ts

import api from "./api";
import {
  Order,
  CheckoutPayload,
  CheckoutResponse,
  OrderHistoryResponse,
  OrderHistoryEntry,
} from "@/types";

export const ordersService = {
  checkout: async (payload: CheckoutPayload): Promise<CheckoutResponse> => {
    const { data } = await api.post<CheckoutResponse>("/checkout", payload);
    return data;
  },

  getOrder: async (id: string): Promise<Order> => {
    const { data } = await api.get<Order>(`/orders/${id}`);
    return data;
  },

  getOrderHistory: async (page = 1): Promise<OrderHistoryResponse> => {
    const { data } = await api.get<OrderHistoryResponse>("/me/orders", {
      params: { page },
    });
    return data;
  },

  getOrderHistory2: async (id: string): Promise<OrderHistoryEntry[]> => {
    // Backend wraps the timeline: { history: [...] }
    const { data } = await api.get<{ history: OrderHistoryEntry[] }>(
      `/admin/orders/${id}/history`
    );
    return data.history ?? [];
  },
};