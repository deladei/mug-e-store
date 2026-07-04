// src/app/(admin)/admin/reports/page.tsx

"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  AreaChart,
  Area,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";
import { TrendingUp, ShoppingBag, DollarSign } from "lucide-react";
import { reportsService } from "@/services/reports.service";
import { Card, CardBody, CardHeader } from "@/components/ui/Card";
import { Skeleton } from "@/components/ui/Skeleton";
import { ProtectedRoute } from "@/components/layout/ProtectedRoute";
import { formatMoney, pesewasToFloat } from "@/utils";
import { cn } from "@/utils";

// ── Days filter ────────────────────────────────────────────────────────────────

const DAY_OPTIONS = [
  { label: "7 days", value: 7 },
  { label: "30 days", value: 30 },
  { label: "90 days", value: 90 },
];

// ── Stat card ──────────────────────────────────────────────────────────────────

function StatCard({
  label,
  value,
  sub,
  icon,
  accent,
}: {
  label: string;
  value: string;
  sub?: string;
  icon: React.ReactNode;
  accent: string;
}) {
  return (
    <Card>
      <CardBody className="space-y-3">
        <div className="flex items-center justify-between">
          <p className="text-sm text-stone-500 dark:text-stone-400">{label}</p>
          <div
            className={cn(
              "w-8 h-8 rounded-xl flex items-center justify-center",
              accent
            )}
          >
            {icon}
          </div>
        </div>
        <div>
          <p className="text-2xl font-bold text-stone-900 dark:text-stone-100">
            {value}
          </p>
          {sub && (
            <p className="text-xs text-stone-400 dark:text-stone-500 mt-0.5">
              {sub}
            </p>
          )}
        </div>
      </CardBody>
    </Card>
  );
}

// ── Custom tooltip ─────────────────────────────────────────────────────────────

function RevenueTooltip({
  active,
  payload,
  label,
}: {
  active?: boolean;
  payload?: Array<{ value: number }>;
  label?: string;
}) {
  if (!active || !payload?.length) return null;
  return (
    <div className="bg-white dark:bg-stone-900 border border-stone-200 dark:border-stone-700 rounded-xl px-3 py-2 shadow-lg">
      <p className="text-xs text-stone-400 dark:text-stone-500 mb-1">
        {label}
      </p>
      <p className="text-sm font-bold text-stone-900 dark:text-stone-100">
        {formatMoney(payload[0].value)}
      </p>
    </div>
  );
}

function OrdersTooltip({
  active,
  payload,
  label,
}: {
  active?: boolean;
  payload?: Array<{ value: number; name: string }>;
  label?: string;
}) {
  if (!active || !payload?.length) return null;
  return (
    <div className="bg-white dark:bg-stone-900 border border-stone-200 dark:border-stone-700 rounded-xl px-3 py-2 shadow-lg">
      <p className="text-xs text-stone-400 dark:text-stone-500 mb-1">
        {label}
      </p>
      {payload.map((p, i) => (
        <p key={i} className="text-sm font-semibold text-stone-800 dark:text-stone-200">
          {p.name}: {p.value}
        </p>
      ))}
    </div>
  );
}

// ── Reports content ────────────────────────────────────────────────────────────

function ReportsContent() {
  const [days, setDays] = useState(30);

  const { data, isLoading } = useQuery({
    queryKey: ["reports", days],
    queryFn: () => reportsService.getSummary(days),
  });

  // Format daily data for recharts
  const dailyData = data?.daily.map((d) => ({
    date: new Date(d.date).toLocaleDateString("en-GH", {
      month: "short",
      day: "numeric",
    }),
    revenue: d.revenue_pesewas,
    orders: d.orders,
    paid_orders: d.paid_orders,
  })) ?? [];

  // Only show every Nth label to avoid crowding
  const tickInterval = days === 7 ? 0 : days === 30 ? 4 : 13;

  return (
    <div className="space-y-6 pb-8">
      {/* Header */}
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div>
          <h1 className="text-xl font-bold text-stone-900 dark:text-stone-100">
            Reports
          </h1>
          {data && (
            <p className="text-sm text-stone-500 dark:text-stone-400">
              {new Date(data.from).toLocaleDateString("en-GH", {
                month: "short",
                day: "numeric",
              })}
              {" — "}
              {new Date(data.to).toLocaleDateString("en-GH", {
                month: "short",
                day: "numeric",
                year: "numeric",
              })}
            </p>
          )}
        </div>

        {/* Days filter */}
        <div className="flex bg-stone-200 dark:bg-stone-800 rounded-xl p-1 gap-1">
          {DAY_OPTIONS.map((opt) => (
            <button
              key={opt.value}
              onClick={() => setDays(opt.value)}
              className={cn(
                "px-3 h-7 rounded-lg text-sm font-medium transition-all duration-150",
                days === opt.value
                  ? "bg-white dark:bg-stone-900 text-stone-900 dark:text-stone-100 shadow-sm"
                  : "text-stone-500 dark:text-stone-400 hover:text-stone-700 dark:hover:text-stone-300"
              )}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>

      {/* Stat cards */}
      {isLoading ? (
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-28 w-full rounded-2xl" />
          ))}
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <StatCard
            label="Total revenue"
            value={formatMoney(data?.totals.revenue_pesewas ?? 0)}
            sub="From confirmed orders only"
            icon={
              <DollarSign
                size={16}
                className="text-emerald-600 dark:text-emerald-400"
              />
            }
            accent="bg-emerald-50 dark:bg-emerald-950/30"
          />
          <StatCard
            label="Paid orders"
            value={String(data?.totals.paid_orders ?? 0)}
            sub={`${data?.totals.orders ?? 0} total placed`}
            icon={
              <ShoppingBag
                size={16}
                className="text-amber-600 dark:text-amber-400"
              />
            }
            accent="bg-amber-50 dark:bg-amber-950/30"
          />
          <StatCard
            label="Avg order value"
            value={
              data && data.totals.paid_orders > 0
                ? formatMoney(
                    Math.round(
                      data.totals.revenue_pesewas / data.totals.paid_orders
                    )
                  )
                : "GHS 0.00"
            }
            sub="Per confirmed order"
            icon={
              <TrendingUp
                size={16}
                className="text-blue-600 dark:text-blue-400"
              />
            }
            accent="bg-blue-50 dark:bg-blue-950/30"
          />
        </div>
      )}

      {/* Revenue chart */}
      <Card>
        <CardHeader>
          <p className="text-sm font-semibold text-stone-700 dark:text-stone-300">
            Daily Revenue (GHS)
          </p>
        </CardHeader>
        <CardBody>
          {isLoading ? (
            <Skeleton className="h-56 w-full" />
          ) : (
            <ResponsiveContainer width="100%" height={220}>
              <AreaChart
                data={dailyData}
                margin={{ top: 4, right: 4, left: 0, bottom: 0 }}
              >
                <defs>
                  <linearGradient
                    id="revenueGradient"
                    x1="0"
                    y1="0"
                    x2="0"
                    y2="1"
                  >
                    <stop
                      offset="5%"
                      stopColor="#92400e"
                      stopOpacity={0.3}
                    />
                    <stop
                      offset="95%"
                      stopColor="#92400e"
                      stopOpacity={0}
                    />
                  </linearGradient>
                </defs>
                <CartesianGrid
                  strokeDasharray="3 3"
                  stroke="currentColor"
                  className="text-stone-100 dark:text-stone-800"
                />
                <XAxis
                  dataKey="date"
                  tick={{
                    fontSize: 11,
                    fill: "currentColor",
                    className: "text-stone-400",
                  }}
                  interval={tickInterval}
                  axisLine={false}
                  tickLine={false}
                />
                <YAxis
                  tickFormatter={(v) =>
                    `${pesewasToFloat(v).toFixed(0)}`
                  }
                  tick={{
                    fontSize: 11,
                    fill: "currentColor",
                    className: "text-stone-400",
                  }}
                  axisLine={false}
                  tickLine={false}
                  width={45}
                />
                <Tooltip content={<RevenueTooltip />} />
                <Area
                  type="monotone"
                  dataKey="revenue"
                  stroke="#92400e"
                  strokeWidth={2}
                  fill="url(#revenueGradient)"
                  dot={false}
                  activeDot={{ r: 4, fill: "#92400e" }}
                />
              </AreaChart>
            </ResponsiveContainer>
          )}
        </CardBody>
      </Card>

      {/* Orders chart */}
      <Card>
        <CardHeader>
          <p className="text-sm font-semibold text-stone-700 dark:text-stone-300">
            Daily Orders
          </p>
        </CardHeader>
        <CardBody>
          {isLoading ? (
            <Skeleton className="h-56 w-full" />
          ) : (
            <ResponsiveContainer width="100%" height={220}>
              <BarChart
                data={dailyData}
                margin={{ top: 4, right: 4, left: 0, bottom: 0 }}
              >
                <CartesianGrid
                  strokeDasharray="3 3"
                  stroke="currentColor"
                  className="text-stone-100 dark:text-stone-800"
                />
                <XAxis
                  dataKey="date"
                  tick={{
                    fontSize: 11,
                    fill: "currentColor",
                    className: "text-stone-400",
                  }}
                  interval={tickInterval}
                  axisLine={false}
                  tickLine={false}
                />
                <YAxis
                  tick={{
                    fontSize: 11,
                    fill: "currentColor",
                    className: "text-stone-400",
                  }}
                  axisLine={false}
                  tickLine={false}
                  width={30}
                  allowDecimals={false}
                />
                <Tooltip content={<OrdersTooltip />} />
                <Bar
                  dataKey="orders"
                  name="Total"
                  fill="#d6d3d1"
                  radius={[4, 4, 0, 0]}
                />
                <Bar
                  dataKey="paid_orders"
                  name="Paid"
                  fill="#92400e"
                  radius={[4, 4, 0, 0]}
                />
              </BarChart>
            </ResponsiveContainer>
          )}
        </CardBody>
      </Card>

      {/* Legend */}
      <div className="flex items-center gap-4 px-1">
        <div className="flex items-center gap-1.5">
          <div className="w-3 h-3 rounded-sm bg-stone-300 dark:bg-stone-600" />
          <span className="text-xs text-stone-500 dark:text-stone-400">
            Total orders
          </span>
        </div>
        <div className="flex items-center gap-1.5">
          <div className="w-3 h-3 rounded-sm bg-amber-800" />
          <span className="text-xs text-stone-500 dark:text-stone-400">
            Paid orders
          </span>
        </div>
        <div className="text-xs text-stone-400 dark:text-stone-500 ml-auto">
          Pending & cancelled orders excluded from revenue
        </div>
      </div>
    </div>
  );
}

export default function AdminReportsPage() {
  return (
    <ProtectedRoute requiredRole="admin">
      <ReportsContent />
    </ProtectedRoute>
  );
}