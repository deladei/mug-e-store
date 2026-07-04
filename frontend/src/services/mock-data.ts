// services/mock-data.ts

import {
  AuthResponse,
  Cart,
  Category,
  MenuItem,
  Order,
  OrderHistoryResponse,
  LoyaltyBalance,
  ReportsSummary,
} from "@/types";

export const MOCK_USER: AuthResponse = {
  access_token: "mock-access-token-abc123",
  user: {
    id: "user-1",
    name: "Kofi Mensah",
    email: "kofi@example.com",
    phone: "0244000001",
    role: "customer",
    created_at: "2026-01-01T00:00:00Z",
  },
};

export const MOCK_ADMIN_USER: AuthResponse = {
  access_token: "mock-access-token-admin",
  user: {
    id: "admin-1",
    name: "Ama Owusu",
    email: "ama@coffeemug.com",
    phone: "0244000002",
    role: "admin",
    created_at: "2026-01-01T00:00:00Z",
  },
};

export const MOCK_CATEGORIES: Category[] = [
  { id: "cat-1", name: "Espresso Drinks", sort_order: 1 },
  { id: "cat-2", name: "Brewed Coffee", sort_order: 2 },
  { id: "cat-3", name: "Pastries", sort_order: 3 },
];

export const MOCK_ITEMS: MenuItem[] = [
  {
    id: "item-1",
    category_id: "cat-1",
    name: "Cappuccino",
    description: "Rich espresso with velvety steamed milk and a thick layer of foam.",
    image_url: "",
    is_available: true,
    variants: [
      { id: "var-1a", name: "Small", price_pesewas: 2200, sort_order: 1 },
      { id: "var-1b", name: "Medium", price_pesewas: 2800, sort_order: 2 },
      { id: "var-1c", name: "Large", price_pesewas: 3400, sort_order: 3 },
    ],
  },
  {
    id: "item-2",
    category_id: "cat-1",
    name: "Espresso",
    description: "A concentrated shot of pure coffee intensity.",
    image_url: "",
    is_available: true,
    variants: [
      { id: "var-2a", name: "Single", price_pesewas: 1500, sort_order: 1 },
      { id: "var-2b", name: "Double", price_pesewas: 2200, sort_order: 2 },
    ],
  },
  {
    id: "item-3",
    category_id: "cat-1",
    name: "Latte",
    description: "Smooth espresso with more steamed milk and light foam.",
    image_url: "",
    is_available: true,
    variants: [
      { id: "var-3a", name: "Small", price_pesewas: 2500, sort_order: 1 },
      { id: "var-3b", name: "Medium", price_pesewas: 3000, sort_order: 2 },
      { id: "var-3c", name: "Large", price_pesewas: 3600, sort_order: 3 },
    ],
  },
  {
    id: "item-4",
    category_id: "cat-2",
    name: "Filter Coffee",
    description: "Slow-brewed single origin, clean and balanced.",
    image_url: "",
    is_available: true,
    variants: [
      { id: "var-4a", name: "Regular", price_pesewas: 1800, sort_order: 1 },
      { id: "var-4b", name: "Large", price_pesewas: 2400, sort_order: 2 },
    ],
  },
  {
    id: "item-5",
    category_id: "cat-3",
    name: "Butter Croissant",
    description: "Flaky, golden and buttery — baked fresh each morning.",
    image_url: "",
    is_available: true,
    variants: [
      { id: "var-5a", name: "Regular", price_pesewas: 1200, sort_order: 1 },
    ],
  },
  {
    id: "item-6",
    category_id: "cat-3",
    name: "Chocolate Muffin",
    description: "Dense, moist, and packed with dark chocolate chips.",
    image_url: "",
    is_available: false,
    variants: [
      { id: "var-6a", name: "Regular", price_pesewas: 1400, sort_order: 1 },
    ],
  },
];

export const MOCK_CART: Cart = {
  lines: [
    {
      line_id: "line-1",
      item_variant_id: "var-1b",
      item_name: "Cappuccino",
      variant_name: "Medium",
      unit_price_pesewas: 2800,
      quantity: 2,
      available: true,
    },
    {
      line_id: "line-2",
      item_variant_id: "var-5a",
      item_name: "Butter Croissant",
      variant_name: "Regular",
      unit_price_pesewas: 1200,
      quantity: 1,
      available: true,
    },
  ],
  subtotal_pesewas: 6800,
};

export const MOCK_ORDER: Order = {
  id: "order-abc123",
  status: "preparing",
  fulfilment: "pickup",
  lines: [
    {
      item_name: "Cappuccino",
      variant_name: "Medium",
      unit_price_pesewas: 2800,
      quantity: 2,
    },
    {
      item_name: "Butter Croissant",
      variant_name: "Regular",
      unit_price_pesewas: 1200,
      quantity: 1,
    },
  ],
  subtotal_pesewas: 6800,
  delivery_fee_pesewas: 0,
  discount_pesewas: 0,
  total_pesewas: 6800,
  created_at: "2026-06-29T09:00:00Z",
};

export const MOCK_ORDER_HISTORY: OrderHistoryResponse = {
  orders: [
    { ...MOCK_ORDER, id: "order-abc123", status: "completed" },
    {
      ...MOCK_ORDER,
      id: "order-def456",
      status: "completed",
      fulfilment: "delivery",
      delivery_fee_pesewas: 500,
      total_pesewas: 7300,
      created_at: "2026-06-28T14:30:00Z",
    },
  ],
  page: 1,
};

export const MOCK_LOYALTY: LoyaltyBalance = {
  balance: 340,
  ledger: [
    {
      order_id: "order-abc123",
      delta: 68,
      reason: "earn_on_completion",
      created_at: "2026-06-29T09:30:00Z",
    },
    {
      order_id: "order-def456",
      delta: -50,
      reason: "redeem_at_checkout",
      created_at: "2026-06-28T14:00:00Z",
    },
  ],
};

export const MOCK_REPORTS: ReportsSummary = {
  from: "2026-05-30",
  to: "2026-06-29",
  totals: {
    orders: 87,
    paid_orders: 79,
    revenue_pesewas: 198500,
  },
  daily: Array.from({ length: 30 }, (_, i) => {
    const date = new Date("2026-05-30");
    date.setDate(date.getDate() + i);
    return {
      date: date.toISOString().split("T")[0],
      orders: Math.floor(Math.random() * 8) + 1,
      paid_orders: Math.floor(Math.random() * 7) + 1,
      revenue_pesewas: Math.floor(Math.random() * 15000) + 3000,
    };
  }),
};