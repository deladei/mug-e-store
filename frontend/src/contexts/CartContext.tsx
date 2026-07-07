// src/contexts/CartContext.tsx

"use client";

import {
  createContext,
  useContext,
  useState,
  useCallback,
  ReactNode,
} from "react";
import { Cart, AddToCartPayload } from "@/types";
import { cartService } from "@/services/cart.service";
import { useAuth } from "./AuthContext";

interface CartContextValue {
  cart: Cart | null;
  isLoading: boolean;
  totalItems: number;
  // Drawer visibility — the cart is a right-side slide-in panel, not a page.
  isOpen: boolean;
  openCart: () => void;
  closeCart: () => void;
  fetchCart: () => Promise<void>;
  addItem: (payload: AddToCartPayload) => Promise<void>;
  updateLine: (lineId: string, quantity: number) => Promise<void>;
  removeLine: (lineId: string) => Promise<void>;
  clearLocalCart: () => void;
}

const CartContext = createContext<CartContextValue | null>(null);

export function CartProvider({ children }: { children: ReactNode }) {
  const [cart, setCart] = useState<Cart | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [isOpen, setIsOpen] = useState(false);
  const { isAuthenticated } = useAuth();

  const openCart = useCallback(() => setIsOpen(true), []);
  const closeCart = useCallback(() => setIsOpen(false), []);

  const fetchCart = useCallback(async () => {
    if (!isAuthenticated) return;
    setIsLoading(true);
    try {
      const data = await cartService.getCart();
      setCart(data);
    } catch {
      setCart(null);
    } finally {
      setIsLoading(false);
    }
  }, [isAuthenticated]);

  const addItem = useCallback(async (payload: AddToCartPayload) => {
    const data = await cartService.addItem(payload);
    // Every mutation returns the full updated cart — replace state from response
    setCart(data);
  }, []);

  const updateLine = useCallback(
    async (lineId: string, quantity: number) => {
      const data = await cartService.updateLine(lineId, { quantity });
      setCart(data);
    },
    []
  );

  const removeLine = useCallback(async (lineId: string) => {
    const data = await cartService.removeLine(lineId);
    setCart(data);
  }, []);

  const clearLocalCart = useCallback(() => {
    setCart(null);
  }, []);

  const totalItems =
    cart?.lines.reduce((sum, line) => sum + line.quantity, 0) ?? 0;

  return (
    <CartContext.Provider
      value={{
        cart,
        isLoading,
        totalItems,
        isOpen,
        openCart,
        closeCart,
        fetchCart,
        addItem,
        updateLine,
        removeLine,
        clearLocalCart,
      }}
    >
      {children}
    </CartContext.Provider>
  );
}

export function useCart(): CartContextValue {
  const context = useContext(CartContext);
  if (!context) {
    throw new Error("useCart must be used within a CartProvider");
  }
  return context;
}