// src/app/(admin)/layout.tsx

import { StaffNav } from "@/components/layout/StaffNav";

export default function AdminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="min-h-screen bg-stone-100 dark:bg-stone-950">
      <StaffNav />
      <main className="max-w-5xl mx-auto px-6 py-6">{children}</main>
    </div>
  );
}
