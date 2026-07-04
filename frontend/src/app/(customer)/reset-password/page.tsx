// src/app/(customer)/reset-password/page.tsx

"use client";

import { useSearchParams, useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Eye, EyeOff } from "lucide-react";
import { useState } from "react";
import { authService } from "@/services/auth.service";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { toast } from "@/hooks/useToast";
import { API_ERROR_CODES } from "@/types";
import {
  passwordResetConfirmSchema,
  PasswordResetConfirmValues,
} from "@/lib/validators/auth.schema";
import { ROUTES } from "@/constants/routes";
import axios from "axios";

export default function ResetPasswordPage() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const token = searchParams.get("token") ?? "";
  const [showPassword, setShowPassword] = useState(false);

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<PasswordResetConfirmValues>({
    resolver: zodResolver(passwordResetConfirmSchema),
  });

  const onSubmit = async (values: PasswordResetConfirmValues) => {
    try {
      await authService.confirmPasswordReset({
        token,
        password: values.password,
      });
      toast.success("Password reset — please sign in");
      router.push(ROUTES.AUTH);
    } catch (err) {
      if (axios.isAxiosError(err)) {
        const code = err.response?.data?.error?.code;
        if (code === API_ERROR_CODES.INVALID_TOKEN) {
          setError("password", {
            message:
              "This reset link is invalid or has expired — request a new one",
          });
        } else {
          toast.error("Something went wrong — please try again");
        }
      }
    }
  };

  if (!token) {
    return (
      <div className="min-h-[70vh] flex items-center justify-center text-center px-4">
        <div className="space-y-4">
          <p className="text-stone-600 dark:text-stone-400">
            Invalid reset link. Please request a new one.
          </p>
          <Button onClick={() => router.push(ROUTES.AUTH)}>
            Back to sign in
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-[70vh] flex items-center justify-center px-4">
      <div className="w-full max-w-sm space-y-6">
        <div className="text-center space-y-1">
          <span className="text-4xl">🔑</span>
          <h1 className="text-2xl font-bold text-stone-900 dark:text-stone-100">
            New password
          </h1>
          <p className="text-sm text-stone-500 dark:text-stone-400">
            Choose a new password for your account
          </p>
        </div>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <Input
            label="New password"
            type={showPassword ? "text" : "password"}
            placeholder="Min. 8 characters"
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
          <Button type="submit" fullWidth size="lg" isLoading={isSubmitting}>
            Reset password
          </Button>
        </form>
      </div>
    </div>
  );
}