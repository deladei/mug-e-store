// types/admin.types.ts

// ─── Category management ─────────────────────────────────────────────────────

export interface CreateCategoryPayload {
  name: string;
  sort_order: number;
}

export interface UpdateCategoryPayload {
  name?: string;
  sort_order?: number;
}

// ─── Item management ─────────────────────────────────────────────────────────

export interface CreateItemPayload {
  category_id: string;
  name: string;
  description: string;
  image_url: string;
}

export interface UpdateItemPayload {
  category_id?: string;
  name?: string;
  description?: string;
  image_url?: string;
}

export interface UpdateAvailabilityPayload {
  is_available: boolean;
}

// ─── Variant management ───────────────────────────────────────────────────────

export interface CreateVariantPayload {
  name: string;
  price_pesewas: number;
  sort_order: number;
}

// ─── Reports ─────────────────────────────────────────────────────────────────

export interface DailyReport {
  date: string;
  orders: number;
  paid_orders: number;
  revenue_pesewas: number;
}

export interface ReportsSummary {
  from: string;
  to: string;
  totals: {
    orders: number;
    paid_orders: number;
    revenue_pesewas: number;
  };
  daily: DailyReport[];
}