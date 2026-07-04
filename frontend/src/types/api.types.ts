// types/api.types.ts

// Every error the backend returns follows this exact shape.
// We branch UI logic on `code`, never on `message` (message is for fallback display only).
export interface ApiError {
  error: {
    code: string;
    message: string;
  };
}

// Known error codes — keeps string literals out of component code
export const API_ERROR_CODES = {
  // Auth
  INVALID_CREDENTIALS: "invalid_credentials",
  EMAIL_TAKEN: "email_taken",
  RATE_LIMITED: "rate_limited",
  INVALID_TOKEN: "invalid_token",
  UNAUTHORIZED: "unauthorized",

  // Cart / checkout
  UNAVAILABLE: "unavailable",
  EMPTY_CART: "empty_cart",
  DUPLICATE_ORDER: "duplicate_order",
  INSUFFICIENT_POINTS: "insufficient_points",
  PAYMENT_INIT_FAILED: "payment_init_failed",

  // General
  VALIDATION: "validation",
  NOT_FOUND: "not_found",
  FORBIDDEN: "forbidden",
  INVALID_TRANSITION: "invalid_transition",
  DUPLICATE: "duplicate",
} as const;

// Helper type — extracts the union of all code string values
export type ApiErrorCode = (typeof API_ERROR_CODES)[keyof typeof API_ERROR_CODES];