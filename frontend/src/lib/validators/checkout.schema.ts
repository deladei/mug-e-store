// src/lib/validators/checkout.schema.ts

import { z } from "zod";

export const checkoutSchema = z
  .object({
    fulfilment: z.enum(["pickup", "delivery"]),
    address: z.string().optional(),
    phone: z.string().optional(),
    points_to_redeem: z.number().int().min(0).optional(),
  })
  .superRefine((data, ctx) => {
    if (data.fulfilment === "delivery") {
      if (!data.address || data.address.trim().length < 5) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ["address"],
          message: "Please enter a delivery address",
        });
      }
      if (!data.phone || data.phone.trim().length < 10) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ["phone"],
          message: "Please enter a valid phone number",
        });
      }
    }
  });

export type CheckoutFormValues = z.infer<typeof checkoutSchema>;