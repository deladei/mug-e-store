// src/app/(customer)/layout.tsx

import { CustomerNav } from "@/components/layout/CustomerNav";
import { RequireAuth } from "@/components/layout/RequireAuth";
import { CartDrawer } from "@/components/cart/CartDrawer";

export default function CustomerLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="min-h-screen bg-stone-50 dark:bg-stone-950">
      <CustomerNav />
      <main className="max-w-2xl mx-auto px-4 py-6">
        <RequireAuth>{children}</RequireAuth>
      </main>
      <CartDrawer />
    </div>
  );
}
