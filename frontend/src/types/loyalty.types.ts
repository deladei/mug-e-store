// types/loyalty.types.ts

export type LedgerReason =
  | "earn_on_completion"
  | "redeem_at_checkout"
  | "refund_on_cancel";

export interface LedgerEntry {
  order_id: string;
  delta: number;       // positive = earned/refunded, negative = redeemed
  reason: LedgerReason;
  created_at: string;
}

export interface LoyaltyBalance {
  balance: number;
  ledger: LedgerEntry[];
}