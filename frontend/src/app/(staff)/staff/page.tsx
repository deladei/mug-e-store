// src/app/(staff)/staff/page.tsx

"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { RefreshCw } from "lucide-react";
import { Eye, EyeOff } from "lucide-react";
import { useAuth } from "@/contexts/AuthContext";
import { staffService } from "@/services/staff.service";
import { OrderQueueBoard } from "@/features/staff/OrderQueueBoard";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { ProtectedRoute } from "@/components/layout/ProtectedRoute";
import { toast } from "@/hooks/useToast";
import { API_ERROR_CODES } from "@/types";
import {
  loginSchema,
  LoginFormValues,
} from "@/lib/validators/auth.schema";
import { cn } from "@/utils";
import axios from "axios";
import { useState as useStateAlias } from "react";
// ── Staff login form ───────────────────────────────────────────────────────────

function StaffLoginForm() {
  const { login } = useAuth();
  const router = useRouter();
  const [showPassword, setShowPassword] = useState(false);

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<LoginFormValues>({ resolver: zodResolver(loginSchema) });

  const onSubmit = async (values: LoginFormValues) => {
    try {
      await login(values);
    } catch (err) {
      if (axios.isAxiosError(err)) {
        const code = err.response?.data?.error?.code;
        if (code === API_ERROR_CODES.INVALID_CREDENTIALS) {
          setError("email", { message: "Invalid email or password" });
          setError("password", { message: "Invalid email or password" });
        } else if (code === API_ERROR_CODES.RATE_LIMITED) {
          toast.error("Too many attempts — wait a moment");
        } else {
          toast.error("Something went wrong");
        }
      }
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-stone-100 dark:bg-stone-950 px-4">
      <div className="w-full max-w-sm space-y-6">
        <div className="text-center space-y-1">
          <span className="text-4xl">☕</span>
          <h1 className="text-2xl font-bold text-stone-900 dark:text-stone-100">
            Staff Login
          </h1>
          <p className="text-sm text-stone-500 dark:text-stone-400">
            Sign in to access the dashboard
          </p>
        </div>

        <div className="bg-white dark:bg-stone-900 rounded-2xl border border-stone-200 dark:border-stone-800 p-6 shadow-sm">
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <Input
              label="Email"
              type="email"
              placeholder="staff@coffeemug.com"
              error={errors.email?.message}
              {...register("email")}
            />
            <Input
              label="Password"
              type={showPassword ? "text" : "password"}
              placeholder="••••••••"
              error={errors.password?.message}
              rightIcon={
                <button
                  type="button"
                  onClick={() => setShowPassword((s) => !s)}
                  className="text-stone-400 hover:text-stone-600"
                >
                  {showPassword ? <EyeOff size={16} /> : <Eye size={16} />}
                </button>
              }
              {...register("password")}
            />
            <Button
              type="submit"
              fullWidth
              size="lg"
              isLoading={isSubmitting}
            >
              Sign in
            </Button>
          </form>
        </div>
      </div>
    </div>
  );
}

// ── Order queue ────────────────────────────────────────────────────────────────

function StaffQueueContent() {
  const { data: orders = [], isLoading, refetch, isFetching } = useQuery({
    queryKey: ["staff-orders"],
    queryFn: () => staffService.getOrders(),
    // Auto-refresh every 30 seconds
    refetchInterval: 30_000,
  });

  return (
    <div className="space-y-5">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold text-stone-900 dark:text-stone-100">
            Order Queue
          </h1>
          <p className="text-sm text-stone-500 dark:text-stone-400">
            Refreshes every 30 seconds
          </p>
        </div>

        <button
          onClick={() => refetch()}
          disabled={isFetching}
          className={cn(
            "flex items-center gap-1.5 px-3 h-8 rounded-lg text-sm",
            "text-stone-500 hover:text-stone-800 dark:hover:text-stone-200",
            "hover:bg-stone-200 dark:hover:bg-stone-800",
            "transition-colors disabled:opacity-50"
          )}
        >
          <RefreshCw
            size={14}
            className={isFetching ? "animate-spin" : ""}
          />
          Refresh
        </button>
      </div>

      <OrderQueueBoard
        orders={orders}
        isLoading={isLoading}
        onAdvanced={() => refetch()}
      />
    </div>
  );
}

// ── Page — shows login if not staff, queue if authenticated ───────────────────

export default function StaffPage() {
  const { user, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="w-8 h-8 border-2 border-amber-600 border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  // Not logged in or wrong role — show login
  if (!user || (user.role !== "staff" && user.role !== "admin")) {
    return <StaffLoginForm />;
  }

  // Authenticated staff/admin — show queue
  return (
    <ProtectedRoute requiredRole="staff">
      <StaffQueueContent />
    </ProtectedRoute>
  );
}