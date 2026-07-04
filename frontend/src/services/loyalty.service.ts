// src/services/loyalty.service.ts

import api from "./api";
import { LoyaltyBalance } from "@/types";

export const loyaltyService = {
  getBalance: async (): Promise<LoyaltyBalance> => {
    const { data } = await api.get<LoyaltyBalance>("/me/loyalty");
    return data;
  },
};