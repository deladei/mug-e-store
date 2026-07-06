// src/app/(customer)/profile/page.tsx

"use client";

import { useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { motion } from "framer-motion";
import { LogOut, User, Phone, Mail, Star, ChevronRight } from "lucide-react";
import { useAuth } from "@/contexts/AuthContext";
import { loyaltyService } from "@/services/loyalty.service";
import { Card, CardBody } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { Skeleton } from "@/components/ui/Skeleton";
import { ProtectedRoute } from "@/components/layout/ProtectedRoute";
import { formatMoney } from "@/utils";
import { ROUTES } from "@/constants/routes";
import { LedgerReason } from "@/types";
import { cn } from "@/utils";

// ── Ledger reason label ────────────────────────────────────────────────────────

const LEDGER_LABELS: Record<LedgerReason, string> = {
  earn_on_completion: "Earned",
  redeem_at_checkout: "Redeemed",
  refund_on_cancel: "Refunded",
};

const LEDGER_COLORS: Record<LedgerReason, string> = {
  earn_on_completion: "text-emerald-600 dark:text-emerald-400",
  redeem_at_checkout: "text-red-500 dark:text-red-400",
  refund_on_cancel: "text-blue-500 dark:text-blue-400",
};

// ── Info row ──────────────────────────────────────────────────────────────────

function InfoRow({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
}) {
  return (
    <div className="flex items-center gap-3 py-3 border-b border-stone-100 dark:border-stone-800 last:border-0">
      <span className="text-stone-400 dark:text-stone-500 shrink-0">
        {icon}
      </span>
      <div className="flex-1 min-w-0">
        <p className="text-xs text-stone-400 dark:text-stone-500">{label}</p>
        <p className="text-sm font-medium text-stone-800 dark:text-stone-200 truncate">
          {value}
        </p>
      </div>
    </div>
  );
}

// ── Profile content ───────────────────────────────────────────────────────────

function ProfileContent() {
  const router = useRouter();
  const { user, logout } = useAuth();

  const { data: loyalty, isLoading: loyaltyLoading } = useQuery({
    queryKey: ["loyalty"],
    queryFn: loyaltyService.getBalance,
  });

  const handleLogout = async () => {
    await logout();
  };

  if (!user) return null;

  // Points worth in cedis: 1 point = 1 pesewa = GHS 0.01
  const pointsValue = loyalty ? formatMoney(loyalty.balance) : null;

  return (
    <div className="space-y-5 pb-8">
      {/* Header */}
      <div className="flex items-center gap-4 py-2">
        <div className="w-16 h-16 rounded-full bg-amber-100 dark:bg-amber-950/40 flex items-center justify-center shrink-0">
          <span className="text-2xl font-bold text-amber-700 dark:text-amber-400">
            {user.name.charAt(0).toUpperCase()}
          </span>
        </div>
        <div className="min-w-0">
          <h1 className="text-xl font-bold text-stone-900 dark:text-stone-100 truncate">
            {user.name}
          </h1>
          <p className="text-sm text-stone-500 dark:text-stone-400 capitalize">
            {user.role}
          </p>
        </div>
      </div>

      {/* Contact info */}
      <Card>
        <CardBody className="py-0">
          <InfoRow
            icon={<Mail size={16} />}
            label="Email"
            value={user.email}
          />
          <InfoRow
            icon={<Phone size={16} />}
            label="Phone"
            value={user.phone}
          />
          <InfoRow
            icon={<User size={16} />}
            label="Member since"
            value={new Date(user.created_at).toLocaleDateString("en-GH", {
              month: "long",
              year: "numeric",
            })}
          />
        </CardBody>
      </Card>

      {/* Loyalty balance */}
      <Card>
        <CardBody className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Star
                size={16}
                className="text-amber-500 fill-amber-500"
              />
              <p className="text-sm font-semibold text-stone-700 dark:text-stone-300">
                Loyalty Points
              </p>
            </div>
            {loyaltyLoading ? (
              <Skeleton className="h-6 w-20" />
            ) : (
              <div className="text-right">
                <p className="text-lg font-bold text-amber-700 dark:text-amber-400">
                  {loyalty?.balance ?? 0} pts
                </p>
                {pointsValue && (
                  <p className="text-xs text-stone-400 dark:text-stone-500">
                    ≈ {pointsValue}
                  </p>
                )}
              </div>
            )}
          </div>

          {/* Ledger */}
          {loyaltyLoading ? (
            <div className="space-y-2">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-10 w-full" />
              ))}
            </div>
          ) : loyalty && loyalty.ledger.length > 0 ? (
            <div className="space-y-0 border-t border-stone-100 dark:border-stone-800 pt-3">
              <p className="text-xs font-medium text-stone-400 dark:text-stone-500 mb-2">
                Recent activity
              </p>
              {loyalty.ledger.map((entry, i) => (
                <motion.div
                  key={i}
                  initial={{ opacity: 0, x: -8 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ delay: i * 0.05 }}
                  className="flex items-center justify-between py-2 border-b border-stone-50 dark:border-stone-800/50 last:border-0"
                >
                  <div className="flex-1 min-w-0">
                    <p className="text-xs font-medium text-stone-600 dark:text-stone-400">
                      {LEDGER_LABELS[entry.reason]}
                    </p>
                    <p className="text-[10px] text-stone-400 dark:text-stone-500 font-mono">
                      #{String(entry.order_id).slice(-8).toUpperCase()}
                    </p>
                  </div>
                  <p
                    className={cn(
                      "text-sm font-semibold",
                      LEDGER_COLORS[entry.reason]
                    )}
                  >
                    {entry.delta > 0 ? "+" : ""}
                    {entry.delta} pts
                  </p>
                </motion.div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-stone-400 dark:text-stone-500 text-center py-2">
              No loyalty activity yet
            </p>
          )}
        </CardBody>
      </Card>

      {/* Quick links */}
      <Card>
        <CardBody className="py-0">
          <button
            onClick={() => router.push(ROUTES.ORDER_HISTORY)}
            className="w-full flex items-center justify-between py-3.5 border-b border-stone-100 dark:border-stone-800 text-sm font-medium text-stone-700 dark:text-stone-300 hover:text-stone-900 dark:hover:text-stone-100 transition-colors"
          >
            Order history
            <ChevronRight
              size={16}
              className="text-stone-400 dark:text-stone-500"
            />
          </button>
          <button
            onClick={() => router.push(ROUTES.AUTH)}
            className="w-full flex items-center justify-between py-3.5 text-sm font-medium text-stone-700 dark:text-stone-300 hover:text-stone-900 dark:hover:text-stone-100 transition-colors"
          >
            Change password
            <ChevronRight
              size={16}
              className="text-stone-400 dark:text-stone-500"
            />
          </button>
        </CardBody>
      </Card>

      {/* Logout */}
      <Button
        variant="outline"
        fullWidth
        onClick={handleLogout}
        className="gap-2 text-red-600 dark:text-red-400 border-red-200 dark:border-red-800 hover:bg-red-50 dark:hover:bg-red-950/30"
      >
        <LogOut size={16} />
        Sign out
      </Button>

      {/* Role badge for staff/admin */}
      {user.role !== "customer" && (
        <div className="text-center">
          <button
            onClick={() =>
              router.push(
                user.role === "admin" ? ROUTES.ADMIN_MENU : ROUTES.STAFF_QUEUE
              )
            }
            className="text-sm text-amber-700 dark:text-amber-500 hover:underline font-medium"
          >
            Go to {user.role} dashboard →
          </button>
        </div>
      )}
    </div>
  );
}

export default function ProfilePage() {
  return (
    <ProtectedRoute>
      <ProfileContent />
    </ProtectedRoute>
  );
}