// src/utils/money.ts

// ─── Core rule: money is ALWAYS stored as integer pesewas ─────────────────────
// The API sends and receives pesewas. We only convert at the display layer.
// Never do arithmetic on the formatted string — always work with pesewas integers.

// Converts pesewas to a display string
// 2800 → "GHS 28.00"
export function formatMoney(pesewas: number): string {
  const cedis = pesewas / 100;
  return `GHS ${cedis.toFixed(2)}`;
}

// Converts pesewas to a plain number in cedis (for inputs)
// 2800 → 28.00
export function pesewasToFloat(pesewas: number): number {
  return pesewas / 100;
}

// Converts a cedi float from a user input back to pesewas (for API calls)
// 28.00 → 2800
// Math.round handles floating point imprecision (28.005 * 100 = 2800.4999...)
export function floatToPesewas(cedis: number): number {
  return Math.round(cedis * 100);
}

// Returns just the numeric part for display alongside a separate "GHS" label
// 2800 → "28.00"
export function formatMoneyValue(pesewas: number): string {
  return (pesewas / 100).toFixed(2);
}

// Given a list of variants, returns the lowest price formatted as "from GHS X"
// Used on item cards in the menu grid
export function formatFromPrice(pricePesewas: number[]): string {
  if (pricePesewas.length === 0) return "";
  const min = Math.min(...pricePesewas);
  return `from ${formatMoney(min)}`;
}