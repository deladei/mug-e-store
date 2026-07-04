// types/cart.types.ts

export interface CartLine {
  line_id: string;
  item_variant_id: string;
  item_name: string;
  variant_name: string;
  unit_price_pesewas: number;
  quantity: number;
  available: boolean;
}

export interface Cart {
  lines: CartLine[];
  subtotal_pesewas: number;
}

export interface AddToCartPayload {
  item_variant_id: string;
  quantity: number;
}

export interface UpdateCartLinePayload {
  quantity: number;
}