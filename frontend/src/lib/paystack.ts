// src/lib/paystack.ts
//
// Loads Paystack's inline v2 script on demand and opens the on-page payment
// popup. We use resumeTransaction(access_code): the transaction is already
// initialized server-side (with the authoritative amount + currency), so the
// browser never sends anything money-related — it only resumes a transaction
// the backend created. The order still only becomes `paid` when Paystack's
// webhook reaches the backend, never from the onSuccess callback here.

const INLINE_SRC = "https://js.paystack.co/v2/inline.js";

// Minimal shape of the pieces of the Paystack inline SDK we use.
interface PaystackTransactionOptions {
  onSuccess?: (transaction: { reference: string }) => void;
  onCancel?: () => void;
  onError?: (error: unknown) => void;
}

interface PaystackPopInstance {
  resumeTransaction: (
    accessCode: string,
    options?: PaystackTransactionOptions
  ) => void;
}

type PaystackPopConstructor = new () => PaystackPopInstance;

declare global {
  interface Window {
    PaystackPop?: PaystackPopConstructor;
  }
}

let loader: Promise<PaystackPopConstructor> | null = null;

// loadPaystack injects the inline script once and resolves with the PaystackPop
// constructor. Concurrent callers share a single in-flight load.
export function loadPaystack(): Promise<PaystackPopConstructor> {
  if (typeof window === "undefined") {
    return Promise.reject(new Error("Paystack can only load in the browser"));
  }
  if (window.PaystackPop) {
    return Promise.resolve(window.PaystackPop);
  }
  if (loader) return loader;

  loader = new Promise<PaystackPopConstructor>((resolve, reject) => {
    const existing = document.querySelector<HTMLScriptElement>(
      `script[src="${INLINE_SRC}"]`
    );
    const script = existing ?? document.createElement("script");

    // On any failure, reset `loader` so a later call can retry, and remove the
    // dead script tag — its load/error events have already fired, so a retry
    // that found it as `existing` would attach listeners that never run and
    // hang forever.
    const fail = (message: string) => {
      loader = null;
      script.remove();
      reject(new Error(message));
    };
    const onReady = () => {
      if (window.PaystackPop) resolve(window.PaystackPop);
      else fail("Paystack inline script loaded without PaystackPop");
    };
    const onFail = () => fail("Failed to load the Paystack inline script");

    script.addEventListener("load", onReady, { once: true });
    script.addEventListener("error", onFail, { once: true });

    if (!existing) {
      script.src = INLINE_SRC;
      script.async = true;
      document.head.appendChild(script);
    }
  });

  return loader;
}

// openPaystackPopup resumes a server-initialized transaction in an on-page
// popup. Rejects if the SDK cannot load so the caller can fall back to the
// hosted authorization_url.
export async function openPaystackPopup(
  accessCode: string,
  options: PaystackTransactionOptions
): Promise<void> {
  const PaystackPop = await loadPaystack();
  const popup = new PaystackPop();
  popup.resumeTransaction(accessCode, options);
}

// ── Payment retry stash ────────────────────────────────────────────────────
//
// Cancelling the inline popup leaves the order unpaid with no way back to the
// popup once the customer leaves checkout. Checkout stashes the codes per
// order in sessionStorage so the order page can offer "Pay now" for the rest
// of this browser session.

export interface PaymentRetry {
  access_code: string;
  authorization_url: string;
}

const retryKey = (orderId: string) => `paystack_retry_${orderId}`;

export function stashPaymentRetry(orderId: string, retry: PaymentRetry): void {
  try {
    sessionStorage.setItem(retryKey(orderId), JSON.stringify(retry));
  } catch {
    // Storage unavailable — the retry button just won't be offered.
  }
}

export function getPaymentRetry(orderId: string): PaymentRetry | null {
  try {
    const raw = sessionStorage.getItem(retryKey(orderId));
    return raw ? (JSON.parse(raw) as PaymentRetry) : null;
  } catch {
    return null;
  }
}

export function clearPaymentRetry(orderId: string): void {
  try {
    sessionStorage.removeItem(retryKey(orderId));
  } catch {
    // Nothing to do — worst case the stale entry dies with the session.
  }
}
