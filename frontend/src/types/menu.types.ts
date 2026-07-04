// types/menu.types.ts

export interface Category {
  id: string;
  name: string;
  sort_order: number;
}

export interface Variant {
  id: string;
  name: string;
  price_pesewas: number;
  sort_order: number;
}

export interface MenuItem {
  id: string;
  category_id: string;
  name: string;
  description: string;
  image_url: string;
  is_available: boolean;
  variants: Variant[];
}