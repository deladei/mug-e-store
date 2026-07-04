// constants/routes.ts

export const ROUTES = {
  // Customer
  HOME: "/",
  ITEM: (id: string) => `/items/${id}`,
  CART: "/cart",
  CHECKOUT: "/checkout",
  ORDER: (id: string) => `/orders/${id}`,
  ORDER_HISTORY: "/orders",
  PROFILE: "/profile",
  AUTH: "/auth",
  RESET_PASSWORD: "/reset-password",

  // Staff
  STAFF_QUEUE: "/staff",
  STAFF_ORDER: (id: string) => `/staff/orders/${id}`,

  // Admin
  ADMIN_MENU: "/admin/menu",
  ADMIN_REPORTS: "/admin/reports",
} as const;