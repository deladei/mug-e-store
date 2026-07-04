// services/api.ts

import axios, {
  AxiosInstance,
  AxiosRequestConfig,
  AxiosResponse,
  InternalAxiosRequestConfig,
} from "axios";
import {
  MOCK_CATEGORIES,
  MOCK_ITEMS,
  MOCK_CART,
  MOCK_ORDER,
  MOCK_ORDER_HISTORY,
  MOCK_LOYALTY,
  MOCK_REPORTS,
  MOCK_USER,
  MOCK_ADMIN_USER,
} from "./mock-data";
import { Category, MenuItem } from "@/types";

// ─── Mock mode flag ───────────────────────────────────────────────────────────
// Set to true when no backend is available.
// Set to false when your teammates' backend is ready.
// This is the ONLY line you change to switch between mock and real.
export const MOCK_MODE = true;

// ─── Token store ─────────────────────────────────────────────────────────────
// The access token lives here — in a module-level variable, NOT localStorage.
// This is intentional: localStorage is readable by any script on the page (XSS risk).
// A module variable is wiped on page refresh, which is fine — the refresh cookie
// will silently restore the session on next load.
let accessToken: string | null = null;

export const tokenStore = {
  get: () => accessToken,
  set: (token: string) => { accessToken = token; },
  clear: () => { accessToken = null; },
};

// ─── Axios instance ───────────────────────────────────────────────────────────
const api: AxiosInstance = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL,
  withCredentials: true, // sends the httpOnly refresh cookie automatically
  headers: {
    "Content-Type": "application/json",
  },
});

// ─── Request interceptor ──────────────────────────────────────────────────────
// Runs before every request leaves the browser.
// If we have an access token in memory, attach it as a Bearer header.
// Components never call api.get(..., { headers: { Authorization: ... } }) manually.
api.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = tokenStore.get();
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// ─── Response interceptor ─────────────────────────────────────────────────────
// Runs after every response comes back.
// If a request fails with 401, we attempt a silent token refresh ONCE,
// then retry the original request with the new token.
// If the refresh itself fails, the session is over — redirect to login.

let isRefreshing = false;
// Queue of requests that arrived while a refresh was already in flight.
// They wait for the refresh to complete, then all retry at once.
let refreshQueue: Array<{
  resolve: (token: string) => void;
  reject: (error: unknown) => void;
}> = [];

const processQueue = (error: unknown, token: string | null) => {
  refreshQueue.forEach(({ resolve, reject }) => {
    if (error) reject(error);
    else if (token) resolve(token);
  });
  refreshQueue = [];
};

api.interceptors.response.use(
  (response: AxiosResponse) => response,
  async (error) => {
    const originalRequest = error.config as AxiosRequestConfig & {
      _retry?: boolean;
    };

    // Only attempt refresh on 401, and only once per request
    if (error.response?.status === 401 && !originalRequest._retry) {
      if (isRefreshing) {
        // Another refresh is already in flight — queue this request
        return new Promise((resolve, reject) => {
          refreshQueue.push({ resolve, reject });
        }).then((token) => {
          if (originalRequest.headers) {
            (originalRequest.headers as Record<string, string>).Authorization =
              `Bearer ${token}`;
          }
          return api(originalRequest);
        });
      }

      originalRequest._retry = true;
      isRefreshing = true;

      try {
        // The refresh cookie travels automatically (withCredentials: true)
        const { data } = await api.post<{ access_token: string }>(
          "/auth/refresh"
        );
        const newToken = data.access_token;
        tokenStore.set(newToken);
        processQueue(null, newToken);

        // Retry the original request with the new token
        if (originalRequest.headers) {
          (originalRequest.headers as Record<string, string>).Authorization =
            `Bearer ${newToken}`;
        }
        return api(originalRequest);
      } catch (refreshError) {
        // Refresh failed — session is truly over
        processQueue(refreshError, null);
        tokenStore.clear();
        // Redirect to login — works in both App Router and Pages Router
        if (typeof window !== "undefined") {
          window.location.href = "/auth";
        }
        return Promise.reject(refreshError);
      } finally {
        isRefreshing = false;
      }
    }

    return Promise.reject(error);
  }
);

// ─── Mock interceptor ─────────────────────────────────────────────────────────
// When MOCK_MODE is true, this intercepts every request before it hits the network
// and returns fake data based on the URL and method.
// The shape of every mock response matches the real API contract exactly.

if (MOCK_MODE) {
  api.interceptors.request.use(async (config) => {
    // Small artificial delay so loading states are visible during development
    await new Promise((r) => setTimeout(r, 400));

    const url = config.url ?? "";
    const method = (config.method ?? "get").toLowerCase();

    // Helper: throw this to simulate a mock API error response
    const mockError = (status: number, code: string, message: string) => {
      const error = new Error(message) as Error & { response: unknown };
      error.response = { status, data: { error: { code, message } } };
      throw error;
    };

    let data: unknown = null;

    // ── Auth ──────────────────────────────────────────────────────────────────
    if (url.includes("/auth/login") && method === "post") {
      const body = typeof config.data === "string" ? JSON.parse(config.data) : (config.data ?? {});
      if (body.email === "ama@coffeemug.com") {
        data = MOCK_ADMIN_USER;
      } else if (body.password === "wrongpassword") {
        mockError(401, "invalid_credentials", "Invalid email or password.");
      } else {
        data = MOCK_USER;
      }
    } else if (url.includes("/auth/register") && method === "post") {
      data = MOCK_USER;
    } else if (url.includes("/auth/refresh") && method === "post") {
      data = {
        access_token: "mock-refreshed-token-admin",
        user: MOCK_ADMIN_USER.user,
      };
    } else if (url.includes("/auth/logout") && method === "post") {
      data = {};
    } else if (url.includes("/auth/password-reset/request") && method === "post") {
      data = { message: "If that address has an account, we've sent a link." };
    } else if (url.includes("/auth/password-reset/confirm") && method === "post") {
      data = { message: "Password reset successfully." };

      // ── Menu ──────────────────────────────────────────────────────────────────
    } else if (url === "/categories" && method === "get") {
      data = MOCK_CATEGORIES;
    } else if ((url === "/items" || url.match(/^\/items\/[^/]+$/)) && method === "get") {
  const categoryId = (config.params as Record<string, string>)?.category;
  data = url === "/items"
    ? categoryId
      ? MOCK_ITEMS.filter(
          (i) => i.category_id === categoryId && i.is_available
        )
      : MOCK_ITEMS.filter((i) => i.is_available)
    : (() => {
        const id = url.split("/").pop();
        const item = MOCK_ITEMS.find((i) => i.id === id);
        if (!item || !item.is_available) mockError(404, "not_found", "Item not found.");
        return item;
      })();

      // ── Cart ──────────────────────────────────────────────────────────────────
    } else if (url === "/cart" && method === "get") {
      data = MOCK_CART;
    } else if (url === "/cart/items" && method === "post") {
      const body = typeof config.data === "string" ? JSON.parse(config.data) : (config.data ?? {});
      const existingLine = MOCK_CART.lines.find(
        (l) => l.item_variant_id === body.item_variant_id
      );
      if (existingLine) {
        existingLine.quantity += body.quantity ?? 1;
      } else {
        const variant = MOCK_ITEMS.flatMap((i) => i.variants).find(
          (v) => v.id === body.item_variant_id
        );
        if (variant) {
          const item = MOCK_ITEMS.find((i) =>
            i.variants.some((v) => v.id === body.item_variant_id)
          );
          MOCK_CART.lines.push({
            line_id: "line-" + Date.now(),
            item_variant_id: body.item_variant_id,
            item_name: item?.name ?? "Item",
            variant_name: variant.name,
            unit_price_pesewas: variant.price_pesewas,
            quantity: body.quantity ?? 1,
            available: true,
          });
        }
      }
      MOCK_CART.subtotal_pesewas = MOCK_CART.lines.reduce(
        (sum, l) => sum + l.unit_price_pesewas * l.quantity,
        0
      );
      data = { ...MOCK_CART, lines: [...MOCK_CART.lines] };
    } else if (url.match(/^\/cart\/items\/.+/) && method === "patch") {
      const lineId = url.split("/").pop();
      const body = typeof config.data === "string"
        ? JSON.parse(config.data)
        : (config.data ?? {});
      const line = MOCK_CART.lines.find((l) => l.line_id === lineId);
      if (line) line.quantity = body.quantity;
      MOCK_CART.subtotal_pesewas = MOCK_CART.lines.reduce(
        (sum, l) => sum + l.unit_price_pesewas * l.quantity,
        0
      );
      data = { ...MOCK_CART, lines: [...MOCK_CART.lines] };
    } else if (url.match(/^\/cart\/items\/.+/) && method === "delete") {
      const lineId = url.split("/").pop();
      MOCK_CART.lines = MOCK_CART.lines.filter((l) => l.line_id !== lineId);
      MOCK_CART.subtotal_pesewas = MOCK_CART.lines.reduce(
        (sum, l) => sum + l.unit_price_pesewas * l.quantity,
        0
      );
      data = { ...MOCK_CART, lines: [...MOCK_CART.lines] };

      // ── Checkout ──────────────────────────────────────────────────────────────
    } else if (url === "/checkout" && method === "post") {
      const body = typeof config.data === "string" ? JSON.parse(config.data) : (config.data ?? {});
      data = {
        order: {
          ...MOCK_ORDER,
          id: "order-new-" + Date.now(),
          status: "pending_payment",
          fulfilment: body.fulfilment,
          delivery_fee_pesewas: body.fulfilment === "delivery" ? 500 : 0,
          total_pesewas: body.fulfilment === "delivery" ? 7300 : 6800,
        },
        // In mock mode this URL won't work — that's expected
        authorization_url: "https://checkout.paystack.com/mock-payment-page",
      };

      // ── Orders ────────────────────────────────────────────────────────────────
    } else if (url.match(/^\/orders\/[^/]+$/) && method === "get") {
      data = MOCK_ORDER;
    } else if (url.match(/\/orders\/.+\/events/)) {
      // SSE is handled separately — return empty for the mock interceptor
      data = {};
    } else if (url === "/me/orders" && method === "get") {
      data = MOCK_ORDER_HISTORY;

      // ── Profile / Loyalty ─────────────────────────────────────────────────────
    } else if (url === "/me/loyalty" && method === "get") {
      data = MOCK_LOYALTY;

      // ── Admin / Staff ─────────────────────────────────────────────────────────
    } else if (url === "/admin/orders" && method === "get") {
      data = { orders: [MOCK_ORDER] };
    } else if (url.match(/\/admin\/orders\/.+\/history/) && method === "get") {
      data = [
        {
          from_status: null,
          to_status: "paid",
          actor: null,
          created_at: "2026-06-29T09:01:00Z",
        },
        {
          from_status: "paid",
          to_status: "preparing",
          actor: "Ama Owusu",
          created_at: "2026-06-29T09:05:00Z",
        },
      ];
    } else if (url.match(/\/admin\/orders\/.+\/transition/) && method === "post") {
      const body = typeof config.data === "string"
        ? JSON.parse(config.data)
        : (config.data ?? {});
      MOCK_ORDER.status = body.to;
      data = { ...MOCK_ORDER, status: body.to };
    } else if (url.match(/\/admin\/items\/.+\/availability/) && method === "patch") {
      const itemId = url.split("/")[3];
      const body = typeof config.data === "string"
        ? JSON.parse(config.data) : (config.data ?? {});
      const item = MOCK_ITEMS.find((i) => i.id === itemId);
      if (item) {
        item.is_available = body.is_available;
      }
      data = item ?? MOCK_ITEMS[0];
    } else if (url === "/admin/categories" && method === "get") {
      data = MOCK_CATEGORIES;
    } else if (url === "/admin/categories" && method === "post") {
      const body = typeof config.data === "string"
        ? JSON.parse(config.data) : (config.data ?? {});
      const newCat: Category = {
        id: "cat-" + Date.now(),
        name: body.name,
        sort_order: body.sort_order ?? MOCK_CATEGORIES.length + 1,
      };
      MOCK_CATEGORIES.push(newCat);
      data = newCat;
    } else if (url.match(/\/admin\/categories\/.+/) && method === "patch") {
      const id = url.split("/").pop();
      const body = typeof config.data === "string"
        ? JSON.parse(config.data) : (config.data ?? {});
      const cat = MOCK_CATEGORIES.find((c) => c.id === id);
      if (cat) Object.assign(cat, body);
      data = cat;
    } else if (url.match(/\/admin\/categories\/.+/) && method === "delete") {
      const id = url.split("/").pop();
      const idx = MOCK_CATEGORIES.findIndex((c) => c.id === id);
      if (idx > -1) MOCK_CATEGORIES.splice(idx, 1);
      data = {};
    } else if (url === "/admin/items" && method === "get") {
      data = MOCK_ITEMS;
    } else if (url === "/admin/items" && method === "post") {
      const body = typeof config.data === "string"
        ? JSON.parse(config.data) : (config.data ?? {});
      const newItem: MenuItem = {
        id: "item-" + Date.now(),
        category_id: body.category_id,
        name: body.name,
        description: body.description ?? "",
        image_url: body.image_url ?? "",
        is_available: true,
        variants: [],
      };
      MOCK_ITEMS.push(newItem);
      data = newItem;
    } else if (url.match(/^\/admin\/items\/[^/]+$/) && method === "patch") {
      const id = url.split("/").pop();
      const body = typeof config.data === "string"
        ? JSON.parse(config.data) : (config.data ?? {});
      const item = MOCK_ITEMS.find((i) => i.id === id);
      if (item) Object.assign(item, body);
      data = item;
    } else if (url.match(/^\/admin\/items\/[^/]+$/) && method === "delete") {
      const id = url.split("/").pop();
      const idx = MOCK_ITEMS.findIndex((i) => i.id === id);
      if (idx > -1) MOCK_ITEMS.splice(idx, 1);
      data = {};
    } else if (url.match(/\/admin\/items\/.+\/variants/) && method === "post") {
      const itemId = url.split("/")[3];
      const body = typeof config.data === "string"
        ? JSON.parse(config.data) : (config.data ?? {});
      const item = MOCK_ITEMS.find((i) => i.id === itemId);
      if (item) {
        item.variants.push({
          id: "var-" + Date.now(),
          name: body.name,
          price_pesewas: body.price_pesewas,
          sort_order: body.sort_order ?? item.variants.length + 1,
        });
      }
      data = item;
    } else if (url.match(/\/admin\/variants\/.+/) && method === "delete") {
      const variantId = url.split("/").pop();
      MOCK_ITEMS.forEach((item) => {
        item.variants = item.variants.filter((v) => v.id !== variantId);
      });
      data = {};
    } else if (url.match(/^\/admin\/reports\/summary/) && method === "get") {
      data = MOCK_REPORTS;
    } else {
      // Any unmatched route returns empty object rather than crashing
      data = {};
    }

    // Resolve the request with a mock AxiosResponse
    return Promise.reject({
      config,
      response: { status: 200, data, headers: {}, config, statusText: "OK" },
      isMockResolved: true,
    });
  });

  // Second interceptor: catch our mock "rejections" and turn them into resolutions
  api.interceptors.response.use(
    (response) => response,
    (error) => {
      if (error.isMockResolved) {
        return Promise.resolve(error.response);
      }
      return Promise.reject(error);
    }
  );
}

export default api;