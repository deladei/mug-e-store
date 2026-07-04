// types/order.types.ts

// Every possible status in the system — do not add to this list
export type OrderStatus =
  | "pending_payment"
  | "paid"
  | "preparing"
  | "ready"
  | "out_for_delivery"
  | "completed"
  | "cancelled";

export type Fulfilment = "pickup" | "delivery";

export interface OrderLine {
  item_name: string;
  variant_name: string;
  unit_price_pesewas: number;
  quantity: number;
}

export interface Order {
  id: string;
  status: OrderStatus;
  fulfilment: Fulfilment;
  address?: string;
  phone?: string;
  lines: OrderLine[];
  subtotal_pesewas: number;
  delivery_fee_pesewas: number;
  discount_pesewas: number;
  total_pesewas: number;
  created_at: string;
}

// What POST /checkout returns
export interface CheckoutResponse {
  order: Order;
  authorization_url: string;
}

export interface CheckoutPayload {
  fulfilment: Fulfilment;
  address?: string;
  phone?: string;
  idempotency_key: string;
  points_to_redeem?: number;
}

// SSE event payload from GET /orders/{id}/events
export interface OrderStatusEvent {
  order_id: string;
  status: OrderStatus;
}

// Staff: order history timeline entry
export interface OrderHistoryEntry {
  from_status: OrderStatus | null;
  to_status: OrderStatus;
  actor: string | null; // null means the payment webhook (system)
  created_at: string;
}

export interface OrderTransitionPayload {
  to: OrderStatus;
}

// Paginated order history for GET /me/orders
export interface OrderHistoryResponse {
  orders: Order[];
  page: number;
}