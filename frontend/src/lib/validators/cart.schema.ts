// src/lib/validators/cart.schema.ts

import { z } from "zod";

export const addToCartSchema = z.object({
  item_variant_id: z.string().min(1, "Please select a size"),
  quantity: z.number().int().min(1).max(20),
});

export type AddToCartFormValues = z.infer<typeof addToCartSchema>;