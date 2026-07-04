// src/features/orders/useLiveOrder.ts

"use client";

import { useState, useEffect, useRef } from "react";
import { Order, OrderStatus } from "@/types";
import { ordersService } from "@/services/orders.service";
import { tokenStore } from "@/services/api";
import { MOCK_MODE } from "@/services/api";
import { isTerminalStatus } from "@/utils";

interface UseLiveOrderResult {
  order: Order | null;
  isLoading: boolean;
  error: string | null;
}

export function useLiveOrder(orderId: string): UseLiveOrderResult {
  const [order, setOrder] = useState<Order | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);
  const pollIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // ── Polling fallback ─────────────────────────────────────────────────────
  const startPolling = (currentOrder: Order | null) => {
    if (currentOrder && isTerminalStatus(currentOrder.status)) return;

    pollIntervalRef.current = setInterval(async () => {
      try {
        const data = await ordersService.getOrder(orderId);
        setOrder(data);
        if (isTerminalStatus(data.status)) {
          stopPolling();
        }
      } catch {
        // Silently continue polling on transient errors
      }
    }, 4000);
  };

  const stopPolling = () => {
    if (pollIntervalRef.current) {
      clearInterval(pollIntervalRef.current);
      pollIntervalRef.current = null;
    }
  };

  // ── SSE connection ───────────────────────────────────────────────────────
  const startSSE = () => {
    const token = tokenStore.get();
    if (!token) return false;

    const url = `${process.env.NEXT_PUBLIC_API_URL}/orders/${orderId}/events?token=${token}`;

    try {
      const es = new EventSource(url);
      eventSourceRef.current = es;

      es.addEventListener("status", (e: MessageEvent) => {
        const payload = JSON.parse(e.data) as {
          order_id: string;
          status: OrderStatus;
        };
        setOrder((prev) =>
          prev ? { ...prev, status: payload.status } : null
        );
        if (isTerminalStatus(payload.status)) {
          es.close();
        }
      });

      es.onerror = () => {
        es.close();
        eventSourceRef.current = null;
        // Try to refresh token then reconnect once
        startPolling(order);
      };

      return true;
    } catch {
      return false;
    }
  };

  useEffect(() => {
    let cancelled = false;

    const init = async () => {
      setIsLoading(true);
      setError(null);

      try {
        // Always fetch the order first for the initial snapshot
        const data = await ordersService.getOrder(orderId);
        if (cancelled) return;
        setOrder(data);
        setIsLoading(false);

        if (isTerminalStatus(data.status)) return;

        // In mock mode, use polling only — SSE needs a real server
        if (MOCK_MODE) {
          startPolling(data);
          return;
        }

        // Try SSE first, fall back to polling if it fails
        const sseStarted = startSSE();
        if (!sseStarted) {
          startPolling(data);
        }
      } catch {
        if (cancelled) return;
        setError("Could not load order. Please refresh the page.");
        setIsLoading(false);
      }
    };

    init();

    return () => {
      cancelled = true;
      eventSourceRef.current?.close();
      stopPolling();
    };
  }, [orderId]);

  return { order, isLoading, error };
}