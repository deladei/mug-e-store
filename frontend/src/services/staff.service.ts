// src/services/staff.service.ts

import api from "./api";
import { Order, OrderStatus, OrderHistoryEntry } from "@/types";

export const staffService = {
  getOrders: async (status?: OrderStatus): Promise<Order[]> => {
    const { data } = await api.get<{ orders: Order[] }>("/admin/orders", {
      params: status ? { status } : undefined,
    });
    return data.orders;
  },

  getOrder: async (id: string): Promise<Order> => {
    const { data } = await api.get<Order>(`/orders/${id}`);
    return data;
  },

  getOrderHistory: async (id: string): Promise<OrderHistoryEntry[]> => {
    // Backend wraps the timeline: { history: [...] }
    const { data } = await api.get<{ history: OrderHistoryEntry[] }>(
      `/admin/orders/${id}/history`
    );
    return data.history ?? [];
  },

  transitionOrder: async (
    id: string,
    to: OrderStatus
  ): Promise<Order> => {
    const { data } = await api.post<Order>(
      `/admin/orders/${id}/transition`,
      { to }
    );
    return data;
  },

  updateAvailability: async (
    itemId: string,
    isAvailable: boolean
  ): Promise<void> => {
    await api.patch(`/admin/items/${itemId}/availability`, {
      is_available: isAvailable,
    });
  },
};